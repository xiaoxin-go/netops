package model

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"netops/database"
	"time"
)

type TDevicePolicy struct {
	BaseModel
	DeviceId  int    `gorm:"column:device_id" json:"device_id" binding:"required"`
	Name      string `gorm:"column:name" json:"name" binding:"required"`
	Direction string `gorm:"column:direction" json:"direction" binding:"required"`
	Src       string `gorm:"column:src" json:"src" binding:"required"`
	SrcGroup  string `gorm:"column:src_group" json:"src_group"`
	Dst       string `gorm:"column:dst" json:"dst" binding:"required"`
	DstGroup  string `gorm:"column:dst_group" json:"dst_group"`
	Port      string `gorm:"column:port" json:"port" binding:"required"`
	PortGroup string `gorm:"column:port_group" json:"port_group" binding:"required"`
	Protocol  string `gorm:"column:protocol" json:"protocol" binding:"required"`
	Action    string `gorm:"column:action" json:"action" binding:"required"`
	Command   string `gorm:"column:command" json:"command"`
	Line      int    `gorm:"column:line" json:"line"`
	Valid     bool   `gorm:"column:valid" json:"valid"`
}

func (TDevicePolicy) TableName() string {
	return "t_device_policy"
}
func (t *TDevicePolicy) DeleteByDeviceId(deviceId int, tx *gorm.DB) error {
	if tx == nil {
		tx = database.DB
	}
	if e := tx.Delete(t, "device_id = ?", deviceId).Error; e != nil {
		return fmt.Errorf("清除历史策略失败, device_id: %d, err: %w", deviceId, e)
	}
	return nil
}

type TDevicePort struct {
	Id       int    `gorm:"primary_key" json:"id"`
	DeviceId int    `gorm:"column:device_id" json:"device_id" binding:"required"`
	Name     string `gorm:"column:name" json:"name" binding:"required"`
	Protocol string `gorm:"column:protocol" json:"protocol" binding:"required"`
	Start    int    `gorm:"column:start" json:"start" binding:"required"`
	End      int    `gorm:"column:end" json:"end" binding:"end"`
}

func (t *TDevicePort) DeleteByDeviceId(deviceId int, tx *gorm.DB) error {
	if tx == nil {
		tx = database.DB
	}
	if e := tx.Delete(t, "device_id = ?", deviceId).Error; e != nil {
		zap.L().Error("删除设备端口失败", zap.Error(e), zap.Int("device_id", deviceId))
		return fmt.Errorf("删除设备端口失败, 设备ID: %d, err: %w", deviceId, e)
	}
	return nil
}
func (TDevicePort) TableName() string {
	return "t_device_port"
}

type TDeviceAddressGroup struct {
	Id          int    `gorm:"primary_key" json:"id"`
	DeviceId    int    `gorm:"column:device_id" json:"device_id" binding:"required"`
	Device      string `gorm:"-" json:"device" binding:"-"`
	Name        string `gorm:"column:name" json:"name" binding:"required"`
	Address     string `gorm:"column:address" json:"address" binding:"required"`
	AddressType string `gorm:"column:address_type" json:"address_type"` // 地址类型，是address 还是address-set
	Zone        string `gorm:"column:zone" json:"zone" binding:"-"`
}

func (TDeviceAddressGroup) TableName() string {
	return "t_device_address_group"
}

type TDeviceNatPool struct {
	EmptyModel
	DeviceId int    `gorm:"column:device_id" json:"device_id"`
	Device   string `gorm:"-" json:"device"`
	NatType  string `gorm:"column:nat_type" json:"nat_type"`
	Name     string `gorm:"column:name" json:"name"`
	Address  string `gorm:"column:address" json:"address"`
	Port     string `gorm:"column:port" json:"port"`
	Command  string `gorm:"column:command" json:"command"`
}

func (t *TDeviceNatPool) GetId() int {
	return t.Id
}

func (TDeviceNatPool) TableName() string {
	return "t_device_nat_pool"
}
func (t *TDeviceNatPool) FirstById(id int) error {
	return firstById(t, id)
}
func (t *TDeviceNatPool) FirstByName(name string) error {
	return firstByField(t, "name", name)
}
func (t *TDeviceNatPool) redisKey() string {
	return fmt.Sprintf("netops_device_nat_pool_info_by_id_%d", t.Id)
}

// QueryById 带有缓存的用户信息查询
func (t *TDeviceNatPool) QueryById(id int) error {
	t.Id = id
	k := t.redisKey()
	fields := []string{"device_id", "nat_type", "name", "address", "port", "command"}
	return queryById(t, k, id, fields)
}
func (t *TDeviceNatPool) QueryByName(name string) error {
	redisKey := fmt.Sprintf("netops_device_nat_pool_info_by_name_%s", name)
	params := map[string]any{"name": name}
	fields := []string{"id", "name", "address"}
	return QueryDataByParams(redisKey, params, fields, t, time.Hour)
}

func (t *TDeviceNatPool) AfterFind(tx *gorm.DB) (err error) {
	device := TFirewallDevice{}
	if err = device.QueryById(t.DeviceId); err == nil {
		t.Device = device.Name
	}
	return
}

type TDeviceNat struct {
	BaseModel
	Id               int    `gorm:"primary_key" json:"id"`
	DeviceId         int    `gorm:"column:device_id" json:"device_id"`
	Device           string `gorm:"-" json:"device"`
	Network          string `gorm:"column:network" json:"network"` // 真实地址
	Static           string `gorm:"column:static" json:"static"`   // 转换后的地址
	Protocol         string `gorm:"column:protocol" json:"protocol"`
	NetworkPort      string `gorm:"column:network_port" json:"network_port"`
	StaticPort       string `gorm:"column:static_port" json:"static_port"`
	Command          string `gorm:"column:command" json:"command"`
	Direction        string `gorm:"column:direction" json:"direction"`
	Destination      string `gorm:"column:destination" json:"destination"`             // 访问目标地址
	DestinationGroup string `gorm:"column:destination_group" json:"destination_group"` // 访问目标地址名
	NetworkGroup     string `gorm:"column:network_group" json:"network_group"`         // 内部真实地址组
	StaticGroup      string `gorm:"column:static_group" json:"static_group"`           // 映射地址组
}

func (TDeviceNat) TableName() string {
	return "t_device_nat"
}

func (d *TDeviceNat) AfterFind(tx *gorm.DB) (err error) {
	device := TFirewallDevice{}
	if err = device.QueryById(d.DeviceId); err == nil {
		d.Device = device.Name
	}
	return
}

type TDeviceSrxNat struct {
	Id        int    `gorm:"primary_key" json:"id"`
	DeviceId  int    `gorm:"column:device_id" json:"device_id"`
	Device    string `gorm:"-" json:"device"`
	Direction string `gorm:"column:direction" json:"direction"`
	Src       string `gorm:"column:src" json:"src"`
	Dst       string `gorm:"column:dst" json:"dst"`
	NatType   string `gorm:"column:nat_type" json:"nat_type"`
	DstPort   string `gorm:"column:dst_port" json:"dst_port"`
	Protocol  string `gorm:"column:protocol" json:"protocol"`
	Rule      string `gorm:"column:rule" json:"rule"`
	Pool      string `gorm:"column:pool" json:"pool"`
	Command   string `gorm:"column:command" json:"command"`
}

func (TDeviceSrxNat) TableName() string {
	return "t_device_srx_nat"
}

func (d *TDeviceSrxNat) AfterFind(tx *gorm.DB) (err error) {
	device := TFirewallDevice{}
	if err = device.QueryById(d.DeviceId); err == nil {
		d.Device = device.Name
	}
	return
}

type TF5Vs struct {
	Id                       int            `gorm:"primary_key" json:"id"`
	DeviceId                 int            `gorm:"column:device_id" json:"device_id"`
	Device                   string         `gorm:"-" json:"device"`
	Name                     string         `gorm:"column:name" json:"name"`
	Partition                string         `gorm:"partition" json:"partition"`
	Source                   string         `gorm:"column:source" json:"source"`
	Destination              string         `gorm:"column:destination" json:"destination"`
	SourceAddressTranslation string         `gorm:"column:source_address_translation" json:"source_address_translation"`
	Enabled                  bool           `gorm:"column:enabled" json:"enabled"`
	ProfilesReference        string         `gorm:"profiles_reference" json:"profiles_reference"`
	Pool                     string         `gorm:"column:pool" json:"pool"`
	Protocol                 string         `gorm:"column:protocol" json:"protocol"`
	Rules                    string         `gorm:"column:rules" json:"rules"`
	Persist                  string         `gorm:"column:persist" json:"persist"`
	TrafficGroup             string         `gorm:"column:traffic_group" json:"traffic_group"`
	PoolNodes                []*TF5PoolNode `gorm:"-" json:"pool_nodes"`
}

func (TF5Vs) TableName() string {
	return "t_f5_vs"
}
func (t *TF5Vs) BulkCreate(data []*TF5Vs) error {
	if e := database.DB.Create(&data).Error; e != nil {
		return fmt.Errorf("批量创建F5vs失败, err: %w", e)
	}
	return nil
}
func (t *TF5Vs) FindByDeviceId(deviceId int) ([]*TF5Vs, error) {
	result := make([]*TF5Vs, 0)
	if err := database.DB.Where("device_id = ?", deviceId).Find(&result).Error; err != nil {
		return nil, fmt.Errorf("获取F5vs信息失败, device_id: %d, err: %w", deviceId, err)
	}
	return result, nil
}
func (t *TF5Vs) FirstByDestination(destination string) error {
	if e := database.DB.Where("destination = ?").First(t).Error; errors.Is(e, gorm.ErrRecordNotFound) {
		return fmt.Errorf("对应的F5vs不存在, 目标地址: %s", destination)
	} else if e != nil {
		return fmt.Errorf("获取F5vs失败, 目标地址: %s, err: %w", destination, e)
	}
	return nil
}
func (t *TF5Vs) AfterFind(tx *gorm.DB) (err error) {
	device := TNLBDevice{}
	if err = device.QueryById(t.DeviceId); err == nil {
		t.Device = device.Name
	}
	database.DB.Where("device_id = ? and pool_name = ?", t.DeviceId, t.Pool).Find(&t.PoolNodes)
	return
}

type TF5Pool struct {
	Id        int    `gorm:"primary_key" json:"id"`
	DeviceId  int    `gorm:"column:device_id" json:"device_id"`
	Name      string `gorm:"column:name" json:"name"`
	Partition string `json:"partition"`
	Monitor   string `json:"monitor"`
}

func (TF5Pool) TableName() string {
	return "t_f5_pool"
}

type TF5PoolNode struct {
	Id        int    `gorm:"primary_key" json:"id"`
	DeviceId  int    `gorm:"column:device_id" json:"device_id"`
	PoolName  string `gorm:"column:pool_name" json:"pool_name"`
	Name      string `gorm:"column:name" json:"name"`
	Partition string `gorm:"column:partition" json:"partition"`
	State     string `gorm:"column:state" json:"state"`
}

func (TF5PoolNode) TableName() string {
	return "t_f5_pool_node"
}

type TPolicyLog struct {
	BaseModel
	DeviceId   int    `gorm:"column:device_id" json:"device_id"`
	Operator   string `gorm:"column:operator" json:"operator"`
	Content    string `gorm:"column:content" json:"content"`
	Status     string `gorm:"column:status" json:"status"`
	DeviceType string `gorm:"column:device_type" json:"device_type"`
}

func (t *TPolicyLog) LastByDeviceIdAndType(deviceId int, deviceType string) error {
	if e := database.DB.Where("device_id = ? and device_type = ?", deviceId, deviceType).Last(t).Error; errors.Is(e, gorm.ErrRecordNotFound) {
		return fmt.Errorf("设备操作日志不存在")
	} else if e != nil {
		return fmt.Errorf("获取设备操作日志失败, 设备ID: %d, err: %w", deviceId, e)
	}
	return nil
}

type TDevicePolicyHitCount struct {
	BaseModel
	DeviceId       int    `gorm:"column:device_id" json:"device_id"`
	Name           string `gorm:"column:name" json:"name"`
	Source         string `gorm:"column:source" json:"source"`
	Destination    string `gorm:"column:destination" json:"destination"`
	Protocol       string `gorm:"column:protocol" json:"protocol"`
	Port           string `gorm:"column:port" json:"port"`
	BeforeHitCount int    `gorm:"column:before_hit_count" json:"before_hit_count"`
	HitCount       int    `gorm:"column:hit_count" json:"hit_count"`
	Command        string `gorm:"column:command" json:"command"`
	State          int    `gorm:"column:state" json:"state"`
}

func (TDevicePolicyHitCount) TableName() string {
	return "t_device_policy_hit_count"
}
