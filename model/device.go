package model

import (
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"netops/conf"
	"netops/database"
	"netops/utils"
)

type TFirewallDevice struct {
	BaseModel
	RegionId            int    `gorm:"column:region_id" json:"region_id"`
	Name                string `gorm:"column:name" json:"name" binding:"required"`
	Host                string `gorm:"column:host" json:"host" binding:"required"`
	Port                int    `gorm:"column:port" json:"port"`
	Username            string `gorm:"column:username" json:"username" binding:"required"`
	Password            string `gorm:"column:password" json:"password" binding:"required"`
	EnablePassword      string `gorm:"column:enable_password" json:"enable_password"`
	DeviceTypeId        int    `gorm:"column:device_type_id" json:"device_type_id" binding:"required"`
	InPolicy            string `gorm:"in_policy" json:"in_policy"`
	OutPolicy           string `gorm:"out_policy" json:"out_policy"`
	Enabled             int    `gorm:"column:enabled" json:"enabled"`
	Region              string `gorm:"-" json:"region" binding:"-"`
	DeviceType          string `gorm:"-" json:"device_type" binding:"-"`
	InDenyPolicyName    string `gorm:"in_deny_policy_name" json:"in_deny_policy_name"`
	OutDenyPolicyName   string `gorm:"out_deny_policy_name" json:"out_deny_policy_name"`
	InPermitPolicyName  string `gorm:"in_permit_policy_name" json:"in_permit_policy_name"`
	OutPermitPolicyName string `gorm:"out_permit_policy_name" json:"out_permit_policy_name"`
	ParseStatus         string `gorm:"parse_status" json:"parse_status"`
	RegionApiServer     string `gorm:"-"`
}

func (TFirewallDevice) TableName() string {
	return "t_firewall_device"
}
func (d *TFirewallDevice) FirstById(id int) error {
	return firstById(d, id)
}
func (d *TFirewallDevice) redisKey() string {
	return fmt.Sprintf("netops_device_info_by_id_%d", d.Id)
}

// QueryById 带有缓存的menu信息查询
func (d *TFirewallDevice) QueryById(id int) error {
	d.Id = id
	k := d.redisKey()
	fields := []string{
		"region_id",
		"name",
		"host",
	}
	return queryById(d, k, id, fields)
}
func (d *TFirewallDevice) UpdateParseStatus(status string) error {
	if e := database.DB.Model(d).Update("parse_status", status).Error; e != nil {
		zap.L().Error("更新设备配置解析状态失败", zap.Error(e), zap.Any("data", d))
		return fmt.Errorf("更新设备配置解析状态失败, err: %w", e)
	}
	return nil
}
func (d *TFirewallDevice) Save(tx *gorm.DB) error {
	if tx == nil {
		tx = database.DB
	}
	if e := tx.Save(d).Error; e != nil {
		zap.L().Error("更新设备信息失败", zap.Error(e))
		return fmt.Errorf("更新设备信息失败, err: %w", e)
	}
	return nil
}

// BeforeCreate 创建时，加密密码
func (d *TFirewallDevice) BeforeCreate(tx *gorm.DB) (err error) {
	d.Password, err = utils.AesEncrypt(d.Password, conf.Config.AesKey)
	d.EnablePassword, err = utils.AesEncrypt(d.EnablePassword, conf.Config.AesKey)
	return
}
func (d *TFirewallDevice) AfterUpdate(tx *gorm.DB) (err error) {
	database.R.Del(d.redisKey())
	return
}
func (d *TFirewallDevice) BeforeUpdate(tx *gorm.DB) (err error) {
	old := TFirewallDevice{}
	if err = old.FirstById(d.Id); err != nil {
		return
	}
	if old.Password != d.Password {
		d.Password, err = utils.AesEncrypt(d.Password, conf.Config.AesKey)
	}
	if old.EnablePassword != d.EnablePassword {
		d.EnablePassword, err = utils.AesEncrypt(d.EnablePassword, conf.Config.AesKey)
	}
	return
}
func (d *TFirewallDevice) AfterFind(tx *gorm.DB) (err error) {
	deviceType := TDeviceType{}
	if err = deviceType.QueryById(d.DeviceTypeId); err == nil {
		d.DeviceType = deviceType.Name
	}
	region := TRegion{}
	if err = region.QueryById(d.RegionId); err == nil {
		d.Region = region.Name
	}
	return
}

type TNLBDevice struct {
	BaseModel
	RegionId     int    `gorm:"column:region_id" json:"region_id"`
	Name         string `gorm:"column:name" json:"name" binding:"required"`
	Host         string `gorm:"column:host" json:"host" binding:"required"`
	Port         int    `gorm:"column:port" json:"port"`
	Username     string `gorm:"column:username" json:"username" binding:"required"`
	Password     string `gorm:"column:password" json:"password" binding:"required"`
	DeviceTypeId int    `gorm:"column:device_type_id" json:"device_type_id" binding:"required"`
	Enabled      int    `gorm:"column:enabled" json:"enabled"`
	Region       string `gorm:"-" json:"region" binding:"-"`
	DeviceType   string `gorm:"-" json:"device_type" binding:"-"`
	ParseStatus  string `gorm:"parse_status" json:"parse_status"`
}

func (TNLBDevice) TableName() string {
	return "t_nlb_device"
}
func (d *TNLBDevice) String() string {
	return fmt.Sprintf("{id: %d, RegionId: %d, host: %s, device_type_id: %d}",
		d.Id, d.RegionId, d.Host, d.DeviceTypeId)
}
func (d *TNLBDevice) FirstById(id int) error {
	return firstById(d, id)
}
func (d *TNLBDevice) redisKey() string {
	return fmt.Sprintf("netops_nlb_device_info_by_id_%d", d.Id)
}

// QueryById 带有缓存的menu信息查询
func (d *TNLBDevice) QueryById(id int) error {
	d.Id = id
	k := d.redisKey()
	fields := []string{
		"region_id",
		"name",
		"host",
	}
	return queryById(d, k, id, fields)
}
func (d *TNLBDevice) UpdateParseStatus(status string) error {
	if e := database.DB.Model(d).Update("parse_status", status).Error; e != nil {
		zap.L().Error("更新解析状态失败", zap.Error(e), zap.String("status", status))
		return fmt.Errorf("更新解析状态失败, status: %s, err: %w", status, e)
	}
	return nil
}

// BeforeCreate 创建时，加密密码
func (d *TNLBDevice) BeforeCreate(tx *gorm.DB) (err error) {
	d.Password, err = utils.AesEncrypt(d.Password, conf.Config.AesKey)
	return
}
func (d *TNLBDevice) AfterUpdate(tx *gorm.DB) (err error) {
	redisKey := fmt.Sprintf("NETOPS_NLB_DEVICE_INFO_%d", d.Id)
	database.R.Del(redisKey)
	return
}
func (d *TNLBDevice) BeforeUpdate(tx *gorm.DB) (err error) {
	old := TNLBDevice{}
	if err = old.FirstById(d.Id); err != nil {
		return
	}
	if old.Password != d.Password {
		d.Password, err = utils.AesEncrypt(d.Password, conf.Config.AesKey)
	}
	return
}
func (d *TNLBDevice) AfterFind(tx *gorm.DB) (err error) {
	deviceType := TDeviceType{}
	if err = deviceType.QueryById(d.DeviceTypeId); err == nil {
		d.DeviceType = deviceType.Name
	}
	region := TRegion{}
	if err = region.QueryById(d.RegionId); err == nil {
		d.Region = region.Name
	}
	return
}

type TDeviceBackup struct {
	BaseModel
	DeviceId int    `gorm:"column:device_id" json:"device_id"`
	Device   string `gorm:"-" json:"device"`
	Filename string `gorm:"column:filename" json:"filename"`
	Md5      string `gorm:"column:md5" json:"md5"`
	Size     int    `gorm:"column:size" json:"size"`
}

func (t *TDeviceBackup) FirstById(id int) error {
	if e := database.DB.First(t, id).Error; e != nil {
		return fmt.Errorf("获取备份信息失败, id: %d, err: %w", id, e)
	}
	return nil
}
func (t *TDeviceBackup) Create() error {
	if e := database.DB.Create(t).Error; e != nil {
		zap.L().Error("保存设备备份信息失败", zap.Error(e), zap.Any("data", t))
		return fmt.Errorf("保存备份信息失败, err: %w", e)
	}
	return nil
}

func (t *TDeviceBackup) AfterFind(tx *gorm.DB) (err error) {
	device := TFirewallDevice{}
	if err = device.QueryById(t.DeviceId); err == nil {
		t.Device = device.Name
	}
	return
}
