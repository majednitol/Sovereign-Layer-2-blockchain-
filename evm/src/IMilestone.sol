// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IMilestone {
    /// @notice Get the milestone configuration and current tracking state.
    /// @param id The identifier of the milestone.
    /// @return milestoneID The milestone identifier.
    /// @return feedID The oracle price feed ID.
    /// @return targetPrice The target price threshold.
    /// @return remainingBlocks The remaining blocks until expiry.
    /// @return state The current state ("pending", "stale-blocked", "achieved", "expired").
    /// @return vestingPoolAddress The account pool address for vesting disbursements.
    function getMilestone(string calldata id) external view returns (
        string memory milestoneID,
        string memory feedID,
        uint64 targetPrice,
        int64 remainingBlocks,
        string memory state,
        string memory vestingPoolAddress
    );
}
