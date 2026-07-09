// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract TestERC721 {
    string public name = "Test ERC721";
    string public symbol = "TERC721";

    mapping(uint256 => address) public ownerOf;
    mapping(address => uint256) public balanceOf;
    mapping(uint256 => address) public getApproved;
    mapping(address => mapping(address => bool)) public isApprovedForAll;

    event Transfer(address indexed from, address indexed to, uint256 indexed tokenId);
    event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId);
    event ApprovalForAll(address indexed owner, address indexed operator, bool approved);

    uint256 private _nextTokenId;

    constructor() {
        mint(msg.sender);
    }

    function transferFrom(address from, address to, uint256 tokenId) public {
        require(ownerOf[tokenId] == from, "Not owner");
        require(to != address(0), "Zero address recipient");
        
        balanceOf[from] -= 1;
        balanceOf[to] += 1;
        ownerOf[tokenId] = to;

        emit Transfer(from, to, tokenId);
    }

    function safeTransferFrom(address from, address to, uint256 tokenId) public {
        transferFrom(from, to, tokenId);
    }

    function safeTransferFrom(address from, address to, uint256 tokenId, bytes memory) public {
        transferFrom(from, to, tokenId);
    }

    function approve(address to, uint256 tokenId) public {
        address owner = ownerOf[tokenId];
        require(msg.sender == owner || isApprovedForAll[owner][msg.sender], "Not authorized");
        getApproved[tokenId] = to;
        emit Approval(owner, to, tokenId);
    }

    function setApprovalForAll(address operator, bool approved) public {
        isApprovedForAll[msg.sender][operator] = approved;
        emit ApprovalForAll(msg.sender, operator, approved);
    }

    function mint(address to) public returns (uint256) {
        _nextTokenId += 1;
        uint256 tokenId = _nextTokenId;
        balanceOf[to] += 1;
        ownerOf[tokenId] = to;
        emit Transfer(address(0), to, tokenId);
        return tokenId;
    }
}
