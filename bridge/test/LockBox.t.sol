// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../src/LockBox.sol";
import "../src/MockERC20.sol";

contract LockBoxTest is Test {
    LockBox public lockBox;
    MockERC20 public token;

    // Relayers
    address[] public relayers;
    uint256[] public relayerKeys;
    uint256 public threshold = 3;

    // Identities
    address public circuitBreaker = address(0x1111111111111111111111111111111111111111);
    address public gnosisSafe = address(0x2222222222222222222222222222222222222222);
    address public user = address(0x3333333333333333333333333333333333333333);

    uint256 public maxUnlockPerBlock = 1000 * 10**18;

    function setUp() public {
        // Deploy token
        token = new MockERC20(1000000 * 10**18);

        // Set up relayers
        for (uint256 i = 1; i <= 5; i++) {
            relayers.push(vm.addr(i));
            relayerKeys.push(i);
        }

        // Deploy LockBox
        lockBox = new LockBox(
            address(token),
            relayers,
            threshold,
            circuitBreaker,
            gnosisSafe,
            maxUnlockPerBlock
        );

        // Mint and approve tokens for user
        token.mint(user, 10000 * 10**18);
        vm.prank(user);
        token.approve(address(lockBox), type(uint256).max);
    }

    // Helper to generate signatures for unlock payload
    function getSignatures(
        address targetUser,
        uint256 amount,
        uint256 nonce,
        uint256 count
    ) internal view returns (bytes[] memory) {
        bytes32 messageHash = keccak256(
            abi.encodePacked(
                "\x19Ethereum Signed Message:\n32",
                keccak256(abi.encodePacked(block.chainid, address(lockBox), targetUser, amount, nonce))
            )
        );

        bytes[] memory sigs = new bytes[](count);
        for (uint256 i = 0; i < count; i++) {
            (uint8 v, bytes32 r, bytes32 s) = vm.sign(relayerKeys[i], messageHash);
            sigs[i] = abi.encodePacked(r, s, v);
        }
        return sigs;
    }

    // 1. Test Lock Box Setup and Parameters
    function testSetup() public {
        assertEq(address(lockBox.token()), address(token));
        assertEq(lockBox.threshold(), threshold);
        assertEq(lockBox.circuitBreakerAddress(), circuitBreaker);
        assertEq(lockBox.gnosisSafeAddress(), gnosisSafe);
        assertEq(lockBox.maxUnlockPerBlock(), maxUnlockPerBlock);
    }

    // 2. Test Lock functionality
    function testLock() public {
        uint256 amount = 100 * 10**18;
        string memory cosmosRecipient = "cosmos1recipientaddr";

        vm.prank(user);
        lockBox.lock(amount, cosmosRecipient);

        assertEq(token.balanceOf(address(lockBox)), amount);
        assertEq(lockBox.userNonce(user), 1);
        assertEq(lockBox.totalLocked(), amount);
    }

    // 3. Test Unlock with Quorum Signatures
    function testUnlock() public {
        uint256 amount = 100 * 10**18;
        // Lock first to have tokens in contract
        vm.prank(user);
        lockBox.lock(amount, "cosmos_recipient");

        uint256 nonce = 99999; // Assume relayer provides nonce
        bytes[] memory sigs = getSignatures(user, amount, nonce, threshold);

        uint256 balanceBefore = token.balanceOf(user);

        lockBox.unlock(user, amount, nonce, sigs);

        assertEq(token.balanceOf(user), balanceBefore + amount);
        assertEq(lockBox.totalReleasedToUsers(), amount);
    }

    // 4. Test Double Spend (Replay of same nonce)
    function testUnlockReplayFails() public {
        uint256 amount = 100 * 10**18;
        vm.prank(user);
        lockBox.lock(amount * 2, "cosmos_recipient");

        uint256 nonce = 12345;
        bytes[] memory sigs = getSignatures(user, amount, nonce, threshold);

        lockBox.unlock(user, amount, nonce, sigs);

        // Replay should fail
        vm.expectRevert("nonce already processed");
        lockBox.unlock(user, amount, nonce, sigs);
    }

    // 5. Test Insufficient Unique Signers
    function testUnlockInsufficientSignaturesFails() public {
        uint256 amount = 100 * 10**18;
        vm.prank(user);
        lockBox.lock(amount, "cosmos_recipient");

        uint256 nonce = 55555;
        bytes[] memory sigs = getSignatures(user, amount, nonce, threshold - 1);

        vm.expectRevert("insufficient signatures");
        lockBox.unlock(user, amount, nonce, sigs);
    }

    // 6. Test Duplicate Relayer Signatures (Quorum Cheating)
    function testUnlockDuplicateSignaturesFails() public {
        uint256 amount = 100 * 10**18;
        vm.prank(user);
        lockBox.lock(amount, "cosmos_recipient");

        uint256 nonce = 77777;
        
        // Generate only 2 unique signatures but repeat one to make length >= threshold
        bytes[] memory sigs = new bytes[](3);
        bytes[] memory uniqueSigs = getSignatures(user, amount, nonce, 2);
        sigs[0] = uniqueSigs[0];
        sigs[1] = uniqueSigs[1];
        sigs[2] = uniqueSigs[0]; // Duplicate of 0

        vm.expectRevert("insufficient unique relayer signatures");
        lockBox.unlock(user, amount, nonce, sigs);
    }

    // 7. Test Rate Limiting
    function testRateLimiting() public {
        uint256 amount = 600 * 10**18; // > maxUnlockPerBlock / 2
        vm.prank(user);
        lockBox.lock(amount * 2, "cosmos_recipient");

        uint256 nonce1 = 11111;
        bytes[] memory sigs1 = getSignatures(user, amount, nonce1, threshold);
        lockBox.unlock(user, amount, nonce1, sigs1);

        // Second unlock in same block puts total at 1200, exceeding maxUnlockPerBlock (1000)
        uint256 nonce2 = 22222;
        bytes[] memory sigs2 = getSignatures(user, amount, nonce2, threshold);
        
        vm.expectRevert("rate limit exceeded");
        lockBox.unlock(user, amount, nonce2, sigs2);

        // Progress block number -> rate limit reset
        vm.roll(block.number + 1);
        lockBox.unlock(user, amount, nonce2, sigs2); // Should pass
    }

    // 8. Test Circuit Breaker Pause/Unpause
    function testCircuitBreaker() public {
        // EOA Circuit Breaker Pauses
        vm.prank(circuitBreaker);
        lockBox.pause();
        assertTrue(lockBox.paused());

        // Lock & Unlock should fail while paused
        vm.expectRevert("bridge is paused");
        vm.prank(user);
        lockBox.lock(100, "cosmos_recipient");

        bytes[] memory sigs = getSignatures(user, 100, 999, threshold);
        vm.expectRevert("bridge is paused");
        lockBox.unlock(user, 100, 999, sigs);

        // EOA cannot unpause
        vm.expectRevert("caller is not the Gnosis Safe");
        vm.prank(circuitBreaker);
        lockBox.unpause();

        // Gnosis Safe can unpause
        vm.prank(gnosisSafe);
        lockBox.unpause();
        assertFalse(lockBox.paused());
    }

    // 9. Test Circuit Breaker Rotation
    function testCircuitBreakerRotation() public {
        address newCB = address(0x9999);
        
        // EOA cannot rotate
        vm.expectRevert("caller is not the Gnosis Safe");
        vm.prank(circuitBreaker);
        lockBox.rotateCircuitBreaker(newCB);

        // Gnosis Safe rotates
        vm.prank(gnosisSafe);
        lockBox.rotateCircuitBreaker(newCB);
        assertEq(lockBox.circuitBreakerAddress(), newCB);

        // Old CB cannot pause anymore
        vm.expectRevert("unauthorized pause caller");
        vm.prank(circuitBreaker);
        lockBox.pause();

        // New CB can pause
        vm.prank(newCB);
        lockBox.pause();
        assertTrue(lockBox.paused());
    }

    // 10. Fuzz Test Lock and Unlock
    function testFuzzLockUnlock(uint256 amount, uint256 nonce) public {
        // Bound inputs to avoid overflow/underflow or division by zero in bitmaps
        amount = bound(amount, 1, 100 * 10**18);
        nonce = bound(nonce, 1, type(uint256).max - 1);

        vm.prank(user);
        lockBox.lock(amount, "cosmos_recipient");

        bytes[] memory sigs = getSignatures(user, amount, nonce, threshold);
        lockBox.unlock(user, amount, nonce, sigs);

        (uint256 totalLockedVal, uint256 totalReleasedVal, uint256 pendingVal) = lockBox.getInvariantState();
        assertEq(totalLockedVal, amount);
        assertEq(totalReleasedVal, amount);
        assertEq(pendingVal, 0);
    }

    // 11. Invariant: totalLocked == totalReleasedToUsers + totalPendingUnlock
    function testInvariantState() public {
        vm.prank(user);
        lockBox.lock(500 * 10**18, "cosmos_recipient");

        bytes[] memory sigs = getSignatures(user, 200 * 10**18, 123456, threshold);
        lockBox.unlock(user, 200 * 10**18, 123456, sigs);

        (uint256 totalLockedVal, uint256 totalReleasedVal, uint256 pendingVal) = lockBox.getInvariantState();
        assertEq(totalLockedVal, 500 * 10**18);
        assertEq(totalReleasedVal, 200 * 10**18);
        assertEq(pendingVal, 300 * 10**18);
        assertEq(totalLockedVal, totalReleasedVal + pendingVal);
    }
}
