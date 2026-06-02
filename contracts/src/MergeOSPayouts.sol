// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

interface IMergeOSPayoutTreasury {
    function release(address recipient, uint256 amount, bytes32 reference) external;
}

/// @title MergeOS payout approvals
/// @notice Records approved bounty payouts and executes each reference once through the treasury.
contract MergeOSPayouts {
    enum PayoutStatus {
        None,
        Approved,
        Executed,
        Cancelled
    }

    struct Payout {
        address recipient;
        uint256 amount;
        bytes32 reference;
        uint64 approvedAt;
        uint64 executedAt;
        PayoutStatus status;
    }

    IMergeOSPayoutTreasury public immutable treasury;
    address public owner;

    mapping(address => bool) public operators;
    mapping(bytes32 => Payout) public payouts;
    mapping(bytes32 => bool) public reservedReferences;

    event PayoutApproved(bytes32 indexed payoutId, address indexed recipient, uint256 amount, bytes32 indexed reference);
    event PayoutExecuted(bytes32 indexed payoutId, address indexed recipient, uint256 amount, bytes32 indexed reference);
    event PayoutCancelled(bytes32 indexed payoutId, bytes32 indexed reference);
    event OperatorUpdated(address indexed account, bool enabled);
    event OwnershipTransferred(address indexed previousOwner, address indexed nextOwner);

    error NotOwner();
    error NotOperator();
    error ZeroAddress();
    error InvalidAmount();
    error InvalidReference();
    error PayoutExists();
    error ReferenceExists();
    error PayoutNotApproved();

    modifier onlyOwner() {
        if (msg.sender != owner) revert NotOwner();
        _;
    }

    modifier onlyOperator() {
        if (msg.sender != owner && !operators[msg.sender]) revert NotOperator();
        _;
    }

    constructor(address treasuryAddress, address initialOwner) {
        if (treasuryAddress == address(0) || initialOwner == address(0)) revert ZeroAddress();
        treasury = IMergeOSPayoutTreasury(treasuryAddress);
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

    function approvePayout(
        bytes32 payoutId,
        address recipient,
        uint256 amount,
        bytes32 reference
    ) external onlyOperator {
        if (payoutId == bytes32(0) || reference == bytes32(0)) revert InvalidReference();
        if (recipient == address(0)) revert ZeroAddress();
        if (amount == 0) revert InvalidAmount();
        if (payouts[payoutId].status != PayoutStatus.None) revert PayoutExists();
        if (reservedReferences[reference]) revert ReferenceExists();

        reservedReferences[reference] = true;
        payouts[payoutId] = Payout({
            recipient: recipient,
            amount: amount,
            reference: reference,
            approvedAt: uint64(block.timestamp),
            executedAt: 0,
            status: PayoutStatus.Approved
        });

        emit PayoutApproved(payoutId, recipient, amount, reference);
    }

    function executePayout(bytes32 payoutId) external onlyOperator {
        Payout storage payout = payouts[payoutId];
        if (payout.status != PayoutStatus.Approved) revert PayoutNotApproved();

        payout.status = PayoutStatus.Executed;
        payout.executedAt = uint64(block.timestamp);
        treasury.release(payout.recipient, payout.amount, payout.reference);

        emit PayoutExecuted(payoutId, payout.recipient, payout.amount, payout.reference);
    }

    function cancelPayout(bytes32 payoutId) external onlyOperator {
        Payout storage payout = payouts[payoutId];
        if (payout.status != PayoutStatus.Approved) revert PayoutNotApproved();

        payout.status = PayoutStatus.Cancelled;
        reservedReferences[payout.reference] = false;
        emit PayoutCancelled(payoutId, payout.reference);
    }
}
