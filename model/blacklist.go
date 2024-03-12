package model

import (
	"fmt"
	"gorm.io/gorm"
	"netops/database"
	"time"
)

type TBlacklistDevice struct {
	BaseModel
	DeviceId                   int      `gorm:"column:device_id" json:"device_id" binding:"required"`
	Device                     string   `gorm:"-" json:"device"`
	Enabled                    int      `gorm:"column:enabled" json:"enabled" binding:"required"`
	PolicyName                 string   `gorm:"policy_name" json:"policy_name"`
	ItemCountLimit             int      `gorm:"column:item_count_limit" json:"item_count_limit" binding:"required"`
	DenyCommandTemplate        string   `gorm:"deny_command_template" json:"deny_command_template"`
	PermitCommandTemplate      string   `gorm:"permit_command_template" json:"permit_command_template"`
	CreateGroupCommandTemplate string   `gorm:"create_group_command_template" json:"create_group_command_template"`
	Description                string   `gorm:"column:description" json:"description"`
	OutboundNetworkTypeIds     []int    `gorm:"-" json:"outbound_network_type_ids"`
	OutboundNetworkTypes       []string `gorm:"-" json:"outbound_network_types"`
}

func (t TBlacklistDevice) TableName() string {
	return "t_blacklist_device"
}
func (t TBlacklistDevice) BeforeCreate(tx *gorm.DB) (err error) {
	if len(t.OutboundNetworkTypeIds) == 0 {
		return fmt.Errorf("网络类型不能为空")
	}
	t.Id = t.DeviceId
	return
}
func (t *TBlacklistDevice) AfterFind(tx *gorm.DB) (err error) {
	// 获取黑名单关联网络类型
	d := TFirewallDevice{}
	if e := d.QueryById(t.DeviceId); e == nil {
		t.Device = d.Name
	}
	if e := database.DB.Model(&TBlacklistDeviceOutboundNetworkTypeRelation{}).Where("blacklist_device_id = ?", t.Id).
		Pluck("outbound_network_type_id", &t.OutboundNetworkTypeIds).Error; e != nil {
		return fmt.Errorf("获取关联网络类型异常: <%w>", e)
	}
	if e := database.DB.Model(&TOutboundNetworkType{}).Where("id in ?", t.OutboundNetworkTypeIds).Pluck("name", &t.OutboundNetworkTypes).Error; e != nil {
		return fmt.Errorf("获取网络类型异常: <%w>", e)
	}
	return
}

func (t *TBlacklistDevice) AfterCreate(tx *gorm.DB) (err error) {
	return t.bulkCreateRelation()
}

func (t *TBlacklistDevice) BeforeUpdate(tx *gorm.DB) (err error) {
	if err = t.clearRelation(); err != nil {
		return
	}
	return t.bulkCreateRelation()
}

// 清除关联关系
func (t *TBlacklistDevice) clearRelation() (err error) {
	if e := database.DB.Where("blacklist_device_id = ?", t.Id).Delete(&TBlacklistDeviceOutboundNetworkTypeRelation{}).Error; e != nil {
		return fmt.Errorf("清除关联网络异常: <%w>", e)
	}
	return nil
}

// 创建网络类型与设备的关联
func (t *TBlacklistDevice) bulkCreateRelation() (err error) {
	bulks := make([]*TBlacklistDeviceOutboundNetworkTypeRelation, 0)
	for _, v := range t.OutboundNetworkTypeIds {
		bulks = append(bulks, &TBlacklistDeviceOutboundNetworkTypeRelation{OutboundNetworkTypeId: v, BlacklistDeviceId: t.Id})
	}
	if e := database.DB.Create(&bulks).Error; e != nil {
		err = fmt.Errorf("关联网络类型异常: <%w>", e)
	}
	return err
}

type TBlacklistDeviceOutboundNetworkTypeRelation struct {
	Id                    int `gorm:"primary_key" json:"id"`
	BlacklistDeviceId     int `gorm:"column:blacklist_device_id" json:"blacklist_device_id"`
	OutboundNetworkTypeId int `gorm:"column:outbound_network_type_id"`
}

func (t TBlacklistDeviceOutboundNetworkTypeRelation) TableName() string {
	return "t_blacklist_device_outbound_network_type_relation"
}

type TBlacklistDeviceGroup struct {
	BaseModel
	DeviceId int    `gorm:"column:device_id" json:"device_id" binding:"required"` // 关联真实设备ID t_device
	Device   string `gorm:"-" json:"device" binding:"-"`
	Name     string `gorm:"column:name" json:"name" binding:"-"`
	IpType   string `gorm:"column:ip_type" json:"ip_type"`
	//ItemCount int    `gorm:"column:item_count" json:"item_count"` // 需要每天同步时，同步组内地址数量
}

func (t *TBlacklistDeviceGroup) AfterFind(tx *gorm.DB) (err error) {
	d := TFirewallDevice{}
	if err = d.QueryById(t.DeviceId); err == nil {
		t.Device = d.Name
	}
	return
}

func (t TBlacklistDeviceGroup) TableName() string {
	return "t_blacklist_device_group"
}

type TBlacklistTask struct {
	BaseModel
	Content        string    `gorm:"column:content" json:"content" binding:"required"`
	TaskType       uint32    `gorm:"column:task_type" json:"task_type" binding:"required"`
	Operator       string    `gorm:"column:operator" json:"operator" binding:"-"`
	Status         uint32    `gorm:"column:status" json:"status" binding:"-"`
	Description    string    `gorm:"column:description" json:"description" binding:"-"`
	ExecuteTime    time.Time `gorm:"column:execute_time" json:"execute_time" binding:"-"`
	ExecuteEndTime time.Time `gorm:"column:execute_end_time" json:"execute_end_time" binding:"-"`
}

func (t TBlacklistTask) TableName() string {
	return "t_blacklist_task"
}

type TBlacklistTaskInfo struct {
	Id      int                         `gorm:"primary_key" json:"id"`
	TaskId  int                         `gorm:"column:task_id" json:"task_id" binding:"-"`
	Ip      string                      `gorm:"column:ip" json:"ip" binding:"-"`
	IpType  string                      `gorm:"ip_type" json:"ip_type"`
	NetType string                      `gorm:"column:net_type" json:"net_type" binding:"-"`
	Results []*TBlacklistTaskInfoResult `gorm:"-" json:"results"`
}

func (t TBlacklistTaskInfo) TableName() string {
	return "t_blacklist_task_info"
}

type TBlacklistTaskInfoResult struct {
	Id                  int       `gorm:"primary_key" json:"id"`
	TaskId              int       `gorm:"column:task_id" json:"task_id" binding:"-"` // 索引
	TaskType            int       `gorm:"column:task_type" json:"task_type"`
	TaskInfoId          int       `gorm:"column:task_info_id" json:"task_info_id"` // 索引
	Ip                  string    `gorm:"column:ip" json:"ip"`                     // 为地址加索引
	IpType              string    `gorm:"column:ip_type" json:"ip_type"`
	DeviceId            int       `gorm:"column:device_id" json:"device_id" binding:"-"` // 关联真实的设备ID
	Device              string    `gorm:"-" json:"device"`
	BlacklistDeviceDesc string    `gorm:"-" json:"blacklist_device_desc"`
	DeviceGroupId       int       `gorm:"column:device_group_id" json:"device_group_id" binding:"-"` // 索引
	Command             string    `gorm:"column:command" json:"command" binding:"-"`
	Status              int       `gorm:"column:status" json:"status" binding:"-"`
	Result              string    `gorm:"column:result" json:"result" binding:"-"`
	StartTime           time.Time `gorm:"column:start_time" json:"start_time" binding:"-"`
	EndTime             time.Time `gorm:"column:end_time" json:"end_time" binding:"-"`
}

func (t TBlacklistTaskInfoResult) TableName() string {
	return "t_blacklist_task_info_result"
}
func (t *TBlacklistTaskInfoResult) AfterFind(tx *gorm.DB) (err error) {
	d := TFirewallDevice{}
	if err = d.QueryById(t.DeviceId); err == nil {
		t.Device = d.Name
	}
	return
}

type TBlacklistDeviceGroupAddress struct {
	BaseModel
	DeviceId      int    `gorm:"column:device_id" json:"device_id"`             // 采用真实设备ID
	DeviceGroupId int    `gorm:"column:device_group_id" json:"device_group_id"` // 建立索引
	Ip            string `gorm:"column:ip" json:"ip"`
	IpType        string `gorm:"column:ip_type" json:"ip_type"`
	//IsActive      bool   `gorm:"column:is_active" json:"is_active"`
}

func (t TBlacklistDeviceGroupAddress) TableName() string {
	return "t_blacklist_device_group_address"
}

type TBlacklistWhitelist struct {
	BaseModel
	Subnet      string `gorm:"column:subnet" json:"subnet"`
	IpType      string `gorm:"column:ip_type" json:"ip_type"`
	Description string `gorm:"column:description" json:"description"`
}

func (t TBlacklistWhitelist) TableName() string {
	return "t_blacklist_whitelist"
}
func (t *TBlacklistWhitelist) FindByIpType(ipType string) ([]*TBlacklistWhitelist, error) {
	result := make([]*TBlacklistWhitelist, 0)
	if e := database.DB.Where("ip_type = ?", ipType).Find(&result).Error; e != nil {
		return nil, fmt.Errorf("获取黑名单白名单列表失败, ip_type: %s, err: %w", ipType, e)
	}
	return result, nil
}
func (t *TBlacklistWhitelist) PluckSubnetByIpType(ipType string) ([]string, error) {
	result := make([]string, 0)
	if e := database.DB.Model(t).Where("ip_type = ?", ipType).Pluck("subnet", &result).Error; e != nil {
		return nil, fmt.Errorf("获取黑名单白名单列表失败, ip_type: %s, err: %w", ipType, e)
	}
	return result, nil
}
