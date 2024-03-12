package device

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"netops/conf"
	"netops/database"
	net_api2 "netops/grpc_client/protobuf/net_api"
	"netops/model"
	"netops/utils"
	"os"
	"strconv"
	"strings"
)

func NewAsaHandler(deviceId int) *AsaHandler {
	result := &AsaHandler{}
	result.DeviceId = deviceId
	return result
}

type AsaHandler struct {
	AsaParse
}

func (a *AsaHandler) init() {
	a.backupCommand = "show run"
	a.base.init()
}

func (a *AsaHandler) ParseConfig() {
	a.addLog("<-------开始解析设备策略------->")
	if e := a.parse(); e != nil {
		a.operateLog.Status = "failed"
		a.addLog(e.Error())
		_ = a.device.UpdateParseStatus(ParseStatusFailed)
		return
	}
	_ = a.device.UpdateParseStatus(ParseStatusSuccess)
	a.addLog("<-------解析策略完成------->")
}
func (a *AsaHandler) Search(info *model.TTaskInfo) (*model.TDevicePolicy, error) {
	if a.error != nil {
		return nil, a.error
	}
	l := zap.L().With(zap.String("func", "Search"), zap.Int("info_id", info.Id))
	l.Debug("策略查询--->", zap.Any("device", a.device), zap.Any("info", info))
	var (
		result    *model.TDevicePolicy
		db        *gorm.DB
		portNames []string
		err       error
	)
	if info.StaticIp != "" {
		l.Debug("若需要做nat，先校验nat是否存在--->")
		if nat := a.SearchNat(info); nat == nil {
			l.Debug("未匹配到nat策略", zap.Any("nat", nat))
			return nil, nil
		}
	}
	l.Info("2. 根据源目地址模糊匹配符合条件的策略--->")
	// 如果是办公网需求，则另加查询条件，办公网对比规则, 源地址组是办公网，目标地址包含，协议相同
	db = database.DB.Where("device_id = ? and dst like ? and protocol in ? and direction = ? and action = ?",
		a.DeviceId, "%"+info.Dst+"%", []string{info.Protocol, "ip"}, info.Direction, "permit")
	if info.Direction == "inside" && (info.Src == conf.BanGongWang || info.Src == conf.BanGongWangV6) {
		db = db.Where("src_group = ?", info.Src)
	} else {
		// 正常逻辑校验规则，源地址包含，目标地址包含，协议相同，方向相同，并且已开通的
		db = db.Where("src like ?", "%"+info.Src+"%")
	}
	if info.Protocol != "ip" {
		l.Debug("根据端口和协议获取端口所在的组--------->")
		if portNames, err = a.getPortName(info.DPort); err != nil {
			l.Error(err.Error())
			return nil, err
		}
		l.Debug("根据port和port_names过滤", zap.String("port", info.DPort), zap.Any("port_names", portNames))
		db = db.Where("port = ? or port = ? or port_group in ?", "any", info.DPort, portNames)
	}
	policies := make([]*model.TDevicePolicy, 0)
	db = db.Find(&policies)
	if db.Error != nil {
		return nil, fmt.Errorf("查询策略表失败, err: %w", db.Error)
	}
	if len(policies) > 0 {
		result = policies[0]
		l.Debug("匹配到的策略--->", zap.Any("result", result))
		return result, nil
	} else {
		l.Debug("3. 根据基本信息未匹配到相应的策略，开始进行网段的匹配------->")
		// 如果没有匹配的策略，则获取所有地址组，根据地址组来查询
		// 先获取源地址为网段的策略信息
		l.Debug("先根据基本条件进行过滤------------>")
		subnetPolicies := make([]*model.TDevicePolicy, 0)
		db = database.DB.Where("action = ? and device_id = ? and direction = ? and protocol in ?",
			"permit", a.DeviceId, info.Direction, []string{info.Protocol, "ip"})
		if info.Protocol != "ip" {
			db = db.Where("port = ? or port = ? or port_group in ?", "any", info.DPort, portNames)
		}
		if e1 := db.Find(&subnetPolicies).Error; e1 != nil {
			return nil, fmt.Errorf("查询策略地址组失败, err: %w", e1)
		}
		// 根据源目地址找到符合条件的策略
		l.Debug("4. 根据源目地址从策略表中匹配策略--------->")
		policies = a.getSubnetPolicy(subnetPolicies, info.Src, info.Dst)
		if len(policies) > 0 {
			// 办公网直接取第一条
			if info.Src == conf.BanGongWang || info.Src == conf.BanGongWangV6 {
				result = policies[0]
			} else {
				// 匹配出最小的策略
				l.Debug("根据匹配到的网段策略, 获取范围最小的策略信息-------->")
				result = a.getPolicy(policies, info.Src, info.Dst)
				l.Debug("匹配到的策略--->", zap.Any("result", result))
			}
			return result, nil
		}
	}
	l.Info("<-------策略查询结束------->")
	return nil, nil
}
func (a *AsaHandler) GetCommand(dp *model.TDevicePolicy) string {
	return dp.Command
}

// 生成端口策略
func (a *AsaHandler) genePortPolicy(groupName string, info *model.TTaskInfo) (result string) {
	// 生成端口组配置------------->
	portCommand := fmt.Sprintf("object-group service %s %s", groupName, info.Protocol)
	for _, p := range strings.Split(info.DPort, ",") {
		rp, _ := utils.ParseRangePort(p)
		if rp.Start == rp.End {
			portCommand += fmt.Sprintf("\n port-object eq %s", p)
		} else {
			portCommand += fmt.Sprintf("\n port-object range %d %d", rp.Start, rp.End)
		}
	}
	return portCommand
}

// 生成源地址策略
func (a *AsaHandler) geneSrcPolicy(groupName string, info *model.TTaskInfo) string {
	srcGroupCommand := fmt.Sprintf("object-group network %s\n", groupName)
	for _, v := range strings.Split(info.Src, ",") {
		ips, mask := ipMask(v)
		srcGroupCommand += fmt.Sprintf(" network-object %s %s\n", ips, mask)
	}
	return srcGroupCommand
}

// 生成目标地址策略
func (a *AsaHandler) geneDstPolicy(groupName string, info *model.TTaskInfo) string {
	// 生成创建目标IP地址组
	dstGroupCmd := fmt.Sprintf("object-group network %s\n", groupName)
	for _, v := range strings.Split(info.Dst, ",") {
		ip, mask := ipMask(v)
		dstGroupCmd += fmt.Sprintf(" network-object %s %s\n", ip, mask)
	}
	return dstGroupCmd
}

// 生成nat策略
func (a *AsaHandler) geneNatPolicy(groupName string, info *model.TTaskInfo, srcGroupName, dstGroupName string) (result string) {
	// object network neiwang\n subnet 172.17.0.0 255.255.0.0
	// object network 172.17.25.49-22 nat (inside,outside) static 116.228.151.3 service tcp 1022 1022
	natCmd := ""
	if info.StaticIp != "" {
		if nat := a.SearchNat(info); nat == nil {
			zap.L().Info("生成nat策略------------------------------->")
			static, mask := ipv4Mask(info.StaticIp)
			staticNetwork := fmt.Sprintf("%s-%s-%s", groupName, static, info.StaticPort)
			natCmd += fmt.Sprintf("object network %s\n subnet %s %s\n", staticNetwork, static, mask)
			dst, mask := ipv4Mask(info.Dst)
			dstNetwork := fmt.Sprintf("%s-%s-%s", groupName, dst, info.DPort)
			natCmd += fmt.Sprintf("object network %s\n subnet %s %s\n", dstNetwork, dst, mask)
			natCmd += fmt.Sprintf("object network %s \nnat (inside,outside) static %s service %s %s %s",
				dstNetwork, staticNetwork, info.Protocol, info.DPort, info.StaticPort)
		} else {
			zap.L().Info(fmt.Sprintf("nat策略已存在: <%+v>", nat))
			info.ExistsConfig = nat.Command
		}
	}
	if info.Direction == "outside" {
		// 如果是CN2类型，则需要做出向nat
		if strings.Contains(info.OutboundNetworkType, "CN2") {
			// 1. 先校验info.PoolName(nat需要映射成的名称)是否存在，如果不存在则不校验
			if info.PoolName == "" {
				a.error = fmt.Errorf("出向<%s>类型，未定义nat映射信息", info.OutboundNetworkType)
				return
			}
			// 2. 查询设备nat策略，匹配nat是否已经存在
			if nat := a.SearchNat(info); nat != nil {
				zap.L().Info(fmt.Sprintf("nat策略已存在: <%+v>", nat))
				info.ExistsConfig = nat.Command
			} else {
				// 如果nat不存在，则需要生成新的nat命令
				// nat (inside,outside) source dynamic dnat172.17.205.62 dnat10.235.227.195 destination static dnat10.233.89.71 dnat10.233.89.71
				// 生成命令是需要使用地址组，所以需要先校验地址组是否存在，不存在要创建，存在引用之前的
				//srcName := fmt.Sprintf("dnat%s", strings.Split(info.Src, "/")[0])
				//var count int64
				//if e1 := database.DB.Model(&model.TDeviceNatNetwork{}).Where("name = ?", srcName).Count(&count).Error; e1 != nil {
				//	a.error = fmt.Errorf("根据src name<%s>获取nat network失败: <%s>", srcName, e1.Error())
				//	zap.L().Error(a.error.Error())
				//	return
				//}
				//if count == 0 {
				//	src, mask := ipv4Mask(info.Src)
				//	natCmd += fmt.Sprintf("object network %s\n subnet %s %s\n", srcName, src, mask)
				//}
				// 校验目标地址组是否存在
				//count = 0
				//dstName := fmt.Sprintf("dnat%s", strings.Split(info.Dst, "/")[0])
				//if e1 := database.DB.Model(&model.TDeviceNatNetwork{}).Where("name = ?", dstName).Count(&count).Error; e1 != nil {
				//	a.error = fmt.Errorf("根据dst name<%s>获取nat network失败: <%s>", dstName, e1.Error())
				//	zap.L().Error(a.error.Error())
				//	return
				//}
				//if count == 0 {
				//	dst, mask := ipv4Mask(info.Dst)
				//	natCmd += fmt.Sprintf("object network %s\n subnet %s %s\n", dstName, dst, mask)
				//}
				natCmd += fmt.Sprintf("nat (inside,outside) source dynamic %s %s destination static %s %s",
					srcGroupName, info.PoolName, dstGroupName, dstGroupName)
			}
		}
	}
	return natCmd
}

// SearchNat 查询nat策略是否存在
func (a *AsaHandler) SearchNat(info *model.TTaskInfo) *model.TDeviceNat {
	l := zap.L().With(zap.Int("task_id", info.Id))
	l.Info("校验nat信息是否存在------->")
	result := &model.TDeviceNat{}
	// 如果是入向，则根据目标地址和映射地址去匹配network和static、目标端口和映射端口的nat配置是否存在
	if info.Direction == "inside" {
		dps := []string{"any", info.DPort}
		sps := []string{"any", info.StaticPort}
		ps := []string{"ip", info.Protocol}
		if err := database.DB.Where("device_id = ? and direction = ? and network = ? and network_port in ? and protocol in ? and static = ? and static_port in ?",
			a.DeviceId, info.Direction, info.Dst, dps, ps, info.StaticIp, sps).First(result).Error; errors.Is(err, gorm.ErrRecordNotFound) { // 如果为空则为空
			return nil
		} else if err != nil { // 否则获取失败报个错
			a.error = fmt.Errorf("获取nat配置信息异常: %s", err.Error())
			return nil
		}
		return result
	} else {
		// 出向则匹配源地址、映射地址名称、目标地址是否已做nat（出向由于是多对多，所以还需要额外匹配网段是否已映射）
		if err := database.DB.Where("device_id = ? and direction = ? and network like ? and static_group = ? and destination like ?",
			a.DeviceId, info.Direction, "%"+info.Src+"%", info.PoolName, "%"+info.Dst+"%").First(result).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			a.error = fmt.Errorf("获取nat配置信息异常: %s", err.Error())
			return nil
		}
		// 如果单个没有查询到，则需要匹配网段是否已做nat
		subnetNats := make([]*model.TDeviceNat, 0)
		if err := database.DB.Where("device_id = ? and direction = ? and static_group = ?", a.DeviceId, "outside", info.PoolName).Find(&subnetNats).Error; err != nil {
			a.error = fmt.Errorf("批量获取nat配置信息异常: %s", err.Error())
			return nil
		}
		l.Info("4. 根据源目地址从策略表中匹配策略--------->")
		nats := a.getSubnetNat(subnetNats, info.Src, info.Dst)
		if len(nats) > 0 {
			// 匹配出最小的策略
			l.Info("匹配最小nat信息-------->")
			result = a.getNat(nats, info.Src, info.Dst)
			l.Info(fmt.Sprintf("匹配到的Nat: <%+v>", result))
		}
	}
	return nil
}

// 生成策略
func (a *AsaHandler) genePolicy(srcGroupName, dstGroupName, portGroupName string, info *model.TTaskInfo) string {
	var accessListType string
	switch info.Direction {
	case "inside":
		accessListType = a.device.InPolicy
	case "outside":
		accessListType = a.device.OutPolicy
	}
	policyCommand := fmt.Sprintf("access-list %s line 10 extended permit %s object-group %s "+
		"object-group %s object-group %s", accessListType, info.Protocol, srcGroupName, dstGroupName, portGroupName)
	return policyCommand
}

// GeneCommand 生成命令
func (a *AsaHandler) GeneCommand(jiraKey string, info *model.TTaskInfo) (string, error) {
	if a.error != nil {
		return "", a.error
	}
	l := zap.L().With(zap.String("func", "GeneCommand"), zap.String("jira_key", jiraKey), zap.Int("info_id", info.Id))
	l.Info("生成策略", zap.Any("info", info))
	var (
		srcGroupName  string
		dstGroupName  string
		portGroupName string
		commands      = make([]string, 0)
		groupName     = a.groupName(jiraKey, info)
	)

	l.Info("1. 生成端口组策略--->")
	portName := a.getDevicePortName(info.DPort, strings.ToLower(info.Protocol))
	if portName != "" {
		portGroupName = portName
	} else {
		portGroupName = a.genePortGroupName(groupName)
		commands = append(commands, a.genePortPolicy(portGroupName, info))
	}

	l.Info("2. 生成源地址组策略--->")
	// 如果办公网，地址组直接设置为办公网
	if info.Src == conf.BanGongWang || info.Src == conf.BanGongWangV6 {
		srcGroupName = info.Src
	} else if srcAddressGroup := a.getAddressGroup(info.Src); srcAddressGroup != nil {
		srcGroupName = srcAddressGroup.Name
	} else {
		srcGroupName = a.geneSrcGroupName(groupName)
		commands = append(commands, a.geneSrcPolicy(srcGroupName, info))
	}

	l.Info("3. 生成目标地址组策略--->")
	dstAddressGroup := a.getAddressGroup(info.Dst)
	if dstAddressGroup != nil {
		dstGroupName = dstAddressGroup.Name
	} else {
		dstGroupName = a.geneDstGroupName(groupName)
		commands = append(commands, a.geneDstPolicy(dstGroupName, info))
	}

	l.Info("4. 生成策略--->")
	commands = append(commands, a.genePolicy(srcGroupName, dstGroupName, portGroupName, info))

	l.Info("5. 生成nat策略--->")
	commands = append(commands, a.geneNatPolicy(groupName, info, srcGroupName, dstGroupName))
	return strings.Join(commands, "\n"), nil
}

// 返回组名
func (a *AsaHandler) getPortName(port string) (results []string, err error) {
	p, err := utils.ParseRangePort(port)
	if err != nil {
		return
	}
	ports := make([]model.TDevicePort, 0)
	db := database.DB.Where("device_id = ? and start <= ? and end >= ?", a.DeviceId, p.Start, p.End).Find(&ports)
	if db.Error != nil {
		return nil, fmt.Errorf("根据port<%s>获取端口组异常: <%s>", port, db.Error.Error())
	}
	results = make([]string, 0)
	for _, p := range ports {
		results = append(results, p.Name)
	}
	results = append(results, "any")
	return results, nil
}

// CheckNat 校验NAT端口是否被占用
func (a *AsaHandler) CheckNat(info *model.TTaskInfo) (err error) {
	// 一个公网端口只能映射一个内网端口，但一个内网端口可以被多个公网端口所映射
	// static对应公网地址
	// network对应内网地址
	// src_port是内网端口，dst_port是公网端口
	nat := &model.TDeviceNat{}
	e := database.DB.Where("device_id = ? and static = ? and static_port = ? and protocol = ? and (network != ? or network_port != ?)",
		a.DeviceId, info.StaticIp, info.StaticPort, info.Protocol, info.Dst, info.DPort).First(nat).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return
	}
	if e != nil {
		err = fmt.Errorf("获取nat配置信息异常: %w", e)
		return
	}
	err = fmt.Errorf("策略<%s-%s>端口已映射到<%s-%s>", info.StaticIp, info.StaticPort, nat.Network, nat.NetworkPort)
	return
}

type AsaParse struct {
	base
	accessListText  string
	objectGroupText string
	serviceText     string
	objectText      string
	natText         string
}

func (a *AsaParse) parse() error {
	if e := a.device.UpdateParseStatus(ParseStatusInit); e != nil {
		return e
	}
	a.addLog("1. 获取配置信息--->")
	if e := a.getConfig(); e != nil {
		return e
	}

	a.addLog("2. 解析service--->")
	services := a.parseServiceText()
	if e := a.parsePort(services); e != nil {
		return e
	}

	a.addLog("3. 解析object--->")
	objects := a.parseObjectText()
	if e := a.parseObject(objects); e != nil {
		return e
	}

	a.addLog("4. 解析object-group---->")
	objectGroups := a.parseObjectGroupText(objects)
	if e := a.parseGroup(objectGroups); e != nil {
		return e
	}

	a.addLog("5. 解析access-list---->")
	accessLists := a.parseAccessListText()
	if e := a.parseAccessList(accessLists, objectGroups, objects, services); e != nil {
		return e
	}

	a.addLog("6. 解析nat---->")
	nats := a.parseNatText()
	if e := a.parseNat(nats, objectGroups, objects, services); e != nil {
		return e
	}

	//a.addLog("7. 解析黑名单地址组--->")
	//a.parseBlacklistGroupAddress(objectGroups)
	return nil
}

func (a *AsaParse) getConfig1() {
	fmt.Println("获取设备策略-------------->")
	fb, err := os.ReadFile("C:\\Users\\34607\\.wind\\profiles\\default.v10\\terminal\\logs\\南京\\nj-asa-access-list.log")
	if err != nil {
		a.error = err
		return
	}
	a.accessListText = string(fb)

	fb, err = os.ReadFile("C:\\Users\\34607\\.wind\\profiles\\default.v10\\terminal\\logs\\南京\\nj-asa-object-group.log")
	if err != nil {
		a.error = err
		return
	}
	a.objectGroupText = string(fb)

	fb, err = os.ReadFile("C:\\Users\\34607\\.wind\\profiles\\default.v10\\terminal\\logs\\南京\\nj-asa-object.log")
	if err != nil {
		a.error = err
		return
	}
	a.objectText = string(fb)

	fb, err = os.ReadFile("C:\\Users\\34607\\.wind\\profiles\\default.v10\\terminal\\logs\\南京\\nj-asa-service.log")
	if err != nil {
		a.error = err
		return
	}
	a.serviceText = string(fb)

	fb, err = os.ReadFile("C:\\Users\\34607\\.wind\\profiles\\default.v10\\terminal\\logs\\南京\\nj-asa-nat.log")
	if err != nil {
		a.error = err
		return
	}
	a.natText = string(fb)
}
func (a *AsaParse) getConfig() error {
	commands := []*net_api2.Command{
		{Id: 1, Cmd: "show run access-list"},
		{Id: 2, Cmd: "show running-config object-group network"},
		{Id: 3, Cmd: "show running-config object-group service"},
		{Id: 4, Cmd: "show running-config object network"},
		{Id: 5, Cmd: "show run nat"},
	}
	result, err := a.send(commands)
	if err != nil {
		return err
	}
	for _, v := range result {
		text := strings.ReplaceAll(v.Result, "<--- More --->\r              \r", " ") // 处理掉more的特殊字符
		switch v.Id {
		case 1:
			a.accessListText = text
		case 2:
			a.objectGroupText = text
		case 3:
			a.serviceText = text
		case 4:
			a.objectText = text
		case 5:
			a.natText = text
		}
	}
	return nil
}

func (a *AsaParse) getAccessListPolicy() string {
	commands := []*net_api2.Command{
		{Id: 1, Cmd: "show access-list"},
	}
	result, e := a.send(commands)
	if e != nil {
		return ""
	}
	return result[0].Result
}

// ParseInvalidPolicy 解析无效策略
func (a *AsaParse) ParseInvalidPolicy() {
	if a.error != nil {
		return
	}
	log := zap.L().With(zap.String("device", fmt.Sprintf("%d:%s-%s", a.DeviceId, a.device.Name, a.device.Host)))
	log.Info(fmt.Sprintf("<-------开始解析设备无效策略------->"))
	log.Info("获取accessList策略信息---------->")
	text := a.getAccessListPolicy()
	log.Info("解析accessList策略信息---------->")
	policies := a.parseAccessListPolicy(text)
	if e := a.parsePolicyHitCount(policies); e != nil {
		log.Error(e.Error())
		return
	}
	log.Info("保存策略命中次数--------->")
	if e := a.savePolicyHitCount(policies); e != nil {
		return
	}
	if a.error != nil {
		log.Error(a.error.Error())
		return
	}
	log.Info("解析无效策略完成!")
}

// 处理当前存在的策略与新的策略，是否有新的命中次数（新的访问次数）
func (a *AsaParse) parsePolicyHitCount(policies []*model.TDevicePolicyHitCount) error {
	// 1. 取出旧的策略信息
	oldPolices := make([]*model.TDevicePolicyHitCount, 0)
	if e := database.DB.Model(&model.TDevicePolicyHitCount{}).Select("name", "hit_count", "id").Where("device_id = ?", a.DeviceId).Find(&oldPolices).Error; e != nil {
		return fmt.Errorf("根据设备ID<%d>获取策略命中次数异常: <%s>", a.DeviceId, e.Error())
	}
	// 取出旧的策略和命中数
	oldNameHits := make(map[string]int, 0)
	for _, v := range oldPolices {
		oldNameHits[v.Name] = v.HitCount
	}
	// 2. 对比旧策略，设置state和before hit count
	for _, v := range policies {
		beforeHit, ok := oldNameHits[v.Name]
		if ok {
			// 命中次数为0设置为无效策略
			v.BeforeHitCount = beforeHit
		}
		// 存在则更新
		if v.HitCount <= v.BeforeHitCount {
			v.State = 0
		} else {
			v.State = 1
		}
	}
	return nil
}

// 获取策略，获取出策略信息
func (a *AsaParse) parseAccessListPolicy(text string) []*model.TDevicePolicyHitCount {
	// access-list out_inside line 1 extended permit ip 172.17.116.0 255.255.252.0 192.168.0.0 255.255.252.0 (hitcnt=2709813) 0x168d041e
	// access-list sec line 182 extended permit tcp object-group UNKNOWN-YWJS-81212-359006_SRC object-group YWJS-46842-247475_DST object-group YWJS-22614-25364_SERVICE (hitcnt=1) 0xef21f9bf
	// access-list sec line 277 extended permit tcp object-group BanGongWang host 58.213.97.134 eq www (hitcnt=413) 0x85888ad8
	// access-list sec line 405 extended permit tcp any object-group XXJS-YWJS-69434_DST eq https (hitcnt=16094360) 0x704cb892
	// 只获取permit的策略，需要获取协议-源地址组-目标地址组-端口组 访问次数，有host则直接取地址
	// 由于line会有多个，因此获取line号，只获取第一个line的策略
	// 需要创建新表
	text = strings.ReplaceAll(text, "<--- More --->\r              \r", "") // 处理掉more的特殊字符
	lineNum := ""
	bulks := make([]*model.TDevicePolicyHitCount, 0)
	for _, line := range strings.Split(text, "\r") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "access-list") || !strings.Contains(line, "hitcnt=") || strings.Contains(line, "standard") {
			continue
		}
		fmt.Println("line---------->", line)
		lines := splitLineBySpace(line)
		if lineNum == lines[3] { // 如果解析过主策略，则忽略子策略行
			continue
		} else {
			lineNum = lines[3]
		}
		// 只解析白名单策略
		if lines[5] != "permit" {
			continue
		}
		protocol, source, destination, p := a.parseSrcDstPort(lines[6 : len(lines)-2])
		// 拼接唯一策略名 src-dst-protocol-port
		name := fmt.Sprintf("%d-%s-%s-%s-%s-%s", a.DeviceId, lines[1], source, destination, protocol, p)

		item := &model.TDevicePolicyHitCount{
			Command:     line,
			DeviceId:    a.DeviceId,
			Protocol:    protocol,
			Name:        name,
			Source:      source,
			Destination: destination,
			Port:        p,
			HitCount:    a.parseHitCount(lines[len(lines)-2]),
		}
		bulks = append(bulks, item)
	}
	return bulks
}

// 从后半段策略中解析出源、目、端、协议
func (a *AsaParse) parseSrcDstPort(lines []string) (protocol, source, destination, port string) {
	// ip 172.17.116.0 255.255.252.0 192.168.0.0 255.255.252.0
	// tcp object-group UNKNOWN-YWJS-81212-359006_SRC object-group YWJS-46842-247475_DST object-group YWJS-22614-25364_SERVICE
	// tcp object-group BanGongWang host 58.213.97.134 eq www
	// tcp any object-group XXJS-YWJS-69434_DST eq https
	protocol = lines[0]
	switch {
	case isAny(lines[1]):
		source = lines[1]
	case lines[1] == "host":
		source = lines[2] + "/32"
	case lines[1] == "object-group" || lines[1] == "object":
		source = lines[2]
	default:
		source = ipMaskSimple(lines[1], lines[2])
	}
	switch {
	case isAny(lines[2]):
		destination = lines[2]
	case isAny(lines[3]):
		destination = lines[3]
	case lines[2] == "host":
		destination = lines[3] + "/32"
	case lines[3] == "host":
		destination = lines[4] + "/32"
	case lines[2] == "object-group" || lines[2] == "object":
		destination = lines[3]
	case lines[3] == "object-group" || lines[3] == "object":
		destination = lines[4]
	default:
		destination = ipMaskSimple(lines[3], lines[4])
	}
	if protocol != "ip" {
		p := lines[len(lines)-1]
		// 如果port 是https www则需要转换为443和80
		if p1, ok := PortMaps[p]; ok {
			port = p1
		} else {
			port = p
		}
	} else {
		port = "any"
	}
	return
}

// 获取策略命中数
func (a *AsaParse) parseHitCount(hit string) int {
	// (hitcnt=413)
	hitCount, _ := strconv.Atoi(hit[8 : len(hit)-1])
	return hitCount
}

type asaServicePort struct {
	start string
	end   string
}

type asaService struct {
	name     string
	protocol string
	ports    []*asaServicePort
}

func (a *AsaParse) parseServiceText() []*asaService {
	/*
		object-group service BigData-services tcp
		 port-object eq ssh
		 port-object eq telnet
		 port-object eq www
		 port-object eq https
		 port-object eq netbios-ns
		 port-object eq netbios-dgm
		 port-object eq 139
		 port-object range 4621 4631
	*/
	results := make([]*asaService, 0)
	var service *asaService
	for i, line := range strings.Split(a.serviceText, "\r") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines := splitLineBySpace(line)
		validLineM := map[string]bool{"object-group": true, "port-object": true}
		if !validLineM[lines[0]] {
			continue
		}
		// object-group service BigData-services tcp
		if lines[0] == "object-group" {
			if len(lines) < 4 {
				continue
			}
			service = &asaService{
				name:     lines[2],
				protocol: lines[3],
			}
			results = append(results, service)
			continue
		}
		if service == nil {
			a.addLog(fmt.Sprintf("无效service配置, line: %s, num: %d", line, i+1))
		}
		// port-object eq netbios-dgm
		if lines[0] == "port-object" {
			// port-object eq 139
			if lines[1] == "eq" {
				service.ports = append(service.ports, &asaServicePort{
					start: lines[2],
					end:   lines[2],
				})
				continue
			}
			// port-object range 4621 4631
			if lines[1] == "range" {
				service.ports = append(service.ports, &asaServicePort{
					start: lines[2],
					end:   lines[3],
				})
				continue
			}
		}
	}
	count := strings.Count(a.serviceText, "object-group service")
	a.addLog(fmt.Sprintf("解析到<%d>个端口组，匹配到<%d>个端口组------>", len(results), count))
	return results
}

// 解析策略端口信息，将文本解析成struct
func (a *AsaParse) parsePort(services []*asaService) error {
	results := make([]*model.TDevicePort, 0)
	for _, v := range services {
		for _, p := range v.ports {
			start, _ := portToInt(p.start)
			end, _ := portToInt(p.end)
			results = append(results, &model.TDevicePort{
				DeviceId: a.DeviceId,
				Name:     v.name,
				Protocol: v.protocol,
				Start:    start,
				End:      end,
			})
		}
	}
	return a.savePort(results)
}

type asaObject struct {
	name      string
	addresses []string
}

func (a *AsaParse) parseObjectText() []*asaObject {
	/*
		object network 命令创建对象规则
			object network object_name
		添加地址规则
		1. host {IPv4_address | IPv6_address} - 单台主机的 IPv4 或 IPv6 地址。例如， 10.1.1.1 或 2001:DB8::0DB8:800:200C:417A。
		2. subnet {IPv4_address IPv4_mask | IPv6_address/IPv6_prefix} - 网络的地址。
			对于 IPv4 子网，请在空格后添加掩码，例如， 10.0.0.0 255.0.0.0。
			对于 IPv6，请将地址和前缀作为一个整体（不带空格），例如 2001:DB8:0:CD30::/60。
		3. range start_address end_address - 地址的范围。可以指定 IPv4 或 IPv6 范围。请勿包含掩码或前缀。
		4. fqdn [v4 | v6] fully_qualified_domain_name - 完全限定域名，即主机的名称，
			例如www.example.com。指定 v4 将地址限定于 IPv4， v6 将地址限定于 IPv6。如果未指定地址类型，则假定为 IPv4。
		例如：
		object network YWJS-86965-97-161
		 subnet 58.213.97.161 255.255.255.255
		 host 172.17.179.83
		 range 172.17.120.11 172.17.120.16
	*/
	if a.error != nil {
		return nil
	}
	results := make([]*asaObject, 0)
	var object *asaObject
	for i, line := range strings.Split(a.objectText, "\r") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines := splitLineBySpace(line)
		validLineM := map[string]bool{"object": true, "subnet": true, "host": true, "range": true}
		if !validLineM[lines[0]] {
			continue
		}
		// object network YWJS-86965-97-161
		if lines[0] == "object" {
			object = &asaObject{
				name: lines[2],
			}
			results = append(results, object)
			continue
		}
		if object == nil {
			a.addLog(fmt.Sprintf("无效的object配置，text: %s, num: %d", line, i+1))
		}
		switch {
		case lines[0] == "subnet": // subnet {IPv4_address IPv4_mask | IPv6_address/IPv6_prefix} - 网络的地址。
			addr := lines[1]
			if utils.GetIpType(addr) == conf.IpTypeV4 {
				addr = ipMaskSimple(addr, lines[2])
			}
			object.addresses = append(object.addresses, addr)
		case lines[0] == "host": // host {IPv4_address | IPv6_address} - 单台主机的 IPv4 或 IPv6 地址
			addr := lines[1]
			if utils.GetIpType(addr) == conf.IpTypeV4 {
				addr = fmt.Sprintf("%s/32", addr)
			} else {
				addr = fmt.Sprintf("%s/128", addr)
			}
			object.addresses = append(object.addresses, addr)
		case lines[0] == "range": // range start_address end_address - 地址的范围。可以指定 IPv4 或 IPv6 范围。请勿包含掩码或前缀。
			// 此处只解析ipv4的范围
			if utils.GetIpType(lines[1]) == conf.IpTypeV6 {
				a.addLog(fmt.Sprintf("不支持object ipv6 range地址解析, text: %s, num: %d", line, i+1))
				continue
			}
			addresses := a.getRangeAddress(lines[1], lines[2])
			object.addresses = append(object.addresses, addresses...)
		}
	}
	a.addLog("解析到%d个object, 匹配到%d个object", len(results), strings.Count(a.objectText, "object network"))
	return results
}

func (a *AsaParse) parseObject(objects []*asaObject) error {
	results := make([]*model.TDeviceAddressGroup, 0)
	for _, v := range objects {
		for _, addr := range v.addresses {
			results = append(results, &model.TDeviceAddressGroup{
				DeviceId:    a.device.Id,
				Address:     addr,
				Name:        v.name,
				AddressType: "object",
			})
		}
	}
	return a.saveGroup(results)
}

type asaObjectGroup struct {
	name      string
	addresses []string
}

func (a *AsaParse) parseObjectGroupText(objects []*asaObject) []*asaObjectGroup {
	/*
		object-group命令解析
		1. network-object host {IPv4_address | IPv6_address} - 单台主机的 IPv4 或 IPv6 地址。例如，10.1.1.1 或 2001:DB8::0DB8:800:200C:417A
		2. network-object {IPv4_address IPv4_mask | IPv6_address/IPv6_prefix} - 网络或主机的地址。
			对于 IPv4 子网，请在空格后添加掩码，例如， 10.0.0.0 255.0.0.0。
			对于 IPv6，请将地址和前缀作为一个整体 （不带空格），例如 2001:DB8:0:CD30::/60
		3. network-object object object_name - 现有网络对象的名称。
		4. group-object object_group_name - 现有网络对象组的名称。
		object-group network dip
		 network-object host 58.213.97.1
		 network-object 172.16.129.128 255.255.255.240
		 network-object host 2409:8900:1b21:bbe:1e9f:3867:7671:6915
	*/
	objectAddressM := a.makeObjectAddressM(objects)
	results := make([]*asaObjectGroup, 0)
	var group *asaObjectGroup
	for i, line := range strings.Split(a.objectGroupText, "\r") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, ">") {
			continue
		}
		lines := splitLineBySpace(line)
		validLineM := map[string]bool{"object-group": true, "network-object": true}
		if !validLineM[lines[0]] {
			continue
		}
		// 获取组名
		if lines[0] == "object-group" {
			group = &asaObjectGroup{
				name: lines[2],
			}
			results = append(results, group)
			continue
		}
		// 获取object
		if lines[0] == "network-object" {
			if group == nil {
				a.addLog(fmt.Sprintf("无效地址组策略, line: %s, num: %d", line, i))
				continue
			}
			// network-object host 58.213.97.1, 默认为单个地址加上掩码
			switch {
			case lines[1] == "host":
				// network-object host {IPv4_address | IPv6_address} - 单台主机的 IPv4 或 IPv6 地址。例如，10.1.1.1 或 2001:DB8::0DB8:800:200C:417A
				addr := lines[2]
				if utils.GetIpType(addr) == conf.IpTypeV4 {
					addr = fmt.Sprintf("%s/32", addr)
				} else {
					addr = fmt.Sprintf("%s/128", addr)
				}
				group.addresses = append(group.addresses, addr)
			case lines[1] == "object": // network-object object object_name - 现有网络对象的名称。
				if addresses, ok := objectAddressM[lines[2]]; ok {
					group.addresses = append(group.addresses, addresses...)
				} else {
					a.addLog(fmt.Sprintf("未获取到object, object: %s", lines[2]))
				}
			default: // network-object {IPv4_address IPv4_mask | IPv6_address/IPv6_prefix} - 网络或主机的地址。
				addr := lines[1]
				if utils.GetIpType(addr) == conf.IpTypeV4 {
					addr = ipMaskSimple(addr, lines[2])
				}
				group.addresses = append(group.addresses, addr)
			}
		}
	}
	count := strings.Count(a.objectGroupText, "object-group network")
	a.addLog("解析到%d个object-group，匹配到%d个object-group------>", len(results), count)
	return results
}

// 解析地址组信息
func (a *AsaParse) parseGroup(objectGroups []*asaObjectGroup) error {
	results := make([]*model.TDeviceAddressGroup, 0)
	for _, group := range objectGroups {
		for _, addr := range group.addresses {
			results = append(results, &model.TDeviceAddressGroup{
				DeviceId:    a.device.Id,
				Name:        group.name,
				Address:     addr,
				AddressType: "object-group",
			})
		}
	}
	return a.saveGroup(results)
}

type asaPolicyParam struct {
	Type  string
	value string
}
type asaAccessList struct {
	name     string
	Type     string
	protocol string
	command  string
	src      *asaPolicyParam
	dst      *asaPolicyParam
	port     *asaPolicyParam
}

// 解析策略信息
func (a *AsaParse) parseAccessListText() []*asaAccessList {
	/*
		access-list sec extended permit tcp object-group YWJS-93506-28501-SRC object-group YWJS-46842-247475_DST object-group YWJS-22614-25364_SERVICE
		access-list sec extended deny ip object-group NJ_blacklist_2023_0 any
		access-list sec extended permit tcp object-group BanGongWang host 58.213.97.132 eq https
		access-list sec extended permit tcp host 218.94.92.234 host 58.213.97.222 eq 13389
		access-list sec extended permit tcp any6 object-group YWJS-90108-DST eq https
		access-list outbound extended permit ip 172.28.180.0 255.255.255.0 any
		access-list outbound extended permit ip 172.28.185.0 255.255.255.0 host 116.228.151.188
		access-list sec extended deny ip object-group NJ_blacklist_2021_0 any
		access-list sec extended deny tcp any any eq 445
		access-list in_outside extended permit tcp object DCN object-group YWJS-21889-24464_DST object-group tcp1080
	*/
	results := make([]*asaAccessList, 0)
	unCount := 0
	for i, line := range strings.Split(a.accessListText, "\r") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines := splitLineBySpace(line)
		// 只匹配extended的策略
		if lines[0] != "access-list" {
			continue
		}
		if lines[2] != "extended" {
			unCount += 1
			continue
		}
		if len(lines) < 7 {
			a.addLog(fmt.Sprintf("无效策略配置, line: %s, num: %d", line, i+1))
			continue
		}
		p := &asaAccessList{
			name:     lines[1],
			Type:     lines[3],
			protocol: lines[4],
			command:  line,
			src:      &asaPolicyParam{},
			dst:      &asaPolicyParam{},
			port:     &asaPolicyParam{},
		}
		// 分为几种情况来区分
		switch {
		case isAny(lines[5]): // 第一种，源为any
			p.src.value = "0.0.0.0/0"
		case lines[5] == "object-group": // 第二种，源为object-group
			p.src.Type = "object-group"
			p.src.value = lines[6]
		case lines[5] == "object": // 第三种，源为object
			p.src.Type = "object"
			p.src.value = lines[6]
		case lines[5] == "host": // // 第四种，源为host
			addr := lines[6]
			if utils.GetIpType(addr) == conf.IpTypeV4 {
				p.src.value = fmt.Sprintf("%s/32", addr)
			} else {
				p.src.value = fmt.Sprintf("%s/128", addr)
			}
		default: //第五种，源为ip mask 172.28.185.0
			if utils.GetIpType(lines[5]) == conf.IpTypeV4 {
				p.src.value = ipMaskSimple(lines[5], lines[6])
			} else {
				p.src.value = lines[5]
			}
		}
		// 解析目标地址
		switch {
		// 第一种，目标为any的
		case isAny(lines[6]) || isAny(lines[7]):
			p.dst.value = "0.0.0.0/0"
		case lines[6] == "object-group": // 第二种 目标为object-group
			p.dst.Type = "object-group"
			p.dst.value = lines[7]
		case lines[7] == "object-group":
			p.dst.Type = "object-group"
			p.dst.value = lines[8]
		case lines[6] == "object": // 第三种 目标为object
			p.dst.Type = "object"
			p.dst.value = lines[7]
		case lines[7] == "object":
			p.dst.Type = "object"
			p.dst.value = lines[8]
		case lines[6] == "host": // host 218.94.92.234
			addr := lines[7]
			if utils.GetIpType(addr) == conf.IpTypeV4 {
				p.dst.value = fmt.Sprintf("%s/32", addr)
			} else {
				p.dst.value = fmt.Sprintf("%s/128", addr)
			}
		case lines[7] == "host": // host 218.94.92.234
			addr := lines[8]
			if utils.GetIpType(addr) == conf.IpTypeV4 {
				p.dst.value = fmt.Sprintf("%s/32", addr)
			} else {
				p.dst.value = fmt.Sprintf("%s/128", addr)
			}
		default:
			/*
				这种再分两种情况，any源只有一个，所以dst下标是6和7，否则是7和8
				access-list outbound extended permit ip 172.28.185.0 255.255.255.0 172.28.185.0 255.255.255.0
				access-list outbound extended permit ip any 172.28.185.0 255.255.255.0
			*/
			addrIndex := 7
			if isAny(lines[5]) { // 如果源为any，则index则是6，mask是7
				addrIndex = 6
			}
			if utils.GetIpType(lines[addrIndex]) == conf.IpTypeV4 {
				p.dst.value = ipMaskSimple(lines[addrIndex], lines[addrIndex+1])
			} else {
				p.dst.value = lines[addrIndex]
			}
		}
		// 解析端口
		// 如果是IP协议，端口则是any
		portType := lines[len(lines)-2]
		p.port.value = lines[len(lines)-1]
		switch {
		case p.protocol == "ip":
			p.port.value = "any"
		case p.port.value == p.dst.value:
			p.port.value = "any"
		case portType == "object-group":
			p.port.Type = "object-group"
		}
		results = append(results, p)
	}
	count := strings.Count(a.accessListText, "access-list")
	parseCount := len(results)
	a.addLog("解析到%d条策略，匹配到%d个策略，排除%d个无效策略------>", parseCount, count, unCount)
	return results
}

func (a *AsaParse) parseAccessList(accessLists []*asaAccessList, objectGroups []*asaObjectGroup, objects []*asaObject, services []*asaService) error {
	objectGroupsM := a.makeObjectGroupAddressesM(objectGroups)
	objectsM := a.makeObjectAddressM(objects)
	serviceM := a.makeServiceM(services)
	results := make([]*model.TDevicePolicy, 0)
	for i, v := range accessLists {
		p := &model.TDevicePolicy{
			DeviceId: a.device.Id,
			Name:     v.name,
			Action:   v.Type,
			Protocol: v.protocol,
			Command:  v.command,
			Line:     i + 1,
		}
		// 策略方向
		switch v.name {
		case a.device.InPolicy:
			p.Direction = "inside"
		case a.device.OutPolicy:
			p.Direction = "outside"
		}
		// 源地址
		switch v.src.Type {
		case "object-group":
			p.SrcGroup = v.src.value
			if addresses, ok := objectGroupsM[v.src.value]; ok {
				p.Src = strings.Join(addresses, ",")
			} else {
				a.addLog(fmt.Sprintf("组装src未获取到object-group: %s", v.src.value))
			}
		case "object":
			p.SrcGroup = v.src.value
			if addresses, ok := objectsM[v.src.value]; ok {
				p.Src = strings.Join(addresses, ",")
			} else {
				a.addLog(fmt.Sprintf("组装src未获取到object: %s", v.src.value))
			}
		default:
			p.Src = v.src.value
		}
		// 目标地址
		switch v.dst.Type {
		case "object-group":
			p.DstGroup = v.dst.value
			if addresses, ok := objectGroupsM[v.dst.value]; ok {
				p.Dst = strings.Join(addresses, ",")
			} else {
				a.addLog(fmt.Sprintf("组装dst未获取到object-group: %s", v.dst.value))
			}
		case "object":
			p.DstGroup = v.dst.value
			if addresses, ok := objectsM[v.dst.value]; ok {
				p.Dst = strings.Join(addresses, ",")
			} else {
				a.addLog(fmt.Sprintf("组装src未获取到object: %s", v.dst.value))
			}
		default:
			p.Dst = v.dst.value
		}
		// 端口
		switch v.port.Type {
		case "object-group":
			if ports, ok := serviceM[v.port.value]; ok {
				p.PortGroup = v.port.value
				p.Port = strings.Join(ports, ",")
			} else {
				a.addLog(fmt.Sprintf("匹配不到端口组: %s", v.port.value))
			}
		default:
			if v1, ok := PortMaps[v.port.value]; ok {
				p.Port = v1
			} else {
				p.Port = v.port.value
			}
		}
		results = append(results, p)
	}
	return a.savePolicy(results)
}

// 组装object address map[object.name][]object.addr
func (a *AsaParse) makeObjectAddressM(objects []*asaObject) map[string][]string {
	result := make(map[string][]string)
	for _, v := range objects {
		result[v.name] = append(result[v.name], v.addresses...)
	}
	return result
}

func (a *AsaParse) makeObjectGroupAddressesM(objectGroups []*asaObjectGroup) map[string][]string {
	result := make(map[string][]string)
	for _, v := range objectGroups {
		result[v.name] = append(result[v.name], v.addresses...)
	}
	return result
}
func (a *AsaParse) makeServiceM(services []*asaService) map[string][]string {
	result := make(map[string][]string)
	for _, v := range services {
		for _, p := range v.ports {
			start, _ := portToInt(p.start)
			end, _ := portToInt(p.end)
			result[v.name] = append(result[v.name], fmt.Sprintf("%d-%d", start, end))
		}
	}
	return result
}

type asaNat struct {
	direction   string
	object      string
	static      string
	protocol    string
	command     string
	line        int
	source      []string
	destination []string
	services    []string
}

// 解析nat, 暂时先不写了，太费时间
func (a *AsaParse) parseNatText() []*asaNat {
	/*
		入向nat
		object network YWJS-86965-177-53
		 nat (inside,outside) static YWJS-86965-97-161
		object network YWJS-92920-27476-172.17.179.215-443
		 nat (inside,outside) static YWJS-92920-27476-116.228.151.161-33322 service tcp https 33322

		nat (inside,outside) after-auto source dynamic SdWan-Host1 DIP-151-192
		nat (inside,outside) after-auto source dynamic SdWan-Host1 DIP-151-191
		nat (inside,outside) after-auto source dynamic neiwang PAT-CN2 destination static CN2 CN2
		nat (inside,outside) after-auto source dynamic any DIP-151.2

		出向nat
		nat (inside,outside) source dynamic neiwang PAT-DCN destination static PAT-DCN-2 PAT-DCN-2
		nat (inside,outside) source dynamic obj_172.17.164.87 dnat10.251.2.11 destination static obj_10.128.86.64 obj_10.128.86.64 service tcp@D8000 tcp@D8000
	*/
	results := make([]*asaNat, 0)
	var nat *asaNat
	for i, line := range strings.Split(a.natText, "\r") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines := splitLineBySpace(line)
		validLineM := map[string]bool{"object": true, "nat": true}
		if !validLineM[lines[0]] {
			continue
		}
		// nat解析分两种情况，存在object的，是入向nat, 如果存在source，则根据方向来算
		// 先解析source情况
		// nat (inside,outside) after-auto source dynamic neiwang PAT-CN2 destination static CN2 CN2
		if lines[2] == "after-auto" && lines[3] == "source" {
			nat = &asaNat{
				direction: lines[1],
				source:    []string{lines[5], lines[6]},
				command:   line,
			}
			if strings.Contains(line, "destination") {
				nat.destination = []string{lines[9], lines[10]}
			}
			if strings.Contains(line, "service") {
				nat.services = []string{lines[12], lines[13]}
			}
			results = append(results, nat)
		} else if lines[2] == "source" {
			nat = &asaNat{
				direction: lines[1],
				source:    []string{lines[4], lines[5]},
				command:   line,
			}
			if strings.Contains(line, "destination") {
				nat.destination = []string{lines[8], lines[9]}
			}
			if strings.Contains(line, "service") {
				nat.services = []string{lines[11], lines[12]}
			}
			results = append(results, nat)
		} else {
			// object network YWJS-86965-177-53
			//	nat (inside,outside) static YWJS-86965-97-161
			//  nat (inside,outside) static YWJS-92920-27476-116.228.151.161-33322 service tcp https 33322
			if lines[0] == "object" {
				nat = &asaNat{
					object:  lines[2],
					command: line,
				}
				continue
			}
			if nat == nil {
				a.addLog(fmt.Sprintf("匹配到无效nat, text: %s, num: %d", line, i+1))
				continue
			}
			nat.command += fmt.Sprintf("\n%s", line)
			if lines[0] == "nat" {
				nat.direction = lines[1]
				nat.static = lines[3]
			}
			if strings.Contains(line, "service") {
				nat.protocol = lines[5]
				nat.services = []string{lines[6], lines[7]}
			}
			results = append(results, nat)
		}
	}
	count := strings.Count(a.natText, "nat (")
	a.addLog("解析到%d条nat, 匹配到%d个nat", len(results), count)
	return results
}

func (a *AsaParse) parseNat(nats []*asaNat, objectGroups []*asaObjectGroup, objects []*asaObject, services []*asaService) error {
	results := make([]*model.TDeviceNat, 0)
	objectGroupsM := a.makeObjectGroupAddressesM(objectGroups)
	objectM := a.makeObjectAddressM(objects)
	serviceM := a.makeServiceM(services)
	// 解析nat信息
	for _, nat := range nats {
		n := &model.TDeviceNat{
			DeviceId: a.device.Id,
			Command:  nat.command,
		}
		results = append(results, n)
		// 如果存在object，说明是入向network转static的，不存在目标地址
		if nat.object != "" {
			n.Direction = "inside"
			if addr := a.getAddress(nat.object, objectGroupsM, objectM); addr != "" {
				n.Network = addr
			} else {
				a.addLog(fmt.Sprintf("无效nat object: %s", nat.object))
				n.Network = nat.object
			}

			if addr := a.getAddress(nat.static, objectGroupsM, objectM); addr != "" {
				n.Static = addr
			} else {
				n.Static = fmt.Sprintf("%s/32", nat.static)
			}

			if len(nat.services) == 0 {
				n.NetworkPort = "any"
				n.StaticPort = "any"
			} else {
				n.Protocol = nat.protocol
				n.NetworkPort = nat.services[0]
				n.StaticPort = nat.services[1]
				// 转换端口
				if p, ok := PortMaps[nat.services[0]]; ok {
					n.NetworkPort = p
				}
				if p, ok := PortMaps[nat.services[1]]; ok {
					n.StaticPort = p
				}
			}
		} else { // 否则可能是出入向，有network static和destination的
			switch nat.direction {
			case "(inside,outside)":
				n.Direction = "outside"
			case "(outside,inside)":
				n.Direction = "inside"
			default:
				a.addLog(fmt.Sprintf("无效nat 方向: %s", nat.direction))
			}
			// 获取地址
			if addr := a.getAddress(nat.source[0], objectGroupsM, objectM); addr != "" {
				n.Network = addr
			} else {
				a.addLog(fmt.Sprintf("无效nat source 0: %s", nat.static))
				n.Network = nat.source[0]
			}

			if addr := a.getAddress(nat.source[1], objectGroupsM, objectM); addr != "" {
				n.Static = addr
			} else {
				a.addLog(fmt.Sprintf("无效nat source 1: %s", nat.static))
				n.Static = nat.source[1]
			}

			if len(nat.destination) == 0 {
				n.Destination = "any"
			} else {
				if addr := a.getAddress(nat.destination[0], objectGroupsM, objectM); addr != "" {
					n.Destination = addr
				} else {
					a.addLog(fmt.Sprintf("无效nat source 1: %s", nat.static))
					n.Destination = nat.destination[0]
				}
			}
			if len(nat.services) == 0 {
				n.NetworkPort = "any"
				n.StaticPort = "any"
			} else {
				if p, ok := serviceM[nat.services[0]]; ok {
					n.NetworkPort = strings.Join(p, ",")
				} else {
					n.NetworkPort = nat.services[0]
				}
				if p, ok := serviceM[nat.services[1]]; ok {
					n.StaticPort = strings.Join(p, ",")
				} else {
					n.StaticPort = nat.services[1]
				}
			}
		}
	}
	return a.saveNat(results)
}

// 解析黑名单组
func (a *AsaParse) parseBlacklistGroupAddress(objectGroups []*asaObjectGroup) {
	deviceGroups, e := a.getBlacklistDeviceGroup()
	if e != nil {
		a.error = e
		return
	}
	if deviceGroups == nil {
		return
	}
	objectGroupsM := a.makeObjectGroupAddressesM(objectGroups)
	a.addLog("需要解析%d个黑名单组", len(deviceGroups))
	for _, v := range deviceGroups {
		fmt.Println("解析地址组--->", v.Name)
		tx := database.DB.Begin()
		addresses, ok := objectGroupsM[v.Name]
		fmt.Println("组内多少个地址--->", len(addresses))
		// 删除地址组内地址
		if e := tx.Delete(&model.TBlacklistDeviceGroupAddress{}, "device_group_id = ?", v.Id).Error; e != nil {
			tx.Rollback()
			a.addLog(fmt.Sprintf("清除组内地址异常, %s", e.Error()))
			continue
		}
		// 如果设备中无此地址组，则需要删除些地址组
		if !ok {
			if e := tx.Delete(v).Error; e != nil {
				tx.Rollback()
				a.addLog("删除无效地址组异常: <%s>", e.Error())
				continue
			}
		} else {
			// 添加新的地址
			bulks := make([]*model.TBlacklistDeviceGroupAddress, 0)
			for _, addr := range addresses {
				bulks = append(bulks, &model.TBlacklistDeviceGroupAddress{
					DeviceId:      v.DeviceId,
					DeviceGroupId: v.Id,
					Ip:            addr,
					IpType:        v.IpType,
				})
			}
			if e := a.saveGroupAddress(tx, bulks); e != nil {
				tx.Rollback()
				a.addLog("保存地址组<%s>地址信息异常: %s", v.Name, e.Error())
				continue
			}
			if e := tx.Commit().Error; e != nil {
				a.addLog("保存地址组<%s>地址信息异常: <%s>", v.Name, e.Error())
			}
		}
	}
}

// 根据地址组从object-group object获取地址信息
func (a *AsaParse) getAddress(name string, objectGroupM, objectM map[string][]string) string {
	if addresses, ok := objectGroupM[name]; ok { // 转换地址名为地址
		return strings.Join(addresses, ",")
	}
	if addresses, ok := objectM[name]; ok {
		return strings.Join(addresses, ",")
	}
	return ""
}
