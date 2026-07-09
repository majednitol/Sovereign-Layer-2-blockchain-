// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

interface IERC20 {
    function balanceOf(address) external view returns (uint256);
    function transferFrom(address, address, uint256) external returns (bool);
    function transfer(address, uint256) external returns (bool);
}

contract TestERC4626 {
    IERC20 public asset;
    uint256 public totalAssets;
    uint256 public totalSupply;

    mapping(address => uint256) public balanceOf;

    event Deposit(address indexed sender, address indexed owner, uint256 assets, uint256 shares);
    event Withdraw(address indexed sender, address indexed receiver, address indexed owner, uint256 assets, uint256 shares);

    constructor(address _asset) {
        asset = IERC20(_asset);
    }

    function deposit(uint256 assets, address receiver) public returns (uint256 shares) {
        shares = assets;
        balanceOf[receiver] += shares;
        totalSupply += shares;
        totalAssets += assets;

        asset.transferFrom(msg.sender, address(this), assets);
        emit Deposit(msg.sender, receiver, assets, shares);
        return shares;
    }

    function withdraw(uint256 assets, address receiver, address owner) public returns (uint256 shares) {
        shares = assets;
        require(balanceOf[owner] >= shares, "Insufficient balance");

        balanceOf[owner] -= shares;
        totalSupply -= shares;
        totalAssets -= assets;

        asset.transfer(receiver, assets);
        emit Withdraw(msg.sender, receiver, owner, assets, shares);
        return shares;
    }
}
