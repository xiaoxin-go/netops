package device

import (
	"fmt"
	"netops/model"
	"strings"
)

type Handler interface {
	Error() error
	Backup() error
	ParseConfig()
	ParseInvalidPolicy()
	Search(info *model.TTaskInfo) (*model.TDevicePolicy, error)
	SearchAll(src, dst, port string) ([]*model.TDevicePolicy, error)
	GetCommand(devicePolicy *model.TDevicePolicy) string
	GeneCommand(jiraKey string, info *model.TTaskInfo) (string, error)
	CheckNat(info *model.TTaskInfo) error
	SearchNat(info *model.TTaskInfo) *model.TDeviceNat
	init()
	GeneShowCmd(groupName, subnet string) string             // 生成黑名单任务命令，subnet必须是带掩码的IP地址
	GeneDenyCmd(groupName, subnet string) string             // 生成黑名单任务命令，subnet必须是带掩码的IP地址
	GenePermitCmd(groupNames []string, subnet string) string // 生成黑名单任务命令，subnet必须是带掩码的IP地址
	GeneCreateGroupCmd(ipType, policyName, groupName string) (result string)
}

func NewDeviceHandler(deviceId int) (result Handler, err error) {
	device := model.TFirewallDevice{}
	if e := device.FirstById(deviceId); e != nil {
		return nil, e
	}
	if device.Enabled == 0 {
		return nil, fmt.Errorf("设备已禁用, 设备名: %s", device.Name)
	}
	deviceType := model.TDeviceType{}
	if e := deviceType.FirstById(device.DeviceTypeId); e != nil {
		return nil, e
	}
	switch strings.ToLower(deviceType.Name) {
	case "asa":
		result = NewAsaHandler(deviceId)
	case "srx":
		result = NewSrxHandler(deviceId)
	case "h3c":
		result = NewH3cHandler(deviceId)
	case "huawei":
		result = NewHuaWeiHandler(deviceId)
	default:
		return nil, fmt.Errorf("暂不支持当前类型的设备, 设备类型: %s", deviceType.Name)
	}
	result.init()
	return
}
