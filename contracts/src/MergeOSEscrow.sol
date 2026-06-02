// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

interface IMergeOSEscrowToken {
    function transfer(address to, uint256 amount) external returns (bool);
    function transferFrom(address from, address to, uint256 amount) external returns (bool);
}

/// @title MergeOS project escrow
/// @notice Tracks funded projects, task reserves, worker releases, fee routing, and refunds.
contract MergeOSEscrow {
    enum ProjectStatus {
        None,
        Funded,
        Closed,
        Refunded
    }

    struct ProjectEscrow {
        address client;
        uint256 totalDeposited;
        uint256 availableWorkPool;
        uint256 platformFee;
        uint256 released;
        uint256 refunded;
        uint64 createdAt;
        ProjectStatus status;
    }

    struct TaskReserve {
        bytes32 projectId;
        address worker;
        uint256 amount;
        bool released;
        bool refunded;
    }

    IMergeOSEscrowToken public immutable token;
    address public owner;
    address public treasury;

    mapping(address => bool) public operators;
    mapping(bytes32 => ProjectEscrow) public projects;
    mapping(bytes32 => TaskReserve) public taskReserves;

    bool private locked;

    event ProjectFunded(
        bytes32 indexed projectId,
        address indexed client,
        uint256 amount,
        uint256 platformFee,
        uint256 workPool
    );
    event TaskReserved(bytes32 indexed projectId, bytes32 indexed taskId, address indexed worker, uint256 amount);
    event TaskPaid(bytes32 indexed projectId, bytes32 indexed taskId, address indexed worker, uint256 amount);
    event TaskRefunded(bytes32 indexed projectId, bytes32 indexed taskId, address indexed recipient, uint256 amount);
    event ProjectClosed(bytes32 indexed projectId);
    event ProjectRefunded(bytes32 indexed projectId, address indexed recipient, uint256 amount);
    event TreasuryUpdated(address indexed treasury);
    event OperatorUpdated(address indexed account, bool enabled);
    event OwnershipTransferred(address indexed previousOwner, address indexed nextOwner);

    error NotOwner();
    error NotOperator();
    error Reentrancy();
    error ZeroAddress();
    error InvalidAmount();
    error InvalidProject();
    error InvalidTask();
    error ProjectExists();
    error ProjectNotFunded();
    error InsufficientWorkPool();
    error TaskExists();
    error TaskNotReserved();
    error TaskAlreadyFinalized();
    error TokenTransferFailed();

    modifier onlyOwner() {
        if (msg.sender != owner) revert NotOwner();
        _;
    }

    modifier onlyOperator() {
        if (msg.sender != owner && !operators[msg.sender]) revert NotOperator();
        _;
    }

    modifier nonReentrant() {
        if (locked) revert Reentrancy();
        locked = true;
        _;
        locked = false;
    }

    constructor(address tokenAddress, address treasuryAddress, address initialOwner) {
        if (tokenAddress == address(0) || treasuryAddress == address(0) || initialOwner == address(0)) {
            revert ZeroAddress();
        }
        token = IMergeOSEscrowToken(tokenAddress);
        treasury = treasuryAddress;
        owner = initialOwner;
        emit OwnershipTransferred(address(0), initialOwner);
        emit TreasuryUpdated(treasuryAddress);
    }

    function transferOwnership(address nextOwner) external onlyOwner {
        if (nextOwner == address(0)) revert ZeroAddress();
        emit OwnershipTransferred(owner, nextOwner);
        owner = nextOwner;
    }

    function setTreasury(address treasuryAddress) external onlyOwner {
        if (treasuryAddress == address(0)) revert ZeroAddress();
        treasury = treasuryAddress;
        emit TreasuryUpdated(treasuryAddress);
    }

    function setOperator(address account, bool enabled) external onlyOwner {
        if (account == address(0)) revert ZeroAddress();
        operators[account] = enabled;
        emit OperatorUpdated(account, enabled);
    }

    function fundProject(
        bytes32 projectId,
        address client,
        uint256 amount,
        uint256 platformFee
    ) external onlyOperator nonReentrant {
        if (projectId == bytes32(0)) revert InvalidProject();
        if (client == address(0)) revert ZeroAddress();
        if (amount == 0 || platformFee > amount) revert InvalidAmount();
        if (projects[projectId].status != ProjectStatus.None) revert ProjectExists();

        uint256 workPool = amount - platformFee;
        projects[projectId] = ProjectEscrow({
            client: client,
            totalDeposited: amount,
            availableWorkPool: workPool,
            platformFee: platformFee,
            released: 0,
            refunded: 0,
            createdAt: uint64(block.timestamp),
            status: ProjectStatus.Funded
        });

        _safeTransferFrom(client, address(this), amount);
        if (platformFee > 0) {
            _safeTransfer(treasury, platformFee);
        }

        emit ProjectFunded(projectId, client, amount, platformFee, workPool);
    }

    function reserveTask(
        bytes32 projectId,
        bytes32 taskId,
        address worker,
        uint256 amount
    ) external onlyOperator {
        ProjectEscrow storage project = projects[projectId];
        if (project.status != ProjectStatus.Funded) revert ProjectNotFunded();
        if (taskId == bytes32(0)) revert InvalidTask();
        if (worker == address(0)) revert ZeroAddress();
        if (amount == 0) revert InvalidAmount();
        if (taskReserves[taskId].amount != 0) revert TaskExists();
        if (project.availableWorkPool < amount) revert InsufficientWorkPool();

        project.availableWorkPool -= amount;
        taskReserves[taskId] = TaskReserve({
            projectId: projectId,
            worker: worker,
            amount: amount,
            released: false,
            refunded: false
        });

        emit TaskReserved(projectId, taskId, worker, amount);
    }

    function releaseTask(bytes32 taskId) external onlyOperator nonReentrant {
        TaskReserve storage reserve = taskReserves[taskId];
        if (reserve.amount == 0) revert TaskNotReserved();
        if (reserve.released || reserve.refunded) revert TaskAlreadyFinalized();

        ProjectEscrow storage project = projects[reserve.projectId];
        reserve.released = true;
        project.released += reserve.amount;
        _safeTransfer(reserve.worker, reserve.amount);

        emit TaskPaid(reserve.projectId, taskId, reserve.worker, reserve.amount);
    }

    function refundTask(bytes32 taskId, address recipient) external onlyOperator nonReentrant {
        TaskReserve storage reserve = taskReserves[taskId];
        if (reserve.amount == 0) revert TaskNotReserved();
        if (reserve.released || reserve.refunded) revert TaskAlreadyFinalized();
        if (recipient == address(0)) revert ZeroAddress();

        ProjectEscrow storage project = projects[reserve.projectId];
        reserve.refunded = true;
        project.refunded += reserve.amount;
        _safeTransfer(recipient, reserve.amount);

        emit TaskRefunded(reserve.projectId, taskId, recipient, reserve.amount);
    }

    function refundProjectRemainder(bytes32 projectId, address recipient) external onlyOperator nonReentrant {
        ProjectEscrow storage project = projects[projectId];
        if (project.status != ProjectStatus.Funded && project.status != ProjectStatus.Closed) revert ProjectNotFunded();
        if (recipient == address(0)) revert ZeroAddress();

        uint256 amount = project.availableWorkPool;
        if (amount == 0) revert InvalidAmount();
        project.availableWorkPool = 0;
        project.refunded += amount;
        project.status = ProjectStatus.Refunded;
        _safeTransfer(recipient, amount);

        emit ProjectRefunded(projectId, recipient, amount);
    }

    function closeProject(bytes32 projectId) external onlyOperator {
        ProjectEscrow storage project = projects[projectId];
        if (project.status != ProjectStatus.Funded) revert ProjectNotFunded();
        project.status = ProjectStatus.Closed;
        emit ProjectClosed(projectId);
    }

    function _safeTransfer(address recipient, uint256 amount) private {
        bool ok = token.transfer(recipient, amount);
        if (!ok) revert TokenTransferFailed();
    }

    function _safeTransferFrom(address from, address recipient, uint256 amount) private {
        bool ok = token.transferFrom(from, recipient, amount);
        if (!ok) revert TokenTransferFailed();
    }
}
