package model

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"netops/conf"
	"netops/database"
	"netops/utils"
)

type TDeviceType struct {
	BaseModel
	Name        string `gorm:"column:name" json:"name" binding:"required"`
	Type        string `gorm:"column:type" json:"type" binding:"required"`
	Description string `gorm:"column:description" json:"description"`
}

func (TDeviceType) TableName() string {
	return "t_device_type"
}
func (t *TDeviceType) FirstById(id int) error {
	return firstById(t, id)
}
func (t *TDeviceType) redisKey() string {
	return fmt.Sprintf("netops_device_type_info_by_id_%d", t.Id)
}

// QueryById 带有缓存的用户信息查询
func (t *TDeviceType) QueryById(id int) error {
	t.Id = id
	k := t.redisKey()
	fields := []string{"name", "description"}
	return queryById(t, k, id, fields)
}
func (t *TDeviceType) BeforeDelete(tx *gorm.DB) (err error) {
	var count int64
	if err := database.DB.Model(&TFirewallDevice{}).Where("device_type_id = ?", t.Id).Count(&count).Error; err != nil {
		return fmt.Errorf("获取设备信息异常")
	}
	if count > 0 {
		return fmt.Errorf("当前数据已被使用")
	}
	return
}

type TImplementType struct {
	BaseModel
	Name        string `gorm:"column:name" json:"name" binding:"required"`
	Description string `gorm:"column:description" json:"description"`
}

func (TImplementType) TableName() string {
	return "t_implement_type"
}
func (t *TImplementType) FirstById(id int) error {
	return firstById(t, id)
}
func (t *TImplementType) FirstByName(name string) error {
	return firstByField(t, "name", name)
}
func (t *TImplementType) redisKey() string {
	return fmt.Sprintf("netops_implement_type_info_by_id_%d", t.Id)
}

// QueryById 带有缓存的用户信息查询
func (t *TImplementType) QueryById(id int) error {
	t.Id = id
	k := t.redisKey()
	fields := []string{"name", "description"}
	return queryById(t, k, id, fields)
}

type TIssueType struct {
	BaseModel
	RegionId        int    `gorm:"column:region_id" json:"region_id"`
	Region          string `gorm:"-" json:"region" binding:"-"`
	JiraRegion      string `gorm:"column:jira_region" json:"jira_region" binding:"required"`
	JiraEnvironment string `gorm:"column:jira_environment" json:"jira_environment" binding:"required"`
	Description     string `gorm:"column:description" json:"description"`
	Enabled         int    `gorm:"column:enabled" json:"enabled"`
}

func (t *TIssueType) FirstByRegionEnvironment(region, environment string) error {
	if err := database.DB.Where("jira_region = ? and jira_environment = ? and enabled = ?",
		region, environment, 1).First(t).Error; err != nil {
		return fmt.Errorf("获取工单类型失败, 属地: %s, 环境: %s, err: %w", region, environment, err)
	}
	return nil
}

func (i *TIssueType) AfterFind(tx *gorm.DB) (err error) {
	region := TRegion{}
	if err = region.QueryById(i.RegionId); err == nil {
		i.Region = region.Name
	}
	return
}

func (TIssueType) TableName() string {
	return "t_issue_type"
}

type TOutboundNetworkType struct {
	BaseModel
	Name            string `gorm:"column:name" json:"name" binding:"required"`
	ImplementTypeId int    `gorm:"column:implement_type_id" json:"implement_type_id" binding:"required"`
	ImplementType   string `gorm:"-" json:"implement_type" binding:"-"`
	Description     string `gorm:"column:description" json:"description"`
}

func (TOutboundNetworkType) TableName() string {
	return "t_outbound_network_type"
}

func (t *TOutboundNetworkType) AfterFind(tx *gorm.DB) (err error) {
	implementType := TImplementType{}
	if err = implementType.QueryById(t.ImplementTypeId); err == nil {
		t.ImplementType = implementType.Name
	}
	return
}

type TRegion struct {
	BaseModel
	Name           string `gorm:"column:name" json:"name" binding:"required"`
	ApiServer      string `gorm:"column:api_server" json:"api_server"`
	TaskTemplateId int    `gorm:"column:task_template_id" json:"task_template_id"`
	TaskTemplate   string `gorm:"-" json:"task_template" binding:"-"`
	Enabled        int    `gorm:"column:enabled" json:"enabled"`
	Description    string `gorm:"column:description" json:"description"`
}

func (t *TRegion) FirstById(id int) error {
	return firstById(t, id)
}
func (t *TRegion) redisKey() string {
	return fmt.Sprintf("netops_region_info_by_id_%d", t.Id)
}

// QueryById 带有缓存的用户信息查询
func (t *TRegion) QueryById(id int) error {
	t.Id = id
	k := t.redisKey()
	fields := []string{"name", "api_server", "task_template_id", "enabled", "description"}
	return queryById(t, k, id, fields)
}

func (t *TRegion) BeforeDelete(tx *gorm.DB) (err error) {
	var count int64
	if err := database.DB.Model(&TFirewallDevice{}).Where("region_id = ?", t.Id).Count(&count).Error; err != nil {
		return fmt.Errorf("获取设备信息异常")
	}
	if count > 0 {
		return fmt.Errorf("当前数据已被使用")
	}
	if err := database.DB.Model(&TIssueType{}).Where("region_id = ?", t.Id).Count(&count).Error; err != nil {
		return fmt.Errorf("获取设备信息异常")
	}
	if count > 0 {
		return fmt.Errorf("当前数据已被使用")
	}
	if err := database.DB.Model(&TTask{}).Where("region_id = ?", t.Id).Count(&count).Error; err != nil {
		return fmt.Errorf("获取设备信息异常")
	}
	if count > 0 {
		return fmt.Errorf("当前数据已被使用")
	}
	return
}

func (t *TRegion) AfterFind(tx *gorm.DB) (err error) {
	taskTemplate := TTaskTemplate{}
	if err = taskTemplate.QueryById(t.TaskTemplateId); err == nil {
		t.TaskTemplate = taskTemplate.Name
	}
	return
}

func (TRegion) TableName() string {
	return "t_region"
}

func (t *TRegion) AfterUpdate(tx *gorm.DB) (err error) {
	database.R.Del(t.redisKey())
	return
}

type TTaskTemplate struct {
	BaseModel
	Name    string `gorm:"column:name" json:"name" binding:"required"`
	Content string `gorm:"column:content" json:"content" binding:"required"`
}

func (TTaskTemplate) TableName() string {
	return "t_task_template"
}

func (t *TTaskTemplate) FirstById(id int) error {
	return firstById(t, id)
}
func (t *TTaskTemplate) redisKey() string {
	return fmt.Sprintf("netops_task_template_info_by_id_%d", t.Id)
}

// QueryById 带有缓存的用户信息查询
func (t *TTaskTemplate) QueryById(id int) error {
	t.Id = id
	k := t.redisKey()
	fields := []string{"name", "content"}
	return queryById(t, k, id, fields)
}

func (t *TTaskTemplate) BeforeDelete(tx *gorm.DB) error {
	var count int64
	if database.DB.Model(&TRegion{}).Where("task_template_id = ?", t.Id).Count(&count).Error != nil {
		return fmt.Errorf("获取网络区域信息异常")
	}
	if count > 0 {
		return fmt.Errorf("当前数据已被使用")
	}
	return nil
}

type TFirewallSubnet struct {
	BaseModel       `binding:"-"`
	InnerSubnet     string `gorm:"column:inner_subnet" json:"inner_subnet" binding:"required"`
	OuterSubnet     string `gorm:"column:outer_subnet" json:"outer_subnet" binding:"required"`
	RegionId        int    `gorm:"column:region_id" json:"region_id"`
	Region          string `gorm:"-" json:"region" binding:"-"`
	ImplementTypeId int    `gorm:"column:implement_type_id" json:"implement_type_id" binding:"required"`
	ImplementType   string `gorm:"-" json:"implement_type" binding:"-"`
	DeviceId        int    `gorm:"column:device_id" json:"device_id" binding:"required"`
	Device          string `gorm:"-" json:"device" binding:"-"`
	Description     string `gorm:"column:description" json:"description"`
}

// FindByRegionIdImplementTypeId 根据工单属地和实施类型获取网段关联的设备
func (d *TFirewallSubnet) FindByRegionIdImplementTypeId(regionId, implementTypeId int) ([]*TFirewallSubnet, error) {
	result := make([]*TFirewallSubnet, 0)
	if e := database.DB.Where("region_id = ? and implement_type_id = ?", regionId, implementTypeId).Find(&result).Error; e != nil {
		zap.L().Error("获取工单属地和实施类型对应网段关联的网络设备失败", zap.Error(e),
			zap.Int("region_id", regionId),
			zap.Int("implementTypeId", implementTypeId))
		return nil, fmt.Errorf("获取工单属地和实施类型对应网段关联的网络设备失败, err: %w", e)
	}
	return result, nil
}

func (d *TFirewallSubnet) AfterFind(tx *gorm.DB) (err error) {
	region := TRegion{}
	if err = region.QueryById(d.RegionId); err == nil {
		d.Region = region.Name
	}
	implementType := TImplementType{}
	if err = implementType.QueryById(d.ImplementTypeId); err == nil {
		d.ImplementType = implementType.Name
	}
	device := TFirewallDevice{}
	if err = device.QueryById(d.DeviceId); err == nil {
		d.Device = device.Name
	}
	return
}

func (d *TFirewallSubnet) BeforeCreate(tx *gorm.DB) (err error) {
	err = d.FormatParams()
	return
}
func (d *TFirewallSubnet) BeforeUpdate(tx *gorm.DB) (err error) {
	err = d.FormatParams()
	return
}
func (d *TFirewallSubnet) FormatParams() error {
	var (
		count int64
	)
	// 1. 校验内部网段
	if result, err := utils.ParseIP(d.InnerSubnet); err != nil {
		return err
	} else {
		d.InnerSubnet = result
	}
	// 2. 校验出向网段
	if result, err := utils.ParseIP(d.OuterSubnet); err != nil {
		return err
	} else {
		d.OuterSubnet = result
	}
	// 3. 获取网络区域信息
	device := TFirewallDevice{}
	if err := device.FirstById(d.DeviceId); err == nil {
		d.RegionId = device.RegionId
	} else {
		return err
	}
	// 4. 校验是否有重复数据，内部网段、外部网段、网络区域、实施类型、设备唯一
	if err := database.DB.Model(&TFirewallSubnet{}).
		Where("inner_subnet = ? and outer_subnet = ? and region_id = ? and implement_type_id = ? and device_id = ?",
			d.InnerSubnet, d.OuterSubnet, d.RegionId, d.ImplementTypeId, d.DeviceId).Count(&count).Error; err != nil {
		zap.L().Error(fmt.Sprintf("查询设备网段信息异常: <%s>", err.Error()))
		return fmt.Errorf("查询重复数据异常，请重试")
	}
	if count > 0 {
		return fmt.Errorf("相同区域的内外部网段只能对应一台设备")
	}

	return nil
}

func (TFirewallSubnet) TableName() string {
	return "t_firewall_subnet"
}

type TNLBSubnet struct {
	BaseModel
	Subnet      string `gorm:"column:subnet" json:"subnet" binding:"required"`
	RegionId    int    `gorm:"column:region_id" json:"region_id"`
	Region      string `gorm:"-" json:"region" binding:"-"`
	DeviceId    int    `gorm:"column:device_id" json:"device_id" binding:"required"`
	Device      string `gorm:"-" json:"device" binding:"-"`
	Description string `gorm:"column:description" json:"description"`
}

func (d *TNLBSubnet) FindByRegionId(regionId int) ([]*TNLBSubnet, error) {
	result := make([]*TNLBSubnet, 0)
	if e := database.DB.Where("region_id = ?", regionId).Find(&result).Error; e != nil {
		return nil, fmt.Errorf("获取负载均衡设备网段失败, region_id: %d, err: %w", regionId, e)
	}
	return result, nil
}
func (d *TNLBSubnet) AfterFind(tx *gorm.DB) (err error) {
	region := TRegion{}
	if err = region.QueryById(d.RegionId); err == nil {
		d.Region = region.Name
	}
	device := TNLBDevice{}
	if err = device.QueryById(d.DeviceId); err == nil {
		d.Device = device.Name
	}
	return
}

func (d *TNLBSubnet) BeforeCreate(tx *gorm.DB) (err error) {
	device := TNLBDevice{}
	if err = device.FirstById(d.DeviceId); err == nil {
		d.RegionId = device.RegionId
	}
	err = d.FormatParams()
	return
}
func (d *TNLBSubnet) BeforeUpdate(tx *gorm.DB) (err error) {
	err = d.FormatParams()
	return
}
func (d *TNLBSubnet) FormatParams() error {
	var (
		count int64
	)
	// 1. 校验出向网段
	if result, err := utils.ParseIP(d.Subnet); err != nil {
		return err
	} else {
		d.Subnet = result
	}
	// 3. 校验是否有重复数据，内部网段、外部网段、网络区域、实施类型、设备唯一
	if err := database.DB.Model(&TNLBSubnet{}).Where("subnet = ? and device_id = ?",
		d.Subnet, d.DeviceId).Count(&count).Error; err != nil {
		zap.L().Error(fmt.Sprintf("查询设备网段信息异常: <%s>", err.Error()))
		return fmt.Errorf("查询重复数据异常，请重试")
	}
	if count > 0 {
		return fmt.Errorf("相同区域的内外部网段只能对应一台设备")
	}
	return nil
}

func (TNLBSubnet) TableName() string {
	return "t_nlb_subnet"
}

type TDeviceNatAddress struct {
	EmptyModel
	RegionId            int    `gorm:"column:region_id" json:"region_id"`
	DeviceId            int    `gorm:"column:device_id" json:"device_id" binding:"required"`
	Subnet              string `gorm:"column:subnet" json:"subnet" binding:"required"`
	DeviceName          string `gorm:"-" json:"device_name"`
	Region              string `gorm:"-" json:"region"`
	StaticName          string `gorm:"static_name" json:"static_name" binding:"required"`
	Static              string `gorm:"-" json:"pool_address"`
	NatName             string `gorm:"nat_name" json:"nat_name"`
	OutboundNetworkType string `gorm:"outbound_network_type" json:"outbound_network_type"`
}

func (t *TDeviceNatAddress) FindByDeviceIdOutboundNetworkType(deviceId int, outboundNetworkType string) ([]*TDeviceNatAddress, error) {
	result := make([]*TDeviceNatAddress, 0)
	if e := database.DB.Where("device_id = ? and outbound_network_type = ?", deviceId, outboundNetworkType).Error; e != nil {
		return nil, fmt.Errorf("获取nat映射信息失败, device_id: %d, outbound_network_type: %s, err: %w", deviceId, outboundNetworkType, e)
	}
	return result, nil
}

func (TDeviceNatAddress) TableName() string {
	return "t_device_nat_address"
}

func (t *TDeviceNatAddress) BeforeCreate(tx *gorm.DB) (err error) {
	device := TFirewallDevice{}
	if err = device.FirstById(t.DeviceId); err == nil {
		t.RegionId = device.RegionId
	}
	return
}

func (t *TDeviceNatAddress) AfterFind(tx *gorm.DB) (err error) {
	device := TFirewallDevice{}
	if err = device.QueryById(t.DeviceId); err == nil {
		t.DeviceName = device.Name
	}
	region := TRegion{}
	if err = region.QueryById(t.RegionId); err == nil {
		t.Region = region.Name
	}
	pool := TDeviceNatPool{}
	if e := pool.FirstByName(t.StaticName); e == nil {
		t.Static = pool.Address
	}
	return
}

type TSubnet struct {
	BaseModel
	Subnet      string `gorm:"column:subnet" json:"subnet" binding:"required"`
	Description string `gorm:"column:description" json:"description"`
	Region      string `gorm:"column:region" json:"region" binding:"required"`
	NetType     string `gorm:"column:net_type" json:"net_type" binding:"required"`
	IpType      string `gorm:"column:ip_type" json:"ip_type" binding:"required"`
}

func (t *TSubnet) PluckSubnet() ([]string, error) {
	result := make([]string, 0)
	if e := database.DB.Model(t).Pluck("subnet", &result).Error; e != nil {
		return nil, fmt.Errorf("获取网段信息失败, err: %w", e)
	}
	return result, nil
}
func (t *TSubnet) FirstBySubnet(subnet string) error {
	if e := database.DB.Where("subnet = ?", subnet).First(t).Error; errors.Is(e, gorm.ErrRecordNotFound) {
		return fmt.Errorf("未找到网段信息, subnet: %s", subnet)
	} else if e != nil {
		return fmt.Errorf("获取网段信息失败, subnet: %s, err: %w", subnet, e)
	}
	return nil
}
func (t *TSubnet) AfterCreate(tx *gorm.DB) (err error) {
	database.R.Del(conf.SubnetListRedisKey)
	return
}
func (t *TSubnet) AfterUpdate(tx *gorm.DB) (err error) {
	redisKey := fmt.Sprintf("NETOPS_IMPLEMENT_TYPE_DETAIL_%s", t.Subnet)
	database.R.Del(redisKey)
	database.R.Del(conf.SubnetListRedisKey)
	return
}
func (t *TSubnet) AfterDelete(tx *gorm.DB) (err error) {
	redisKey := fmt.Sprintf("NETOPS_IMPLEMENT_TYPE_DETAIL_%s", t.Subnet)
	database.R.Del(redisKey)
	database.R.Del(conf.SubnetListRedisKey)
	return
}

func (TSubnet) TableName() string {
	return "t_subnet"
}

type TTaskStatus struct {
	BaseModel
	Operate        string `gorm:"column:operate" json:"operate" binding:"required"`
	TaskStatus     string `gorm:"column:task_status" json:"task_status" binding:"required"`
	TaskNextStatus string `gorm:"column:task_next_status" json:"task_next_status"`
	JiraStatus     string `gorm:"column:jira_status" json:"jira_status" binding:"required"`
	JiraNextStatus string `gorm:"column:jira_next_status" json:"jira_next_status"`
	Assignee       string `gorm:"column:assignee" json:"assignee"`
}

func (TTaskStatus) TableName() string {
	return "t_task_status"
}
func (t *TTaskStatus) AfterUpdate(tx *gorm.DB) (err error) {
	database.R.Del(fmt.Sprintf("NETOPS_TASK_STATUS_INFO_%s", t.Operate))
	return
}
func (t *TTaskStatus) FirstByOperate(operate string) error {
	if e := database.DB.Where("operate = ?", operate).First(t).Error; errors.Is(e, gorm.ErrRecordNotFound) {
		return fmt.Errorf("操作对应的任务状态不存在, 请联系管理员处理, 操作: %s", operate)
	} else if e != nil {
		zap.L().Error("获取任务状态失败", zap.String("operate", operate), zap.Error(e))
		return fmt.Errorf("根据操作获取任务状态失败, 操作: %s, err: %w", operate, e)
	}
	return nil
}

type TF5SnatPool struct {
	BaseModel
	RegionId   int    `gorm:"column:region_id" json:"region_id"`
	Region     string `gorm:"-" json:"region" binding:"-"`
	DeviceId   int    `gorm:"column:device_id" json:"device_id" binding:"required"`
	DeviceName string `gorm:"-" json:"device_name" binding:"-"`
	Name       string `gorm:"column:name" json:"name" binding:"-"`
	Subnet     string `gorm:"column:subnet" json:"subnet" binding:"-"`
}

func (TF5SnatPool) TableName() string {
	return "t_f5_snat_pool"
}
func (d *TF5SnatPool) FindByDeviceId(deviceId int) ([]*TF5SnatPool, error) {
	result := make([]*TF5SnatPool, 0)
	if e := database.DB.Where("device_id = ?", deviceId).Find(&result).Error; e != nil {
		return nil, fmt.Errorf("获取F5SnatPool失败, 设备ID: %d, e: %w", deviceId, e)
	}
	return result, nil
}
func (d *TF5SnatPool) BeforeCreate(tx *gorm.DB) (err error) {
	device := TFirewallDevice{}
	if err := device.FirstById(d.DeviceId); err == nil {
		d.RegionId = device.RegionId
	}
	return
}
func (d *TF5SnatPool) AfterFind(tx *gorm.DB) (err error) {
	device := TNLBDevice{}
	if err = device.QueryById(d.DeviceId); err == nil {
		d.DeviceName = device.Name
	}
	region := TRegion{}
	if err = region.QueryById(d.RegionId); err == nil {
		d.Region = region.Name
	}
	return
}
