// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract TestERC1155 {
    mapping(uint256 => mapping(address => uint256)) public balanceOf;
    mapping(address => mapping(address => bool)) public isApprovedForAll;

    event TransferSingle(address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value);
    event ApprovalForAll(address indexed account, address indexed operator, bool approved);

    constructor() {
        mint(msg.sender, 1, 100);
        mint(msg.sender, 99, 500);
    }

    function safeTransferFrom(address from, address to, uint256 id, uint256 value, bytes memory) public {
        require(msg.sender == from || isApprovedForAll[from][msg.sender], "Not authorized");
        require(balanceOf[id][from] >= value, "Insufficient balance");
        
        balanceOf[id][from] -= value;
        balanceOf[id][to] += value;
        
        emit TransferSingle(msg.sender, from, to, id, value);
    }

    function setApprovalForAll(address operator, bool approved) public {
        isApprovedForAll[msg.sender][operator] = approved;
        emit ApprovalForAll(msg.sender, operator, approved);
    }

    function mint(address to, uint256 id, uint256 value) public {
        balanceOf[id][to] += value;
        emit TransferSingle(msg.sender, address(0), to, id, value);
    }
}
