package device

import (
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"netops/conf"
	"netops/database"
	netApi2 "netops/grpc_client/protobuf/net_api"
	"netops/model"
	"netops/utils"
	"os"
	"strings"
)

func NewH3cHandler(deviceId int) *H3cHandler {
	result := &H3cHandler{}
	result.DeviceId = deviceId
	return result
}

type H3cHandler struct {
	h3cParse
}

func (h *H3cHandler) init() {
	h.backupCommand = "display cur"
	h.base.init()
}

var actions = map[string]string{"pass": "permit", "drop": "deny"}

// ParseConfig 获取并解析配置
func (h *H3cHandler) ParseConfig() {
	h.addLog("<-------开始解析设备策略------->")
	if e := h.parse(); e != nil {
		h.operateLog.Status = "failed"
		h.addLog(e.Error())
		_ = h.device.UpdateParseStatus(ParseStatusFailed)
		return
	}
	_ = h.device.UpdateParseStatus(ParseStatusSuccess)
	h.addLog("<-------解析策略完成------->")
}
func (h *H3cHandler) Search(info *model.TTaskInfo) (*model.TDevicePolicy, error) {
	if h.error != nil {
		return nil, h.error
	}
	l := zap.L().With(zap.Int("infoId", info.Id), zap.String("func", "Search"))
	l.Info("策略查询", zap.Any("info", info))
	l.Debug("1. 根据源目地址模糊匹配符合条件的策略---------->")
	var (
		result    *model.TDevicePolicy
		db        *gorm.DB
		portNames = []string{"any"}
		err       error
	)
	db = database.DB.Where("device_id = ? and dst like ? and direction = ? and action = ?",
		h.DeviceId, "%"+info.Dst+"%", info.Direction, "permit")
	if info.Direction == "inside" && (info.Src == conf.BanGongWang || info.Src == conf.BanGongWangV6) {
		db = db.Where("src_group = ?", info.Src)
	} else {
		db = db.Where("src like ?", "%"+info.Src+"%")
	}

	if info.Protocol != "ip" {
		l.Info("根据端口和协议获取端口所在的组--------->")
		if portNames, err = h.getPortNames(info.DPort, info.Protocol); err != nil {
			return nil, err
		}
		l.Info(fmt.Sprintf("port:<%s> -> portNames:<%+v>", info.DPort, portNames))
		db = db.Where("port = ? or port = ? or port_group in ?", "any", info.DPort, portNames)
	}
	policies := make([]*model.TDevicePolicy, 0)
	db = db.Find(&policies)
	if db.Error != nil {
		h.error = fmt.Errorf("查询策略失败, err: %w", db.Error)
	}
	if len(policies) > 0 {
		l.Info("匹配到策略信息------------------>")
		result = policies[0]
		l.Debug("匹配到策略", zap.Any("policy", result))
		return result, nil
	} else {
		l.Info("2. 根据基本信息未匹配到相应的策略，开始进行网段的匹配------->")
		// 如果没有匹配的策略，则获取所有地址组，根据地址组来查询
		// 先获取源地址为网段的策略信息
		l.Info("先根据基本条件进行过滤------------>")
		subnetPolicies := make([]*model.TDevicePolicy, 0)
		db = database.DB.Where("action = ? and device_id = ? and direction = ?",
			"permit", h.DeviceId, info.Direction)
		if info.Protocol != "ip" {
			db = db.Where("port = ? or port = ? or port in ?", "any", info.DPort, portNames)
		}
		if e1 := db.Find(&subnetPolicies).Error; e1 != nil {
			return nil, fmt.Errorf("查询策略失败, err: %w", e1)
		}
		// 根据源目地址找到符合条件的策略
		l.Info("3. 根据源目地址从策略表中匹配策略--------->")
		policies = h.getSubnetPolicy(subnetPolicies, info.Src, info.Dst)
		if len(policies) > 0 {
			if info.Src == conf.BanGongWang || info.Src == conf.BanGongWangV6 {
				result = policies[0]
			} else {
				// 匹配出最小的策略
				l.Info("根据匹配到的网段策略, 获取范围最小的策略信息-------->")
				result = h.getPolicy(policies, info.Src, info.Dst)
				l.Info(fmt.Sprintf("匹配到的策略: <%+v>", result))
			}
			return result, nil
		}
		l.Info("查询结果", zap.Any("policies", policies))
	}
	l.Info("<-------策略查询结束------->")
	return nil, nil
}

// 获取组名
func (h *H3cHandler) getPortNames(port, protocol string) (results []string, err error) {
	p, err := utils.ParseRangePort(port)
	if err != nil {
		return
	}
	ports := make([]model.TDevicePort, 0)
	db := database.DB.Where("device_id = ? and protocol like ? and start <= ? and end >= ?", h.DeviceId, "%"+protocol+"%",
		p.Start, p.End).Find(&ports)
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
func (h *H3cHandler) GetCommand(dp *model.TDevicePolicy) string {
	return dp.Command
}

// 生成地址组策略
func (h *H3cHandler) geneAddressCmd(direction, name, address, cmdType string) string {
	result := fmt.Sprintf("object-group %s address %s\n", cmdType, name)
	result += fmt.Sprintf(" security-zone %s\n", direction)
	for _, v := range strings.Split(address, ",") {
		ip, mask := ipMask(v)
		if mask == "255.255.255.255" || mask == "128" {
			result += fmt.Sprintf(" network host address %s\n", ip)
		} else {
			result += fmt.Sprintf(" network subnet %s %s\n", ip, mask)
		}
	}

	return result
}

// 获取存在的端口名
// 获取组名
func (h *H3cHandler) getPortName(p utils.RangePort, protocol string) *model.TDevicePort {
	result := &model.TDevicePort{}
	if database.DB.Where("device_id = ? and protocol like ? and start = ? and end = ?", h.DeviceId, "%"+protocol+"%",
		p.Start, p.End).First(result).Error != nil {
		return nil
	}
	return result
}

// 生成port策略
func (h *H3cHandler) genePortCmd(info *model.TTaskInfo) (portNames []string, portCmd string) {
	if info.Protocol != "ip" { // 协议为ip的端口直接any，否则要分割端口生成端口策略
		portNames = make([]string, 0)
		portCmd = ""
		// 将多个端口和范围端口拆分为单个端口
		for _, p := range strings.Split(info.DPort, ",") {
			rp, _ := utils.ParseRangePort(p)
			// TODO 需要确认端口具体生成策略，多个端口若一个端口存在组怎么处理
			if dp := h.getPortName(rp, info.Protocol); dp == nil {
				portName := h.genePortName(info.Protocol, rp)
				portCmd += fmt.Sprintf("object-group service %s\n",
					portName)
				if rp.Start == rp.End {
					portCmd += fmt.Sprintf("service %s destination eq %d\n", info.Protocol, rp.Start)
				} else {
					portCmd += fmt.Sprintf("service %s destination range %d %d\n", info.Protocol, rp.Start, rp.End)
				}
				portNames = append(portNames, portName)
			} else {
				portNames = append(portNames, dp.Name)
			}
		}
		return
	}
	return []string{"any"}, ""
}

// 生成策略命令
func (h *H3cHandler) genePolicyCmd(direction, name, srcGroup, dstGroup string, portNames []string, ipType string) (string, error) {
	var sourceZone, destinationZone string
	switch direction {
	case "inside":
		sourceZone = h.device.OutPolicy
		destinationZone = h.device.InPolicy
	case "outside":
		sourceZone = h.device.InPolicy
		destinationZone = h.device.OutPolicy
	default:
		return "", fmt.Errorf("未知的策略方向")
	}
	policyCmd := fmt.Sprintf("security-policy %s\n", ipType)
	policyCmd += fmt.Sprintf(" rule name %s\n", name)
	policyCmd += fmt.Sprintf("  action pass\n")
	policyCmd += fmt.Sprintf("  source-zone %s\n", sourceZone)
	policyCmd += fmt.Sprintf("  destination-zone %s\n", destinationZone)
	if srcGroup != "" {
		policyCmd += fmt.Sprintf("  source-ip %s\n", srcGroup)
	}
	if dstGroup != "" {
		policyCmd += fmt.Sprintf("  destination-ip %s\n", dstGroup)
	}
	for _, v := range portNames {
		policyCmd += fmt.Sprintf("  service %s\n", v)
	}
	//policyCmd += fmt.Sprintf("security-policy %s\n", cmdType)
	//policyCmd += fmt.Sprintf(" move rule name %s before name Trust_To_Untrsut_Deny", name)
	return policyCmd, nil
}

// 生成端口组名
func (h *H3cHandler) genePortName(protocol string, rp utils.RangePort) string {
	if protocol == "ip" {
		return "any"
	}
	if rp.Start == rp.End {
		return fmt.Sprintf("%s-%d", strings.ToUpper(protocol), rp.Start)
	}
	return fmt.Sprintf("%s-%d-%d", strings.ToUpper(protocol), rp.Start, rp.End)
}

// 生成端口策略
func (h *H3cHandler) genePortPolicy(groupName string, info *model.TTaskInfo) (result string) {
	// 生成端口组配置------------->
	portCommand := fmt.Sprintf("object-group service %s\n", groupName)
	for i, p := range strings.Split(info.DPort, "/") {
		rp, _ := utils.ParseRangePort(p)
		if rp.Start == rp.End {
			portCommand += fmt.Sprintf(" %d service %s destination eq %s\n", i, info.Protocol, p)
		} else {
			portCommand += fmt.Sprintf(" %d service %s destination range %d %d\n", i, info.Protocol, rp.Start, rp.End)
		}
	}
	return portCommand
}

func (h *H3cHandler) getIpType(address string) string {
	switch address {
	case conf.BanGongWangV6:
		return "ipv6"
	case conf.BanGongWang:
		return "ip"
	}
	// 先判断IPV6，因为有办公网组存在
	if utils.GetIpType(address) == "ipv6" {
		return "ipv6"
	}
	return "ip"
}

// GeneCommand 生成策略命令
func (h *H3cHandler) GeneCommand(jiraKey string, info *model.TTaskInfo) (string, error) {
	if h.error != nil {
		return "", h.error
	}
	l := zap.L().With(zap.String("func", "GeneCommand"), zap.Int("info_id", info.Id), zap.String("jira_key", jiraKey))
	l.Info("生成策略", zap.Any("info", info))
	var (
		srcGroupName string
		dstGroupName string
		portNames    []string
		portCmd      string
		commands     = make([]string, 0)
		groupName    = h.groupName(jiraKey, info)
		ipType       = h.getIpType(info.Src)
	)
	l.Info("1. 生成源地址组策略命令--->")
	// 如果办公网，地址组直接设置为办公网
	if info.Src == conf.BanGongWang || info.Src == conf.BanGongWangV6 {
		srcGroupName = info.Src
	} else if srcAddressGroup := h.getAddressGroup(info.Src); srcAddressGroup != nil {
		srcGroupName = srcAddressGroup.Name
	} else if info.Src == "::/0" { // 如果源地址是any 则不生成策略组

	} else {
		srcGroupName = h.geneSrcGroupName(groupName)
		if info.Direction == "inside" {
			commands = append(commands, h.geneAddressCmd(h.device.OutPolicy, srcGroupName, info.Src, ipType))
		} else {
			commands = append(commands, h.geneAddressCmd(h.device.InPolicy, srcGroupName, info.Src, ipType))
		}

	}
	l.Info("2. 生成目标地址组策略命令--->")
	dstAddressGroup := h.getAddressGroup(info.Dst)
	if dstAddressGroup != nil {
		dstGroupName = dstAddressGroup.Name
	} else if info.Dst == "::/0" { // 如果目标地址是any 则不生成策略组

	} else {
		dstGroupName = h.geneDstGroupName(groupName)
		if info.Direction == "inside" {
			commands = append(commands, h.geneAddressCmd(h.device.InPolicy, dstGroupName, info.Dst, ipType))
		} else {
			commands = append(commands, h.geneAddressCmd(h.device.OutPolicy, dstGroupName, info.Dst, ipType))
		}
	}

	l.Info("3. 生成端口策略命令--->")
	if portNames, portCmd = h.genePortCmd(info); portCmd != "" {
		commands = append(commands, portCmd)
	}

	l.Info("4. 生成策略--->")
	policyCmd, e := h.genePolicyCmd(info.Direction, groupName, srcGroupName, dstGroupName, portNames, ipType)
	if e != nil {
		return "", e
	}
	commands = append(commands, policyCmd)
	return strings.Join(commands, "\n"), nil
}

func (h *H3cHandler) CheckNat(info *model.TTaskInfo) (err error) {
	return
}

type h3cParse struct {
	base
	policyText string
	groupText  string
}

func (h *h3cParse) parse() error {
	h.addLog("初始化设备状态--->")
	if e := h.device.UpdateParseStatus(ParseStatusInit); e != nil {
		return e
	}
	h.addLog("1. 获取完整配置信息--->")
	if e := h.getConfig(); e != nil {
		return e
	}
	h.addLog("2. 解析service object group信息--->")
	services := h.parseServiceText()
	if e := h.saveServices(services); e != nil {
		return e
	}

	h.addLog("3. 解析地址组信息--->")
	addresses := h.parseAddressText()
	if e := h.saveAddresses(addresses); e != nil {
		return e
	}

	h.addLog("4. 解析策略信息--->")
	rules := h.parseRuleText()
	if e := h.saveRules(rules, services, addresses); e != nil {
		return e
	}

	//h.addLog("5. 解析黑名单地址组信息--->")
	//h.parseBlacklistGroupAddress(addresses)
	//h.saveNatPool(h.parseNatPool())
	return nil
}
func (h *h3cParse) getConfig1() {
	fb, err := os.ReadFile("C:\\Users\\34607\\.wind\\profiles\\default.v10\\terminal\\logs\\南京\\wh-h3c-ip.log")
	if err != nil {
		h.error = err
		return
	}
	h.policyText += string(fb)
	fb, err = os.ReadFile("C:\\Users\\34607\\.wind\\profiles\\default.v10\\terminal\\logs\\南京\\wh-h3c-ipv6.log")
	if err != nil {
		h.error = err
		return
	}
	h.policyText += string(fb)
	h.policyText += string(fb)
	fb, err = os.ReadFile("C:\\Users\\34607\\.wind\\profiles\\default.v10\\terminal\\logs\\南京\\wh-h3c-group.log")
	if err != nil {
		h.error = err
		return
	}
	h.groupText = string(fb)
}
func (h *h3cParse) getConfig() error {
	commands := []*netApi2.Command{
		{Id: 1, Cmd: "dis security-policy ip"},
		{Id: 2, Cmd: "dis object-group"},
		{Id: 3, Cmd: "dis security-policy ipv6"},
	}
	result, e := h.send(commands)
	if e != nil {
		return e
	}
	for _, item := range result {
		switch item.Id {
		case 1:
			h.policyText += strings.ReplaceAll(item.Result, "---- More ----", "")
		case 2:
			h.groupText = strings.ReplaceAll(item.Result, "---- More ----", "")
		case 3:
			h.policyText += strings.ReplaceAll(item.Result, "---- More ----", "")
		}
	}
	return nil
}

type h3cServiceItem struct {
	protocol string
	start    int
	end      int
}
type h3cService struct {
	name  string
	items []*h3cServiceItem
}

// 解析service
func (h *h3cParse) parseServiceText() []*h3cService {
	/*
			Service object group TBJ-Port: 9 objects(in use)
			TBJ-Port
			 10 service tcp destination range 23389 23389
			Service object group TCP-3302: 1 object(out of use)
			 0 service tcp destination eq 3302
			Service object group TCP-0-65535: 1 object(in use)
		   	 0 service tcp
	*/
	results := make([]*h3cService, 0)
	var service *h3cService
	for i, line := range strings.Split(h.groupText, "\r") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines := splitLineBySpace(line)
		if len(lines) < 3 {
			continue
		}
		// 获取service group object名称
		if lines[0] == "Service" {
			service = &h3cService{
				name: strings.Trim(lines[3], ":"),
			}
			results = append(results, service)
			continue
		}
		// 跳过无效策略
		if lines[1] != "service" {
			continue
		}
		if service == nil {
			h.addLog("无效service, text: %s, num: %d", line, i+1)
			continue
		}
		item := &h3cServiceItem{
			protocol: lines[2],
		}
		// 如果没有定义端口，说明是any
		if len(lines) == 3 {
			item.start = 0
			item.end = 65535
		} else { // 解析端口
			if len(lines) < 6 {
				h.addLog("无效service, text: %d, num: %d", line, i+1)
				continue
			}
			switch lines[4] {
			case "eq": // 0 service tcp destination eq 3302
				start, _ := portToInt(lines[5])
				item.start = start
				item.end = start
			case "range": // 10 service tcp destination range 23389 23389
				start, _ := portToInt(lines[5])
				end, _ := portToInt(lines[6])
				item.start = start
				item.end = end
			}
		}
		service.items = append(service.items, item)
	}
	h.addLog("解析到<%d>个service", len(results))
	return results
}

// 保存service
func (h *h3cParse) saveServices(data []*h3cService) error {
	bulks := make([]*model.TDevicePort, 0)
	for _, v := range data {
		for _, item := range v.items {
			bulks = append(bulks, &model.TDevicePort{
				DeviceId: h.device.Id,
				Name:     v.name,
				Protocol: item.protocol,
				Start:    item.start,
				End:      item.end,
			})
		}
	}
	return h.savePort(bulks)
}

// 组装端口名和端口
func (h *h3cParse) makeServiceM(data []*h3cService) map[string]*h3cService {
	results := make(map[string]*h3cService)
	for _, v := range data {
		results[v.name] = v
		//for _, item := range v.items {
		//	results[v.name] = append(results[v.name], fmt.Sprintf("%d-%d", item.start, item.end))
		//}
	}
	return results
}

type h3cAddressGroup struct {
	name      string
	zone      string
	ipType    string
	addresses []string
}

// 解析地址组 address object group
func (h *h3cParse) parseAddressText() []*h3cAddressGroup {
	/*
		Ip address object group WYJS-29932-SRC: 4 objects(out of use) //地址组
		 security-zone Trust //定义出口
		 10 network host address 172.24.2.2 // 单个地址
		 30 network subnet 172.24.4.0 255.255.255.0
	*/
	groups := make([]*h3cAddressGroup, 0)
	var group *h3cAddressGroup
	for i, line := range strings.Split(h.groupText, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines := splitLineBySpace(line)
		if len(lines) < 2 {
			continue
		}
		// 获取到地址组名  Ip address object group WYJS-29932-SRC: 4 objects(out of use)
		if strings.HasPrefix(line, "Ip address object group") {
			group = &h3cAddressGroup{
				name:   strings.Trim(lines[4], ":"),
				ipType: conf.IpTypeV4,
			}
			groups = append(groups, group)
			continue
		}

		// Ipv6 address object group Ipv6_WH_GW_Blacklist_2023_01: 13 objects(in use)
		if strings.HasPrefix(line, "Ipv6 address object group") {
			group = &h3cAddressGroup{
				name:   strings.Trim(lines[4], ":"),
				ipType: conf.IpTypeV6,
			}
			groups = append(groups, group)
			continue
		}
		// 排除无效策略
		if !(lines[0] == "security-zone" || lines[1] == "network") {
			continue
		}
		if group == nil {
			h.addLog(fmt.Sprintf("无效配置, line: %s, num: %d", line, i+1))
			continue
		}
		switch {
		case lines[0] == "security-zone": // 获取zone Trust or Untrust
			group.zone = lines[1]
		case lines[1] == "network":
			if len(lines) < 3 {
				h.addLog(fmt.Sprintf("无效配置, line: %s, num: %d", line, i+1))
				continue
			}
			// 解析地址信息
			switch lines[2] {
			case "host": // 解析单个地址   10 network host address 172.24.2.2
				address := lines[4]
				if group.ipType == conf.IpTypeV4 {
					address = address + "/32"
				} else {
					address = address + "/128"
				}
				group.addresses = append(group.addresses, address)
			case "subnet":
				// 解析网段 30 network subnet 172.24.4.0 255.255.255.0
				// network subnet 1.1.1.1 32
				// network subnet 1.1.1.1/24
				address := lines[3]
				if !strings.Contains(address, "/") {
					mask := lines[4]
					if group.ipType == conf.IpTypeV6 {
						address = fmt.Sprintf("%s/%s", address, mask)
					} else {
						// network subnet 0.0.0.0 wildcard 255.255.255.255
						if mask == "wildcard" {
							h.addLog(fmt.Sprintf("解析到反掩码地址: %s", line))
							continue
						}
						// network subnet 172.24.4.0 255.255.255.0
						if strings.Contains(mask, ".") {
							address = ipMaskSimple(address, mask)
						} else { // network subnet 1.1.1.1 32
							address = fmt.Sprintf("%s/%s", address, mask)
						}
					}
				}
				group.addresses = append(group.addresses, address)
			case "range": // 160 network range 113.140.93.161 113.140.93.167
				group.addresses = append(group.addresses, h.getRangeAddress(lines[3], lines[4])...)
			default:
				h.addLog(fmt.Sprintf("无效配置, line: %s, num: %d", line, i+1))
			}
		}
	}
	h.addLog("解析到<%d>个address group", len(groups))
	return groups
}

// 保存地址组
func (h *h3cParse) saveAddresses(data []*h3cAddressGroup) error {
	bulks := make([]*model.TDeviceAddressGroup, 0)
	for _, v := range data {
		for _, addr := range v.addresses {
			bulks = append(bulks, &model.TDeviceAddressGroup{
				DeviceId:    h.device.Id,
				Zone:        v.zone,
				Name:        v.name,
				Address:     addr,
				AddressType: v.ipType,
			})
		}
	}
	h.addLog("保存%d个地址组", len(bulks))
	return h.saveGroup(bulks)
}

// 组装地址组地址map
func (h *h3cParse) makeAddressM(data []*h3cAddressGroup) map[string][]string {
	results := make(map[string][]string)
	for _, v := range data {
		results[v.name] = v.addresses
	}
	return results
}

// 解析黑名单组
func (h *h3cParse) parseBlacklistGroupAddress(addresses []*h3cAddressGroup) {
	h.addLog("解析黑名单地址组信息--->")
	deviceGroups, e := h.getBlacklistDeviceGroup()
	if e != nil {
		h.error = e
		return
	}
	if deviceGroups == nil {
		return
	}
	h.addLog("黑名单地址组有<%d>个", len(deviceGroups))
	groupM := h.makeAddressM(addresses)
	for _, v := range deviceGroups {
		fmt.Println("解析地址组--->", v.Name)
		tx := database.DB.Begin()
		addrL, ok := groupM[v.Name]
		fmt.Println("组内多少个地址--->", len(addrL))
		// 删除地址组内地址
		if e := tx.Delete(&model.TBlacklistDeviceGroupAddress{}, "device_group_id = ?", v.Id).Error; e != nil {
			tx.Rollback()
			h.addLog(fmt.Sprintf("清除组内地址异常, %s", e.Error()))
			continue
		}
		// 如果设备中无此地址组，则需要删除些地址组
		if !ok {
			if e := tx.Delete(v).Error; e != nil {
				tx.Rollback()
				h.addLog(fmt.Sprintf("删除无效地址组异常: <%s>", e.Error()))
				continue
			}
		} else {
			// 添加新的地址
			bulks := make([]*model.TBlacklistDeviceGroupAddress, 0)
			for _, ip := range addrL {
				bulks = append(bulks, &model.TBlacklistDeviceGroupAddress{
					DeviceId:      v.DeviceId,
					DeviceGroupId: v.Id,
					Ip:            ip,
					IpType:        v.IpType,
				})
			}
			if e := h.saveGroupAddress(tx, bulks); e != nil {
				tx.Rollback()
				h.addLog(fmt.Sprintf("保存地址组<%s>地址信息异常: <%s>", v.Name, e.Error()))
				continue
			}
		}
		if e := tx.Commit().Error; e != nil {
			h.addLog(fmt.Sprintf("保存地址组<%s>地址信息异常: <%s>", v.Name, e.Error()))
		}
	}
}

type h3cRule struct {
	name            string
	action          string
	sourceZone      string
	destinationZone string
	sourceIp        string
	destinationIp   string
	services        []string
	command         string
	line            int
}

// 解析策略
func (h *h3cParse) parseRuleText() []*h3cRule {
	if h.error != nil {
		return nil
	}
	/*
	 rule 21 name YWJS-93513-28431
	  action pass
	  source-zone Trust
	  destination-zone Untrust
	  source-ip YWJS-93513-28431-SRC
	  destination-ip YWJS-90212-21980-DST
	  service TCP-0-65535
	*/
	results := make([]*h3cRule, 0)
	var p *h3cRule
	for i, line := range strings.Split(h.policyText, "\r") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		validM := map[string]bool{"rule": true, "action": true, "source-zone": true, "destination-zone": true,
			"source-ip": true, "destination-ip": true, "service": true}
		lines := splitLineBySpace(line)
		if _, ok := validM[lines[0]]; !ok {
			continue
		}
		if len(lines) < 2 {
			continue
		}

		// 获取策略名
		if lines[0] == "rule" {
			p = &h3cRule{
				name:    lines[3],
				command: line,
				line:    i,
			}
			results = append(results, p)
			continue
		}
		if p == nil {
			h.addLog("无效策略, text: %s, num: %d", line, i+1)
			continue
		}
		p.command += fmt.Sprintf(" %s\n", line)

		switch lines[0] {
		case "action": // action pass
			p.action = lines[1]
		case "source-zone": // source-zone Trust\
			p.sourceZone = lines[1]
		case "destination-zone":
			p.destinationZone = lines[1]
		case "source-ip":
			p.sourceIp = lines[1]
		case "destination-ip":
			p.destinationIp = lines[1]
		case "service":
			p.services = append(p.services, lines[1])
		}
	}
	h.addLog("解析到<%d>个rule", len(results))
	return results
}

// 保存策略
func (h *h3cParse) saveRules(rules []*h3cRule, services []*h3cService, groups []*h3cAddressGroup) error {
	groupM := h.makeAddressM(groups)
	serviceM := h.makeServiceM(services)
	bulks := make([]*model.TDevicePolicy, 0)
	for _, v := range rules {
		direction := h.parseDirection(v.sourceZone, v.destinationZone)
		if len(v.services) == 0 {
			v.services = append(v.services, "any")
		}
		for _, service := range v.services {
			port, protocol := h.findPort(service, serviceM)
			data := &model.TDevicePolicy{
				DeviceId:  h.DeviceId,
				Name:      v.name,
				Action:    actions[v.action],
				Direction: direction,
				SrcGroup:  v.sourceIp,
				Src:       h.findAddress(v.sourceIp, groupM),
				DstGroup:  v.destinationIp,
				Dst:       h.findAddress(v.destinationIp, groupM),
				PortGroup: service,
				Port:      port,
				Command:   v.command,
				Line:      v.line,
				Protocol:  protocol,
			}
			if service == "any" {
				data.Port = "any"
			}
			bulks = append(bulks, data)
		}
	}
	return h.savePolicy(bulks)
}

// 根据from-zone和to-zone区分出方向
func (h *h3cParse) parseDirection(sourceZone, destinationZone string) string {
	// 解析方向 如果 source-zone in destination-zone out则是出向， source-zone out destination-zone in 则是入向
	switch {
	case sourceZone == h.device.InPolicy && destinationZone == h.device.OutPolicy:
		return "outside"
	case sourceZone == h.device.OutPolicy && destinationZone == h.device.InPolicy:
		return "inside"
	}
	return fmt.Sprintf("%s-%s", sourceZone, destinationZone)
}

// 根据端口组从所有端口组map中找到对应的端口信息
func (h *h3cParse) findPort(portName string, groupPorts map[string]*h3cService) (port, protocol string) {
	if portName == "" || portName == "any" {
		return "any", "ip"
	}
	portList := make([]string, 0)
	if p, ok := PortMaps[portName]; ok {
		return p, "ip"
	}
	if p, ok := groupPorts[portName]; ok {
		for _, v := range p.items {
			protocol = v.protocol
			portList = append(portList, fmt.Sprintf("%d-%d", v.start, v.end))
		}
	} else {
		portList = append(portList, portName)
	}
	port = strings.Join(portList, ",")
	return
}

// 根据地址组从地址组map中找到对应的地址信息
func (h *h3cParse) findAddress(groupName string, groupAddress map[string][]string) string {
	if groupName == "" {
		return "0.0.0.0/0"
	}
	addressList := make([]string, 0)
	for _, dg := range strings.Split(groupName, ",") {
		if addresses, ok := groupAddress[dg]; ok {
			addressList = append(addressList, addresses...)
		} else {
			addressList = append(addressList, dg)
		}
	}
	return strings.Join(addressList, ",")
}
