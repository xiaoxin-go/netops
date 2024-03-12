package model

import (
	"fmt"
	"gorm.io/gorm"
	"netops/database"
)

type TPublicWhitelist struct {
	BaseModel
	RegionId    int    `gorm:"column:region_id" json:"region_id" binding:"required"`
	Region      string `gorm:"-" json:"region" binding:"-"`
	Type        string `gorm:"column:type" json:"type" binding:"-"`
	DeviceId    int    `gorm:"column:device_id" json:"device_id" binding:"required"`
	Device      string `gorm:"-" json:"device" binding:"-"`
	NlbDeviceId int    `gorm:"column:nlb_device_id" json:"nlb_device_id"`
	NlbDevice   string `gorm:"-" json:"nlb_device" binding:"-"`
	Description string `gorm:"column:description" json:"description" binding:"-"`
}

func (TPublicWhitelist) TableName() string {
	return "t_public_whitelist"
}
func (t *TPublicWhitelist) FirstById(id int) error {
	if e := database.DB.Where("id = ?", id).First(t).Error; e != nil {
		return fmt.Errorf("获取公网暴露白名单任务失败, id: %d, err: %w", id, e)
	}
	return nil
}
func (t *TPublicWhitelist) AfterFind(tx *gorm.DB) (err error) {
	region := TRegion{}
	if err = region.QueryById(t.RegionId); err == nil {
		t.Region = region.Name
	}
	device := TFirewallDevice{}
	if err = device.QueryById(t.DeviceId); err == nil {
		t.Device = device.Name
	}
	if t.NlbDeviceId > 0 {
		nlbDevice := TNLBDevice{}
		if err = nlbDevice.QueryById(t.NlbDeviceId); err == nil {
			t.NlbDevice = nlbDevice.Name
		}
	}
	return
}

type TVpnChange struct {
	BaseModel
	PolicyName  string `gorm:"column:policy_name" json:"policy_name" binding:"required"`
	State       int    `gorm:"column:state" json:"state"`
	Description string `gorm:"column:description" json:"description" binding:"required"`
	RegionId    int    `gorm:"column:region_id" json:"region_id" binding:"required"`
	Region      string `gorm:"-" json:"region" binding:"-"`
	DeviceId    int    `gorm:"column:device_id" json:"device_id" binding:"required"`
	Device      string `gorm:"-" json:"device" binding:"-"`
}

func (TVpnChange) TableName() string {
	return "t_vpn_change"
}

func (t *TVpnChange) AfterFind(tx *gorm.DB) (err error) {
	region := TRegion{}
	if err = region.QueryById(t.RegionId); err == nil {
		t.Region = region.Name
	}
	device := TFirewallDevice{}
	if err = device.QueryById(t.DeviceId); err == nil {
		t.Device = device.Name
	}
	return
}

type TInvalidPolicyTask struct {
	BaseModel
	RegionId    int    `gorm:"column:region_id" json:"region_id" binding:"required"`
	Region      string `gorm:"-" json:"region" binding:"-"`
	Status      string `gorm:"column:status;default:ready" json:"status"`
	DeviceId    int    `gorm:"column:device_id" json:"device_id" binding:"required"`
	Device      string `gorm:"-" json:"device" binding:"-"`
	Description string `gorm:"column:description" json:"description" binding:"-"`
}

func (TInvalidPolicyTask) TableName() string {
	return "t_invalid_policy_task"
}
func (t *TInvalidPolicyTask) FirstById(id int) error {
	if e := database.DB.Where("id = ?", id).First(t); e != nil {
		return fmt.Errorf("获取无效策略任务失败, id: %d, err: %w", id, e)
	}
	return nil
}

func (t *TInvalidPolicyTask) AfterFind(tx *gorm.DB) (err error) {
	region := TRegion{}
	if err = region.QueryById(t.RegionId); err == nil {
		t.Region = region.Name
	}
	device := TFirewallDevice{}
	if err = device.QueryById(t.DeviceId); err == nil {
		t.Device = device.Name
	}
	return
}
func (t *TInvalidPolicyTask) AfterUpdate(tx *gorm.DB) (err error) {
	redisKey := fmt.Sprintf("NETOPS_INVALID_POLICY_TASK_%d", t.Id)
	database.R.Del(redisKey)
	return
}
func (t *TInvalidPolicyTask) AfterDelete(tx *gorm.DB) (err error) {
	redisKey := fmt.Sprintf("NETOPS_INVALID_POLICY_TASK_%d", t.Id)
	database.R.Del(redisKey)
	return
}
