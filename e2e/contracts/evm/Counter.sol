// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

/**
 * @title Counter
 * @dev Production-grade test contract for Sovereign L1 EVM verification.
 *      Implements read/write operations, events, access control, and constructor args
 *      to fully exercise the explorer's dynamic ABI form generation.
 */
contract Counter {
    // --- State ---
    uint256 private _count;
    address public owner;
    string  public label;
    bool    public paused;
    mapping(address => uint256) public incrementsByUser;

    // --- Events ---
    event Incremented(address indexed by, uint256 newValue);
    event Decremented(address indexed by, uint256 newValue);
    event Reset(address indexed by, uint256 oldValue);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);
    event Paused(address indexed by);
    event Unpaused(address indexed by);
    event LabelUpdated(string oldLabel, string newLabel);

    // --- Errors ---
    error NotOwner();
    error ContractPaused();
    error CounterUnderflow();

    // --- Modifiers ---
    modifier onlyOwner() {
        if (msg.sender != owner) revert NotOwner();
        _;
    }

    modifier whenNotPaused() {
        if (paused) revert ContractPaused();
        _;
    }

    /**
     * @dev Constructor with arguments (tests constructor_args verification).
     * @param _initialCount Starting counter value.
     * @param _label Human-readable label for the counter instance.
     */
    constructor(uint256 _initialCount, string memory _label) {
        _count = _initialCount;
        label  = _label;
        owner  = msg.sender;
    }

    // ========== READ FUNCTIONS ==========

    /// @notice Returns the current counter value.
    function count() external view returns (uint256) {
        return _count;
    }

    /// @notice Returns the number of increments performed by a specific user.
    function getIncrementsByUser(address user) external view returns (uint256) {
        return incrementsByUser[user];
    }

    /// @notice Returns a summary tuple for dashboard display.
    function summary() external view returns (
        uint256 currentCount,
        address currentOwner,
        string memory currentLabel,
        bool isPaused
    ) {
        return (_count, owner, label, paused);
    }

    // ========== WRITE FUNCTIONS ==========

    /// @notice Increment the counter by 1.
    function increment() external whenNotPaused {
        _count += 1;
        incrementsByUser[msg.sender] += 1;
        emit Incremented(msg.sender, _count);
    }

    /// @notice Increment the counter by an arbitrary amount.
    function incrementBy(uint256 amount) external whenNotPaused {
        _count += amount;
        incrementsByUser[msg.sender] += 1;
        emit Incremented(msg.sender, _count);
    }

    /// @notice Decrement the counter by 1.
    function decrement() external whenNotPaused {
        if (_count == 0) revert CounterUnderflow();
        _count -= 1;
        emit Decremented(msg.sender, _count);
    }

    /// @notice Reset the counter to 0 (owner only).
    function reset() external onlyOwner whenNotPaused {
        uint256 old = _count;
        _count = 0;
        emit Reset(msg.sender, old);
    }

    /// @notice Pause the contract (owner only).
    function pause() external onlyOwner {
        paused = true;
        emit Paused(msg.sender);
    }

    /// @notice Unpause the contract (owner only).
    function unpause() external onlyOwner {
        paused = false;
        emit Unpaused(msg.sender);
    }

    /// @notice Update the counter label (owner only).
    function setLabel(string calldata newLabel) external onlyOwner {
        string memory old = label;
        label = newLabel;
        emit LabelUpdated(old, newLabel);
    }

    /// @notice Transfer ownership to a new address (owner only).
    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "Zero address not allowed");
        address old = owner;
        owner = newOwner;
        emit OwnershipTransferred(old, newOwner);
    }
}
