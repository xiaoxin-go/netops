package conf

const (
	SessionKey = "netops_session_id"

	SubnetListRedisKey = "NetopsSubnetList"

	// 工单任务状态
	TaskStatusInit      = "init"
	TaskStatusReady     = "ready"
	TaskStatusExecuting = "executing"
	TaskStatusReview    = "review"
	TaskStatusFailed    = "failed"
	TaskStatusSuccess   = "success"
	TaskStatusReject    = "reject"

	// 工单任务操作
	TaskOperateEdit       = "修改策略"
	TaskOperateToLeader   = "送Leader审核"
	TaskOperateGeneConfig = "生成配置"
	TaskOperateVerifyPass = "审核通过"
	TaskOperateToExecutor = "送执行方审批"
	TaskOperateExec       = "执行"
	TaskOperateReject     = "驳回"

	// 任务类型
	TaskTypeFirewall = "firewall"
	TaskTypeNlb      = "nlb"

	// 地址类型
	AddressTypeAddress    = "address"
	AddressTypeAddressSet = "address-set"

	BanGongWang   = "BanGongWang"
	BanGongWangV6 = "BanGongWangIPv6"

	// 黑名单任务状态
	BlacklistTaskStatusReady     = 0
	BlacklistTaskStatusSuccess   = 1
	BlacklistTaskStatusFailed    = 2
	BlacklistTaskStatusExecuting = 3

	// 黑名单任务详情状态
	BlacklistTaskResultStatusReady     = 0
	BlacklistTaskResultStatusSuccess   = 1
	BlacklistTaskResultStatusFailed    = 2
	BlacklistTaskResultStatusExecuting = 3
	BlacklistTaskResultStatusRepeat    = 4

	ExecResultStatusFailed  = "failed"
	ExecResultStatusSuccess = "success"

	// 黑名单任务类型
	BlacklistTaskTypeShow   = 1
	BlacklistTaskTypeDeny   = 2
	BlacklistTaskTypePermit = 3

	IpTypeV4 = "ipv4"
	IpTypeV6 = "ipv6"
)
