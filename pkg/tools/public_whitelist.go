package tools

import (
	"fmt"
	"go.uber.org/zap"
	"netops/database"
	"netops/model"
	"netops/pkg/device"
	"netops/pkg/subnet"
	"netops/utils"
	"strings"
)

const (
	F5Policy  = "f5-policy"
	NatPolicy = "nat-policy"
)

type PublicWhitelistResult struct {
	RegionId int    `json:"region_id"`
	VsId     int    `json:"vs_id"`
	Region   string `json:"region"`
	Ip       string `json:"address"`
	Port     string `json:"port"`
	Host     string `json:"host"`
	Vs       string `json:"vs"`
	Pool     string `json:"pool"`
	Protocol string `json:"protocol"`
	UK       string `json:"uk"`
}

func NewPublicWhitelistHandler(id int) *publicWhitelistHandler {
	p := &publicWhitelistHandler{id: id}
	p.init()
	return p
}

type publicWhitelistHandler struct {
	base
	id   int
	data *model.TPublicWhitelist
}

func (h *publicWhitelistHandler) init() {
	data := model.TPublicWhitelist{}
	if e := data.FirstById(h.id); e != nil {
		h.error = e
	}
	h.data = &data
	h.initRegion()
}
func (h *publicWhitelistHandler) initRegion() {
	region := model.TRegion{}
	if e := region.FirstById(h.data.RegionId); e != nil {
		h.error = e
		return
	}
	h.region = &region
}

// Parse 解析公网白名单暴露面
func (h *publicWhitelistHandler) Parse() ([]*PublicWhitelistResult, error) {
	if h.error != nil {
		return nil, h.error
	}
	d := model.TFirewallDevice{}
	if e := d.FirstById(h.data.DeviceId); e != nil {
		return nil, e
	}
	dt := model.TDeviceType{}
	if e := dt.FirstById(d.DeviceTypeId); e != nil {
		return nil, e
	}
	// 根据类型调用对应的方法
	switch h.data.Type {
	case F5Policy:
		return h.f5PolicyParse()
	case NatPolicy:
		switch strings.ToLower(dt.Name) {
		case "asa":
			return h.asaNatPolicyParse()
		case "srx":
			return h.srxNatPolicyParse(d)
		}
	}
	return nil, fmt.Errorf("不支持的类型, type: %s", h.data.Type)
}

// srx策略解析
func (h *publicWhitelistHandler) srxNatPolicyParse(d model.TFirewallDevice) ([]*PublicWhitelistResult, error) {
	l := zap.L().With(zap.Int("id", h.data.Id), zap.String("func", "srxNatPolicy"))
	l.Debug("实例化设备handler--->")
	parser, err := device.NewDeviceHandler(h.data.DeviceId)
	if err != nil {
		return nil, err
	}
	l.Debug("获取设备pool-------->")
	natPools := make([]*model.TDeviceNatPool, 0)
	if err := database.DB.Where("device_id = ? and nat_type = ?", h.data.DeviceId, "destination").Find(&natPools).Error; err != nil {
		return nil, fmt.Errorf("根据设备ID<%d>获取入向nat pool异常: <%s>", h.data.DeviceId, err.Error())
	}

	poolNames := make([]string, 0)
	// 循环natPools
	for _, nat := range natPools {
		info := &model.TTaskInfo{
			Direction: "inside",
			Src:       "0.0.0.0/0",
			Dst:       nat.Address,
			DPort:     nat.Port,
			Protocol:  "tcp",
		}
		l.Debug("查找策略信息", zap.String("permit_policy_name", d.InPermitPolicyName), zap.Any("info", info))
		// 如果permit策略不为空，说明是要反向查询，是permit兜底，做deny
		if d.InPermitPolicyName != "" {
			// 匹配所有策略，如果匹配到permit，说明没做deny，则说明暴露了
			policies, e := parser.SearchAll("0.0.0.0/0", nat.Address, nat.Port)
			if e != nil {
				return nil, e
			}
			l.Debug("策略信息", zap.String("address", nat.Address), zap.String("port", nat.Port))
			l.Debug("匹配到的策略数量", zap.Int("total", len(policies)))
			if len(policies) > 0 {
				l.Debug("第一条策略", zap.Any("policy", policies[0]))
				if policies[0].Name == d.InPermitPolicyName {
					poolNames = append(poolNames, nat.Name)
				}
			}
		} else {
			if p, e := parser.Search(info); e != nil {
				return nil, e
			} else if p != nil {
				poolNames = append(poolNames, nat.Name)
			}
		}
	}
	l.Debug("显示poolNames", zap.Strings("pool_names", poolNames))
	nats := make([]*model.TDeviceSrxNat, 0)
	if err := database.DB.Where("device_id = ? and pool in ?", h.data.DeviceId, poolNames).Find(&nats).Error; err != nil {
		return nil, fmt.Errorf("根据设备ID<%d>和natNames<%+v>获取nat异常: <%s>", h.data.DeviceId, poolNames, err.Error())
	}
	result := make([]*PublicWhitelistResult, 0)
	for _, v := range nats {
		result = append(result, &PublicWhitelistResult{
			RegionId: h.region.Id,
			Region:   h.region.Name,
			Ip:       v.Dst,
			Port:     v.DstPort,
			Protocol: v.Protocol,
			Pool:     v.Pool,
			UK:       fmt.Sprintf("%s:%s", v.Dst, v.DstPort),
		})
	}
	return result, nil
}

// srx

// 上海沙箱解析
func (h *publicWhitelistHandler) asaNatPolicyParse() ([]*PublicWhitelistResult, error) {
	l := zap.L().With(zap.Int("id", h.data.Id), zap.String("func", "asaNatPolicyParse"))
	l.Debug("1. 获取设备入向any的策略，拿到访问目标-------------->")
	// 1. 获取设备策略源 入向 源是0.0.0.0/0 协议是 ip udp tcp的策略
	policies := make([]*model.TDevicePolicy, 0)
	if err := database.DB.Where("device_id = ? and direction = ? and action = ? and src = ? and dst != ? and protocol in ?",
		h.data.DeviceId, "inside", "permit", "0.0.0.0/0", "0.0.0.0/0", []string{"ip", "udp", "tcp"}).Find(&policies).Error; err != nil {
		return nil, fmt.Errorf("根据设备ID<%d>获取设备策略异常: <%s>", h.data.DeviceId, err.Error())
	}
	anyDst := make([]string, 0)
	for _, v := range policies {
		anyDst = append(anyDst, v.Dst)
	}
	l.Debug("2. 找到所有开通any的目标地址的nat信息--------->")
	// 2. 循环这些策略，从nat映射表中寻找对应的nat信息
	nats := make([]*model.TDeviceNat, 0)
	if err := database.DB.Where("device_id = ? and direction = ? and network in ?", h.data.DeviceId, "inside", anyDst).Find(&nats).Error; err != nil {
		return nil, fmt.Errorf("根据设备ID<%d>获取设备Nat异常: <%s>", h.data.DeviceId, err.Error())
	}
	l.Debug("3. 循环nat信息，通过组装源0.0.0.0、目、端、映射地址、映射端口调用search方法查询策略是否开通----------->")
	parser, err := device.NewDeviceHandler(h.data.DeviceId)
	if err != nil {
		return nil, err
	}
	result := make([]*PublicWhitelistResult, 0)
	for _, nat := range nats {
		// 再对nat的映射地址进行过滤，只要公网地址
		if net, err := subnet.GetIPNet(nat.Static); err != nil {
			return nil, err
		} else if net.NetType != "公网" {
			continue
		}
		info := &model.TTaskInfo{
			Src:        "0.0.0.0/0",
			Dst:        nat.Network,
			DPort:      nat.NetworkPort,
			StaticIp:   nat.Static,
			StaticPort: nat.StaticPort,
			Protocol:   nat.Protocol,
			Direction:  "inside",
		}
		p, e := parser.Search(info)
		if e != nil {
			return nil, e
		}
		if p != nil {
			result = append(result, &PublicWhitelistResult{
				RegionId: h.region.Id,
				Region:   h.region.Name,
				Ip:       nat.Static,
				Port:     nat.StaticPort,
				Host:     nat.Network,
				Protocol: nat.Protocol,
				UK:       fmt.Sprintf("%s:%s", nat.Static, nat.StaticPort),
			})
		}
	}
	l.Debug("4. 解析完成-----------------!")
	return result, nil
}

// F5到策略解析
func (h *publicWhitelistHandler) f5PolicyParse() ([]*PublicWhitelistResult, error) {
	l := zap.L().With(zap.Int("id", h.data.Id), zap.String("func", "f5PolicyParse"))
	l.Debug("1. 获取F5所有vs信息-------------->")
	// 1. 获取F5设备所有destination，要区分出地址和端口、协议
	vss, e := new(model.TF5Vs).FindByDeviceId(h.data.NlbDeviceId)
	if e != nil {
		return nil, e
	}
	l.Debug("2. 循环F5地址和端口，查找对应的入向any策略-------->")
	//2. 循环这些地址和端口，查询对应的防火墙的设备策略，查询条件是入向，源是0.0.0.0/0 目标是此地址和端口、协议
	parser, err := device.NewDeviceHandler(h.data.DeviceId)
	if err != nil {
		return nil, err
	}
	result := make([]*PublicWhitelistResult, 0)
	for _, vs := range vss {
		var dstPorts []string
		// 如果目标地址包含的.大于1，说明是V4地址
		if strings.Count(vs.Destination, ".") > 1 {
			dstPorts = strings.Split(vs.Destination, ":")
		} else {
			dstPorts = strings.Split(vs.Destination, ".")
		}
		if len(dstPorts) < 2 {
			continue
		}
		dst := dstPorts[0]
		port := dstPorts[1]
		if dst == "0.0.0.0" {
			continue
		}
		info := &model.TTaskInfo{
			Direction: "inside",
			Src:       "0.0.0.0/0",
			Dst:       utils.AddMask(dst),
			DPort:     port,
			Protocol:  vs.Protocol,
		}
		policy, e := parser.Search(info)
		if e != nil {
			return nil, e
		}
		if policy != nil {
			hosts := make([]string, 0)
			for _, node := range vs.PoolNodes {
				hosts = append(hosts, fmt.Sprintf("%s:%s", node.Name, node.State))
			}
			result = append(result, &PublicWhitelistResult{
				RegionId: h.region.Id,
				VsId:     vs.Id,
				Region:   h.region.Name,
				Ip:       dst,
				Port:     port,
				Vs:       vs.Name,
				Pool:     vs.Pool,
				Host:     strings.Join(hosts, ","),
				UK:       fmt.Sprintf("%s:%s", dst, port),
			})
		}
	}
	l.Debug("3. 解析完成-----------------!")
	return result, nil
}
