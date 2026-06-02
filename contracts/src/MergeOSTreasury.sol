// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

interface IMergeOSTreasuryToken {
    function transfer(address to, uint256 amount) external returns (bool);
}

/// @title MergeOS treasury
/// @notice Holds MRG reserves and releases approved payouts through trusted operators.
contract MergeOSTreasury {
    IMergeOSTreasuryToken public immutable token;
    address public owner;

    mapping(address => bool) public operators;

    event TreasuryRelease(address indexed recipient, uint256 amount, bytes32 indexed reference);
    event OperatorUpdated(address indexed account, bool enabled);
    event OwnershipTransferred(address indexed previousOwner, address indexed nextOwner);

    error NotOwner();
    error NotOperator();
    error ZeroAddress();
    error TokenTransferFailed();

    modifier onlyOwner() {
        if (msg.sender != owner) revert NotOwner();
        _;
    }

    modifier onlyOperator() {
        if (msg.sender != owner && !operators[msg.sender]) revert NotOperator();
        _;
    }

    constructor(address tokenAddress, address initialOwner) {
        if (tokenAddress == address(0) || initialOwner == address(0)) revert ZeroAddress();
        token = IMergeOSTreasuryToken(tokenAddress);
        owner = initialOwner;
        emit OwnershipTransferred(address(0), initialOwner);
    }

    function transferOwnership(address nextOwner) external onlyOwner {
        if (nextOwner == address(0)) revert ZeroAddress();
        emit OwnershipTransferred(owner, nextOwner);
        owner = nextOwner;
    }

    function setOperator(address account, bool enabled) external onlyOwner {
        if (account == address(0)) revert ZeroAddress();
        operators[account] = enabled;
        emit OperatorUpdated(account, enabled);
    }

    function release(address recipient, uint256 amount, bytes32 reference) external onlyOperator {
        if (recipient == address(0)) revert ZeroAddress();
        _safeTransfer(recipient, amount);
        emit TreasuryRelease(recipient, amount, reference);
    }

    function sweep(address recipient, uint256 amount, bytes32 reference) external onlyOwner {
        if (recipient == address(0)) revert ZeroAddress();
        _safeTransfer(recipient, amount);
        emit TreasuryRelease(recipient, amount, reference);
    }

    function _safeTransfer(address recipient, uint256 amount) private {
        bool ok = token.transfer(recipient, amount);
        if (!ok) revert TokenTransferFailed();
    }
}
