package task

import (
	"fmt"
	"netops/conf"
	"netops/model"
)

func GetJiraAttachment(taskId int, operator string) error {
	th := NewTaskHandlerById(taskId)
	if e := th.SyncJiraStatus(); e != nil {
		return e
	}
	if _, e := th.CanOperate(conf.TaskOperateEdit); e != nil {
		return e
	}
	if e := th.GetTaskInfos(); e != nil {
		return e
	}
	model.AddLog(operator, fmt.Sprintf("获取工单<%s>附件", th.Task().JiraKey))
	return nil
}

// AddTaskInfo 添加工单策略
func AddTaskInfo(info *model.TTaskInfo, operator string) error {
	th := NewTaskHandlerById(info.TaskId)
	if e := th.SyncJiraStatus(); e != nil {
		return e
	}
	if _, e := th.CanOperate(conf.TaskOperateEdit); e != nil {
		return e
	}
	if e := th.AddInfo(info); e != nil {
		return e
	}
	model.AddLog(operator, fmt.Sprintf("添加工单<%s>策略信息: <%+v>", th.Task().JiraKey, info))
	return nil
}

// DelTaskInfo 删除工单策略
func DelTaskInfo(info *model.TTaskInfo, operator string) error {
	th := NewTaskHandlerById(info.TaskId)
	if e := th.SyncJiraStatus(); e != nil {
		return e
	}
	if _, e := th.CanOperate(conf.TaskOperateEdit); e != nil {
		return e
	}
	if e := info.Delete(); e != nil {
		return e
	}
	model.AddLog(operator, fmt.Sprintf("删除工单<%s>策略信息: <%+v>", th.GetJiraKey(), info))
	return nil
}

// GeneConfig 生成配置
func GeneConfig(taskId int, operator string) error {
	// 更新工单状态 -> 校验流程 -> 生成配置 -> 更新jira流程
	th := NewTaskHandlerById(taskId)
	if e := th.SyncJiraStatus(); e != nil {
		return e
	}
	if _, e := th.CanOperate(conf.TaskOperateGeneConfig); e != nil {
		return e
	}
	if th.GetTaskType() == conf.TaskTypeFirewall {
		if e := th.GeneFirewallConfig(); e != nil {
			return e
		}
	} else {
		if e := th.GeneNlbConfig(); e != nil {
			return e
		}
	}
	if e := th.UpdateStatusByOperate(conf.TaskOperateGeneConfig); e != nil {
		return e
	}
	model.AddLog(operator, fmt.Sprintf("生成工单<%s>配置", th.Task().JiraKey))
	return nil
}

// ExecTask 执行工单任务
func ExecTask(taskId int, operator string) error {
	th := NewTaskHandlerById(taskId)
	th.SetOperator(operator)
	if e := th.SyncJiraStatus(); e != nil {
		return e
	}
	if _, e := th.CanOperate(conf.TaskOperateExec); e != nil {
		return e
	}
	if e := th.Exec(); e != nil {
		return e
	}
	model.AddLog(operator, fmt.Sprintf("执行工单<%s>", th.Task().JiraKey))
	return nil
}

// VerifyPass 审核通过
func VerifyPass(taskId int, operator string) error {
	th := NewTaskHandlerById(taskId)
	if e := th.SyncJiraStatus(); e != nil {
		return e
	}
	if _, e := th.CanOperate(conf.TaskOperateVerifyPass); e != nil {
		return e
	}
	if e := th.UpdateJiraTransitionByOperate(conf.TaskOperateVerifyPass); e != nil {
		return e
	}
	if e := th.UpdateStatusByOperate(conf.TaskOperateVerifyPass); e != nil {
		return e
	}
	model.AddLog(operator, fmt.Sprintf("审核通过<%s>", th.Task().JiraKey))
	return nil
}

// ToExecutor 送执行方审批
func ToExecutor(taskId int, operator string) error {
	th := NewTaskHandlerById(taskId)
	if e := th.SyncJiraStatus(); e != nil {
		return e
	}
	if _, e := th.CanOperate(conf.TaskOperateToExecutor); e != nil {
		return e
	}
	if e := th.UpdateJiraTransitionByOperate(conf.TaskOperateToExecutor); e != nil {
		return e
	}
	if e := th.UpdateStatusByOperate(conf.TaskOperateToExecutor); e != nil {
		return e
	}
	model.AddLog(operator, fmt.Sprintf("送执行方审批<%s>", th.Task().JiraKey))
	return nil
}

// Reject 驳回工单
func Reject(taskId int, content, operator string) error {
	th := NewTaskHandlerById(taskId)
	if e := th.SyncJiraStatus(); e != nil {
		return e
	}
	if _, e := th.CanOperate(conf.TaskOperateReject); e != nil {
		return e
	}
	if content != "" {
		if e := th.AddJiraComment(content); e != nil {
			return e
		}
	}
	if e := th.UpdateStatusByOperate(conf.TaskOperateReject); e != nil {
		return e
	}
	if e := th.UpdateJiraTransitionByOperate(conf.TaskOperateReject); e != nil {
		return e
	}
	model.AddLog(operator, fmt.Sprintf("驳回工单<%s>->%s", th.Task().JiraKey, content))
	return nil
}

// SyncJira 同步工单信息
func SyncJira(taskId int, operator string) error {
	th := NewTaskHandlerById(taskId)
	if e := th.SyncJiraStatus(); e != nil {
		return e
	}
	if e := th.SyncJiraRegionEnvironment(); e != nil {
		return e
	}
	model.AddLog(operator, fmt.Sprintf("同步工单状态<%s>", th.Task().JiraKey))
	return nil
}

// ToLeader 送Leader审核
func ToLeader(taskId int, operator string) error {
	th := NewTaskHandlerById(taskId)
	if e := th.SyncJiraStatus(); e != nil {
		return e
	}
	if _, e := th.CanOperate(conf.TaskOperateToLeader); e != nil {
		return e
	}
	if e := th.UpdateJiraTransitionByOperate(conf.TaskOperateToLeader); e != nil {
		return e
	}
	model.AddLog(operator, fmt.Sprintf("送leader审核<%s>", th.Task().JiraKey))
	return nil
}
