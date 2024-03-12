package model

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"netops/conf"
	"netops/database"
	"time"
)

type TTask struct {
	BaseModel
	JiraKey         string `gorm:"column:jira_key" json:"jira_key" binding:"required"`
	JiraRegion      string `gorm:"column:jira_region" json:"jira_region"`
	JiraEnvironment string `gorm:"column:jira_environment" json:"jira_environment"`
	Summary         string `gorm:"column:summary" json:"summary"`
	Creator         string `gorm:"column:creator" json:"creator"`
	Department      string `gorm:"column:department" json:"department"`
	JiraStatus      string `gorm:"column:jira_status" json:"jira_status"`
	Description     string `gorm:"column:description" json:"description"`
	Assignee        string `gorm:"column:assignee" json:"assignee"`
	Status          string `gorm:"column:status" json:"status"`
	ErrorInfo       string `gorm:"column:error_info" json:"error_info"`
	RegionId        int    `gorm:"column:region_id" json:"region_id"`
	Region          string `gorm:"-" json:"region"`
	TemplateId      int    `gorm:"-" json:"template_id"`
	ImplementType   string `gorm:"column:implement_type" json:"implement_type"`
	//NetworkOpeningRange string     `gorm:"column:network_opening_range" json:"network_opening_range"` // 网络范围，如果是办公网则需要切换成固定的地址组
	Type           string     `gorm:"column:type" json:"type"`
	ExecuteTime    *time.Time `gorm:"column:execute_time" json:"execute_time"`
	ExecuteEndTime *time.Time `gorm:"column:execute_end_time" json:"execute_end_time"`
	ExecuteUseTime int        `gorm:"column:execute_use_time" json:"execute_use_time"`
	IsDeleted      int        `gorm:"column:is_deleted" json:"is_deleted"`
}

func (t *TTask) FirstById(id int) error {
	return firstById(t, id)
}
func (t *TTask) redisKeyById() string {
	return fmt.Sprintf("netops_task_info_by_id_%d", t.Id)
}
func (t *TTask) ExistsByJiraKey(jiraKey string) (bool, error) {
	if e := database.DB.Where("jira_key = ?", jiraKey).First(t).Error; errors.Is(e, gorm.ErrRecordNotFound) {
		return false, nil
	} else if e != nil {
		return false, fmt.Errorf("根据工单号获取工单是否存在失败, 工单号: %w", e)
	}
	return true, nil
}

// QueryById 带有缓存的menu信息查询
func (t *TTask) QueryById(id int) error {
	t.Id = id
	k := t.redisKeyById()
	fields := []string{
		"jira_key",
		"jira_region",
		"status",
		"jira_status",
		"region_iid",
		"type",
	}
	return queryById(t, k, id, fields)
}

func (t *TTask) Create() error {
	if e := database.DB.Create(t).Error; e != nil {
		zap.L().Error("创建任务失败", zap.Any("task", t), zap.Error(e))
		return fmt.Errorf("创建任务失败, err: %w", e)
	}
	return nil
}
func (t *TTask) UpdateStatus(status string) error {
	if e := database.DB.Model(t).Update("status", status).Error; e != nil {
		zap.L().Error("更新任务状态失败", zap.Error(e), zap.Any("task", t), zap.String("status", status))
		return fmt.Errorf("更新任务状态失败, status: %s, err: %w", status, e)
	}
	return nil
}
func (t *TTask) UpdateJiraStatus(status string) error {
	if e := database.DB.Model(t).Update("jira_status", status).Error; e != nil {
		zap.L().Error("更新任务状态工单失败", zap.Error(e), zap.Any("task", t), zap.String("status", status))
		return fmt.Errorf("更新任务状态工单失败, status: %s, err: %w", status, e)
	}
	return nil
}
func (t *TTask) Save() error {
	if e := database.DB.Save(t).Error; e != nil {
		zap.L().Error("保存任务失败", zap.Error(e), zap.Any("task", t))
		return fmt.Errorf("保存任务失败, err: %w", e)
	}
	return nil
}

func (TTask) TableName() string {
	return "t_task"
}

func (t *TTask) AfterFind(tx *gorm.DB) (err error) {
	region := TRegion{}
	if err = region.QueryById(t.RegionId); err == nil {
		t.Region = region.Name
		t.TemplateId = region.TaskTemplateId
	}
	return
}

func (t *TTask) AfterDelete(tx *gorm.DB) (err error) {
	database.DB.Delete(&TTaskInfo{}, "task_id = ?", t.Id)
	return
}

type TTaskInfo struct {
	BaseModel
	TaskId              int    `gorm:"column:task_id" json:"task_id"`
	Src                 string `gorm:"column:src" json:"src" binding:"required"`
	Dst                 string `gorm:"column:dst" json:"dst" binding:"required"`
	DPort               string `gorm:"column:dport" json:"dport" binding:"required"`
	Direction           string `gorm:"column:direction" json:"direction" binging:"required"`
	OutboundNetworkType string `gorm:"column:outbound_network_type" json:"outbound_network_type"`
	PoolName            string `gorm:"column:pool_name" json:"pool_name"` // nat策略映射的名称
	NatName             string `gorm:"column:nat_name" json:"nat_name"`   // srx nat的名称
	PoolAddress         string `gorm:"-" json:"pool_address"`
	Protocol            string `gorm:"column:protocol" json:"protocol" binding:"required"`
	StaticIp            string `gorm:"column:static_ip" json:"static_ip"`
	StaticPort          string `gorm:"column:static_port" json:"static_port"`
	Action              string `gorm:"column:action" json:"action"`
	DeviceId            int    `gorm:"column:device_id" json:"device_id"`
	Device              string `gorm:"-" json:"device" binding:"-"`
	Command             string `gorm:"column:command" json:"command"`
	ExistsConfig        string `gorm:"column:exists_config" json:"exists_config"`
	Result              string `gorm:"column:result" json:"result"`
	Status              string `gorm:"column:status" json:"status"`
	Node                string `gorm:"node" json:"node"`
	NodePort            string `gorm:"node_port" json:"node_port"`
	SNat                string `gorm:"s_nat" json:"s_nat"`
	VsCommand           string `gorm:"vs_command" json:"vs_command"`
	PoolCommand         string `gorm:"pool_command" json:"pool_command"`
}

func (TTaskInfo) TableName() string {
	return "t_task_info"
}
func (t *TTaskInfo) FirstById(id int) error {
	if e := database.DB.Where("id = ?", id).First(t).Error; errors.Is(e, gorm.ErrRecordNotFound) {
		return fmt.Errorf("任务策略不存在, id: %d", id)
	} else if e != nil {
		return fmt.Errorf("获取任务策略失败, id: %d, err: %w", id, e)
	}
	return nil
}
func (t *TTaskInfo) Create() error {
	if e := database.DB.Create(t).Error; e != nil {
		zap.L().Error("添加任务详情状态失败", zap.Error(e), zap.Any("info", t))
		return fmt.Errorf("添加更新任务详情状态失败, err: %w", e)
	}
	return nil
}
func (t *TTaskInfo) Save() error {
	if e := database.DB.Save(t).Error; e != nil {
		zap.L().Error("更新任务详情状态失败", zap.Error(e), zap.Any("info", t))
		return fmt.Errorf("更新任务详情状态失败, err: %w", e)
	}
	return nil
}
func (t *TTaskInfo) UpdateStatus(status string) error {
	if e := database.DB.Model(t).Update("status", status).Error; e != nil {
		zap.L().Error("更新任务详情状态失败", zap.Error(e),
			zap.String("status", status),
			zap.Int("task_id", t.Id))
		return fmt.Errorf("更新任务详情状态失败, err: %w", e)
	}
	return nil
}
func (t *TTaskInfo) UpdateStatusAndResult(status, result string) error {
	if e := database.DB.Model(t).Updates(map[string]string{"status": status, "result": result}).Error; e != nil {
		zap.L().Error("更新任务详情状态和结果失败", zap.Error(e),
			zap.String("status", status),
			zap.String("result", result),
			zap.Int("task_id", t.Id))
		return fmt.Errorf("更新任务详情状态和结果失败, err: %w", e)
	}
	return nil
}
func (t *TTaskInfo) DeleteByTaskId(taskId int) error {
	if e := database.DB.Delete(t, "task_id = ?", taskId).Error; e != nil {
		return fmt.Errorf("删除工单信息失败, task_id: %d, err: %w", taskId, e)
	}
	return nil
}
func (t *TTaskInfo) Delete() error {
	if e := database.DB.Delete(t).Error; e != nil {
		return fmt.Errorf("删除工单策略信息失败, id: %d, err: %w", t.Id, e)
	}
	return nil
}
func (t *TTaskInfo) FindByTaskId(taskId int) ([]*TTaskInfo, error) {
	result := make([]*TTaskInfo, 0)
	if e := database.DB.Where("task_id = ?", taskId).Find(&result).Error; e != nil {
		zap.L().Error("根据任务ID获取任务详情失败", zap.Error(e), zap.Int("task_id", taskId))
		return nil, fmt.Errorf("根据任务ID获取任务详情失败, 任务ID: %d, err: %w", taskId, e)
	}
	return result, nil
}
func (t *TTaskInfo) FindExecInfoByTaskId(taskId int) ([]*TTaskInfo, error) {
	result := make([]*TTaskInfo, 0)
	if e := database.DB.Where("task_id = ? and action = ? and status != ?", taskId, "deny", "success").Find(&result).Error; e != nil {
		zap.L().Error("获取需要执行的任务详情失败", zap.Error(e), zap.Int("task_id", taskId))
		return nil, fmt.Errorf("获取需要执行的任务详情失败, 任务ID: %d, err: %w", taskId, e)
	}
	return result, nil
}
func (t *TTaskInfo) BulkCreate(data []*TTaskInfo) error {
	if e := database.DB.Create(&data).Error; e != nil {
		return fmt.Errorf("保存工单详情失败, err: %w", e)
	}
	return nil
}
func (t *TTaskInfo) AfterFind(tx *gorm.DB) (err error) {
	if t.PoolName != "" {
		pool := TDeviceNatPool{}
		if e := pool.QueryByName(t.PoolName); e == nil {
			t.PoolAddress = pool.Address
		}
	}
	task := TTask{}
	if err = task.QueryById(t.TaskId); err == nil {
		if task.Type == conf.TaskTypeFirewall {
			device := TFirewallDevice{}
			if err = device.QueryById(t.DeviceId); err == nil {
				t.Device = device.Name
			}
		} else {
			device := TNLBDevice{}
			if err = device.QueryById(t.DeviceId); err == nil {
				t.Device = device.Name
			}
		}
	}
	return
}

type TTaskOperateLog struct {
	BaseModel
	TaskId   int    `gorm:"column:task_id" json:"task_id"`
	Operator string `gorm:"column:operator" json:"operator"`
	Content  string `gorm:"column:content" json:"content"`
}

func (t *TTaskOperateLog) LastByTaskId(taskId int) error {
	if e := database.DB.Where("task_id = ?", taskId).Last(t).Error; errors.Is(e, gorm.ErrRecordNotFound) {
		return fmt.Errorf("工单没有日志")
	} else if e != nil {
		return fmt.Errorf("获取任务日志失败, task_id: %d, err: %w", taskId, e)
	}
	return nil
}

func (TTaskOperateLog) TableName() string {
	return "t_task_operate_log"
}
