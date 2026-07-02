// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Script.sol";
import "../src/LockBox.sol";
import "../src/MockERC20.sol";

contract DeployLockBox is Script {
    function run() external {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");

        // Parse configurations from environment
        address tokenAddress;
        try vm.envAddress("TOKEN_ADDRESS") returns (address addr) {
            tokenAddress = addr;
        } catch {
            tokenAddress = address(0);
        }

        address circuitBreaker = vm.envAddress("CIRCUIT_BREAKER");
        address gnosisSafe = vm.envAddress("GNOSIS_SAFE");
        uint256 threshold = vm.envUint("THRESHOLD");
        uint256 maxUnlockPerBlock = vm.envUint("MAX_UNLOCK_PER_BLOCK");

        address relayer1 = vm.envAddress("RELAYER_1");
        address relayer2 = vm.envAddress("RELAYER_2");
        address relayer3 = vm.envAddress("RELAYER_3");

        address[] memory relayers = new address[](3);
        relayers[0] = relayer1;
        relayers[1] = relayer2;
        relayers[2] = relayer3;

        vm.startBroadcast(deployerPrivateKey);

        // Deploy Mock ERC20 if TOKEN_ADDRESS is not provided (0x0)
        if (tokenAddress == address(0)) {
            MockERC20 mockToken = new MockERC20(1_000_000 * 1e18);
            tokenAddress = address(mockToken);
            console.log("Deployed MockERC20 at:", tokenAddress);
        } else {
            console.log("Using existing token at:", tokenAddress);
        }

        // Deploy LockBox
        LockBox lockBox = new LockBox(
            tokenAddress,
            relayers,
            threshold,
            circuitBreaker,
            gnosisSafe,
            maxUnlockPerBlock
        );

        console.log("Deployed LockBox at:", address(lockBox));

        vm.stopBroadcast();
    }
}
