// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IOracle {
    /// @notice Get the latest aggregated price and the height at which it was aggregated.
    /// @param feedID The identifier of the price feed (e.g. "BTC_USD").
    /// @return price The aggregated price.
    /// @return blockHeight The block height of aggregation.
    function getLatestPrice(string calldata feedID) external view returns (uint64 price, int64 blockHeight);

    /// @notice Check if the price feed is currently stale based on staleness threshold.
    /// @param feedID The identifier of the price feed.
    /// @return stale True if the feed is stale, false otherwise.
    function isFeedStale(string calldata feedID) external view returns (bool stale);
}
