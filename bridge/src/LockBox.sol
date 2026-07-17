// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

interface IERC20 {
    function transfer(address to, uint256 value) external returns (bool);
    function transferFrom(address from, address to, uint256 value) external returns (bool);
}

contract LockBox {
    IERC20 public immutable token;

    // Nonces
    mapping(address => uint256) public userNonce;
    // Bitmap nonce registry to support out-of-order execution while preventing replays
    mapping(uint256 => uint256) public nonceBitmap;

    // Relayer committee
    address[] public relayers;
    mapping(address => bool) public isRelayer;
    uint256 public immutable threshold;

    // Invariants tracking
    uint256 public totalLocked;
    uint256 public totalReleasedToUsers;

    // Circuit Breaker
    bool public paused;
    address public circuitBreakerAddress;
    address public gnosisSafeAddress;

    // Rate limiting
    uint256 public maxUnlockPerBlock;
    uint256 public currentBlockUnlockAmount;
    uint256 public lastUnlockBlock;

    // Events
    event Locked(address indexed user, uint256 amount, string cosmosRecipient, uint256 nonce);
    event Unlocked(address indexed user, uint256 amount, uint256 nonce);
    event Paused(address indexed caller);
    event Unpaused(address indexed caller);
    event CircuitBreakerRotated(address indexed oldCB, address indexed newCB);

    modifier onlyGnosisSafe() {
        require(msg.sender == gnosisSafeAddress, "caller is not the Gnosis Safe");
        _;
    }

    modifier whenNotPaused() {
        require(!paused, "bridge is paused");
        _;
    }

    constructor(
        address _token,
        address[] memory _relayers,
        uint256 _threshold,
        address _circuitBreaker,
        address _gnosisSafe,
        uint256 _maxUnlockPerBlock
    ) {
        require(_token != address(0), "invalid token address");
        require(_threshold > 0 && _threshold <= _relayers.length, "invalid threshold");
        require(_circuitBreaker != address(0), "invalid circuit breaker address");
        require(_gnosisSafe != address(0), "invalid Gnosis Safe address");

        token = IERC20(_token);
        threshold = _threshold;
        circuitBreakerAddress = _circuitBreaker;
        gnosisSafeAddress = _gnosisSafe;
        maxUnlockPerBlock = _maxUnlockPerBlock;

        for (uint256 i = 0; i < _relayers.length; i++) {
            address relayer = _relayers[i];
            require(relayer != address(0), "invalid relayer address");
            require(!isRelayer[relayer], "duplicate relayer address");
            isRelayer[relayer] = true;
            relayers.push(relayer);
        }
    }

    // lock locks tokens on BSC and generates a contract-unpredictable nonce.
    // Emits Locked(user, amount, cosmosRecipient, nonce)
    function lock(uint256 amount, string calldata cosmosRecipient) external whenNotPaused {
        require(amount > 0, "amount must be greater than zero");
        require(bytes(cosmosRecipient).length > 0, "cosmos recipient cannot be empty");

        // Generate contract-side unpredictable nonce
        uint256 nonce = uint256(
            keccak256(
                abi.encodePacked(
                    msg.sender,
                    userNonce[msg.sender]++,
                    block.number,
                    amount,
                    block.timestamp
                )
            )
        );

        require(token.transferFrom(msg.sender, address(this), amount), "token transfer failed");

        totalLocked += amount;

        emit Locked(msg.sender, amount, cosmosRecipient, nonce);
    }

    // unlock releases tokens to a BSC user based on a quorum proof from the relayer committee.
    function unlock(
        address user,
        uint256 amount,
        uint256 nonce,
        bytes[] calldata signatures
    ) external whenNotPaused {
        require(user != address(0), "invalid user address");
        require(amount > 0, "amount must be greater than zero");
        require(signatures.length >= threshold, "insufficient signatures");

        // Verify nonce has not been processed
        uint256 wordIndex = nonce / 256;
        uint256 bitIndex = nonce % 256;
        uint256 mask = 1 << bitIndex;
        require((nonceBitmap[wordIndex] & mask) == 0, "nonce already processed");

        // Enforce rate limiting per block
        if (block.number > lastUnlockBlock) {
            lastUnlockBlock = block.number;
            currentBlockUnlockAmount = amount;
        } else {
            currentBlockUnlockAmount += amount;
        }
        require(currentBlockUnlockAmount <= maxUnlockPerBlock, "rate limit exceeded");

        // Verify signatures and recover relayer committee quorum
        bytes32 messageHash = keccak256(
            abi.encodePacked(
                "\x19Ethereum Signed Message:\n32",
                keccak256(abi.encodePacked(block.chainid, address(this), user, amount, nonce))
            )
        );

        address[] memory recoveredSigners = new address[](signatures.length);
        uint256 uniqueCount = 0;

        for (uint256 i = 0; i < signatures.length; i++) {
            address recovered = recoverSigner(messageHash, signatures[i]);
            require(isRelayer[recovered], "recovered signer is not a registered relayer");
            
            // Check for duplicate signatures in submission to prevent double-counting
            bool duplicate = false;
            for (uint256 j = 0; j < uniqueCount; j++) {
                if (recoveredSigners[j] == recovered) {
                    duplicate = true;
                    break;
                }
            }
            if (!duplicate) {
                recoveredSigners[uniqueCount] = recovered;
                uniqueCount++;
            }
        }

        require(uniqueCount >= threshold, "insufficient unique relayer signatures");

        // Update nonce bitmap
        nonceBitmap[wordIndex] |= mask;
        totalReleasedToUsers += amount;

        require(token.transfer(user, amount), "token transfer failed");

        emit Unlocked(user, amount, nonce);
    }

    // Emergency Pause - callable by circuitBreakerAddress or Gnosis Safe (pause-only for EOA)
    function pause() external {
        require(
            msg.sender == circuitBreakerAddress || msg.sender == gnosisSafeAddress,
            "unauthorized pause caller"
        );
        paused = true;
        emit Paused(msg.sender);
    }

    // Unpause - restricted to Gnosis Safe
    function unpause() external onlyGnosisSafe {
        paused = false;
        emit Unpaused(msg.sender);
    }

    // Rotate circuit breaker address - restricted to Gnosis Safe
    function rotateCircuitBreaker(address newCB) external onlyGnosisSafe {
        require(newCB != address(0), "invalid circuit breaker address");
        address oldCB = circuitBreakerAddress;
        circuitBreakerAddress = newCB;
        emit CircuitBreakerRotated(oldCB, newCB);
    }

    // Helper method to recover ECDSA signer address
    // C-05: Require non-zero recovered address to prevent ecrecover zero address vulnerability
    function recoverSigner(bytes32 messageHash, bytes memory sig) public pure returns (address) {
        if (sig.length != 65) {
            return address(0);
        }
        bytes32 r;
        bytes32 s;
        uint8 v;
        assembly {
            r := mload(add(sig, 32))
            s := mload(add(sig, 64))
            v := byte(0, mload(add(sig, 96)))
        }
        address recovered = ecrecover(messageHash, v, r, s);
        require(recovered != address(0), "recovered address cannot be zero");
        return recovered;
    }

    // Invariant getter: totalLocked == totalReleasedToUsers + totalPendingUnlock
    function getInvariantState() external view returns (uint256 locked, uint256 released, uint256 pending) {
        return (totalLocked, totalReleasedToUsers, totalLocked - totalReleasedToUsers);
    }
}
