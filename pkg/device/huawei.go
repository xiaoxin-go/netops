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

func NewHuaWeiHandler(deviceId int) *HuaWeiHandler {
	result := &HuaWeiHandler{}
	result.DeviceId = deviceId
	return result
}

type HuaWeiHandler struct {
	huaWeiParse
}

func (h *HuaWeiHandler) init() {
	h.backupCommand = "display cur"
	h.base.init()
}

// ParseConfig 获取并解析配置
func (h *HuaWeiHandler) ParseConfig() {
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
func (h *HuaWeiHandler) Search(info *model.TTaskInfo) (*model.TDevicePolicy, error) {
	if h.error != nil {
		return nil, h.error
	}
	l := zap.L().With(zap.Int("infoId", info.Id), zap.String("func", "Search"))
	l.Debug("策略查询--->", zap.Any("device", h.device), zap.Any("info", info))
	var (
		result    *model.TDevicePolicy
		db        *gorm.DB
		portNames = []string{"any"}
		err       error
	)
	// 根据设备ID，目标地址和方向获取已经开通的策略
	l.Debug("1. 根据源目地址模糊匹配符合条件的策略--->")
	db = database.DB.Where("device_id = ? and dst like ? and direction = ? and action = ?",
		h.DeviceId, "%"+info.Dst+"%", info.Direction, "permit")
	if info.Direction == "inside" && (info.Src == conf.BanGongWang || info.Src == conf.BanGongWangV6) { // 如果是办公网，则获取源为办公网的策略
		db = db.Where("src_group = ?", info.Src)
	} else {
		db = db.Where("src like ?", "%"+info.Src+"%") // 否则获取源
	}

	// 如果协议不是IP，则获取端口所有所在的组
	if info.Protocol != "ip" {
		l.Debug("根据端口和协议获取端口所在的组--------->")
		if portNames, err = h.getPortNames(info.DPort, info.Protocol); err != nil {
			return nil, err
		}
		l.Debug("根据port和port_names过滤", zap.String("port", info.DPort), zap.Any("port_names", portNames))
		// 根据源目地址获取到的策略，再进一步过滤端口是否开通，端口全匹配，或者端口为any，或者存在一个端口组
		db = db.Where("port = ? or port = ? or port_group in ?", "any", info.DPort, portNames)
	}
	// 查询以上符合条件的策略
	policies := make([]*model.TDevicePolicy, 0)
	if err1 := db.Find(&policies).Error; err1 != nil {
		return nil, fmt.Errorf("查询策略表失败, err: %w", err1)
	}
	// 如果匹配到策略，就返回第一条匹配的策略信息
	if len(policies) > 0 {
		l.Debug("匹配到策略信息--->")
		result = policies[0]
		l.Debug(fmt.Sprintf("匹配到的策略: <%+v>", *result))
		return result, nil
	} else {
		// 否则进行模糊匹配，匹配地址所在的组策略是否开通
		l.Info("2. 根据基本信息未匹配到相应的策略，开始进行网段的匹配------->")
		// 如果没有匹配的策略，则获取所有地址组，根据地址组来查询
		// 先获取源地址为网段的策略信息
		l.Info("先根据基本条件进行过滤------------>")
		subnetPolicies := make([]*model.TDevicePolicy, 0)
		// 获取所有出向或入向所有开通的策略
		db = database.DB.Where("action = ? and device_id = ? and direction = ?",
			"permit", h.DeviceId, info.Direction)
		// 加上端口过滤
		if info.Protocol != "ip" {
			db = db.Where("port = ? or port = ? or port in ?", "any", info.DPort, portNames)
		}
		// 获取到所有端口和方向已经开通的策略
		if e1 := db.Find(&subnetPolicies).Error; e1 != nil {
			return nil, fmt.Errorf("查询策略地址组失败, err: %w", e1)
		}
		// 根据源目地址找到符合条件的策略
		l.Info("3. 根据源目地址从策略表中匹配策略--------->")
		policies = h.getSubnetPolicy(subnetPolicies, info.Src, info.Dst)
		if len(policies) > 0 {
			if info.Src == conf.BanGongWang || info.Src == conf.BanGongWangV6 {
				result = policies[0]
			} else {
				// 匹配出最小的策略
				l.Debug("根据匹配到的网段策略, 获取范围最小的策略信息-------->")
				result = h.getPolicy(policies, info.Src, info.Dst)
				l.Debug("匹配到的策略--->", zap.Any("result", result))
			}
			return result, nil
		}
	}
	l.Debug("<-------策略查询结束------->")
	return nil, nil
}

// 获取端口所有组
func (h *HuaWeiHandler) getPortNames(port, protocol string) (results []string, err error) {
	p, err := utils.ParseRangePort(port)
	if err != nil {
		return
	}
	ports := make([]model.TDevicePort, 0)
	db := database.DB.Where("device_id = ? and protocol = ? and start <= ? and end >= ?", h.DeviceId, protocol,
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
func (h *HuaWeiHandler) GetCommand(dp *model.TDevicePolicy) string {
	return dp.Command
}

// 生成地址组策略
func (h *HuaWeiHandler) geneAddressCmd(name, address string) string {
	/*
		ip address-set YWJS-87297-001-SRC type object
		 address 0 202.102.24.159 mask 32
		 address 1 14.18.124.150 mask 32
		 address 2 124.78.72.250 mask 32
	*/
	result := fmt.Sprintf("ip address-set %s type object\n", name)
	for index, v := range strings.Split(address, ",") {
		ip, mask := ipMask(v)
		// IPV4
		if strings.Contains(ip, ".") {
			result += fmt.Sprintf(" address %d %s mask %d\n", index, ip, maskSimple(mask))
		} else {
			result += fmt.Sprintf(" address %d %s %s\n", index, ip, mask)
		}
	}
	return result
}

// 获取存在的端口名
// 获取组名
func (h *HuaWeiHandler) getPortName(p utils.RangePort, protocol string) *model.TDevicePort {
	result := &model.TDevicePort{}
	if database.DB.Where("device_id = ? and protocol = ? and start = ? and end = ?", h.DeviceId, protocol,
		p.Start, p.End).First(result).Error != nil {
		return nil
	}
	return result
}

// 生成port策略
func (h *HuaWeiHandler) genePortCmd(info *model.TTaskInfo) (portNames []string, portCmd string) {
	/*
		ip service-set TCP-9091 type object
		service 0 protocol tcp source-port 0 to 65535 destination-port 9091
		ip service-set TCP-9090-9100 type object 1044（这个值不能和前面的重复）
		service 0 protocol tcp source-port 0 to 65535 destination-port 9090 to 9100
	*/
	if info.Protocol != "ip" { // 协议为ip的端口直接any，否则要分割端口生成端口策略
		portNames = make([]string, 0)
		portCmd = ""
		// 将多个端口和范围端口拆分为单个端口
		for _, p := range strings.Split(info.DPort, ",") {
			rp, _ := utils.ParseRangePort(p)
			// TODO 需要确认端口具体生成策略，多个端口若一个端口存在组怎么处理
			if dp := h.getPortName(rp, info.Protocol); dp == nil {
				portName := h.genePortName(info.Protocol, rp)
				portCmd += fmt.Sprintf("ip service-set %s type object\n", portName)
				if rp.Start == rp.End {
					portCmd += fmt.Sprintf("service 0 protocol %s source-port 0 to 65535 destination-port %d\n", info.Protocol, rp.Start)
				} else {
					portCmd += fmt.Sprintf("service 0 protocol %s source-port 0 to 65535 destination-port %d to %d\n", info.Protocol, rp.Start, rp.End)
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
func (h *HuaWeiHandler) genePolicyCmd(direction, name, srcGroup, dstGroup string, portNames []string) string {
	/*
	 security-policy
	 rule name YWJS-87034-Pre
	  source-zone untrust
	  destination-zone trust
	  source-address address-set YWJS-87034
	  destination-address address-set YWJS-87034-Det
	  destination-address 58.217.201.11 mask 255.255.255.255
	  service TCP-3307
	  service TCP-8001
	  action permit
	*/
	var sourceZone, destinationZone string
	switch direction {
	case "inside":
		sourceZone = h.device.OutPolicy
		destinationZone = h.device.InPolicy
	case "outside":
		sourceZone = h.device.InPolicy
		destinationZone = h.device.OutPolicy
	default:
		h.error = fmt.Errorf("未知的策略方向")
	}
	policyCmd := fmt.Sprintf("security-policy \n")
	policyCmd += fmt.Sprintf(" rule name %s\n", name)
	policyCmd += fmt.Sprintf("  source-zone %s\n", sourceZone)
	policyCmd += fmt.Sprintf("  destination-zone %s\n", destinationZone)
	if srcGroup != "any" {
		policyCmd += fmt.Sprintf("  source-address address-set %s\n", srcGroup)
	}
	if dstGroup != "any" {
		policyCmd += fmt.Sprintf("  destination-address address-set %s\n", dstGroup)
	}
	for _, v := range portNames {
		policyCmd += fmt.Sprintf("  service %s\n", v)
	}
	policyCmd += fmt.Sprintf("  action permit\n")
	return policyCmd
}

// 生成端口组名
func (h *HuaWeiHandler) genePortName(protocol string, rp utils.RangePort) string {
	if protocol == "ip" {
		return "any"
	}
	if rp.Start == rp.End {
		return fmt.Sprintf("%s-%d", strings.ToUpper(protocol), rp.Start)
	}
	return fmt.Sprintf("%s-%d-%d", strings.ToUpper(protocol), rp.Start, rp.End)
}

// 生成端口策略
func (h *HuaWeiHandler) genePortPolicy(groupName string, info *model.TTaskInfo) (result string) {
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

func (h *HuaWeiHandler) getCmdType(address string) string {
	if utils.GetIpType(address) == "ipv4" {
		return "ip"
	}
	return "ipv6"
}

// GeneCommand 生成策略命令
func (h *HuaWeiHandler) GeneCommand(jiraKey string, info *model.TTaskInfo) (string, error) {
	if h.error != nil {
		return "", h.error
	}
	l := zap.L().With(zap.Int("info_id", info.Id), zap.String("func", "GeneCommand"), zap.String("jira_key", jiraKey))
	l.Debug("生成策略命令--->", zap.Any("info", info))
	var (
		srcGroupName string
		dstGroupName string
		portNames    []string
		portCmd      string
		commands     = make([]string, 0)
		groupName    = h.groupName(jiraKey, info)
	)
	l.Info("1. 生成源地址组策略命令--->")
	// 如果办公网，地址组直接设置为办公网
	if info.Src == "0.0.0.0/0" {
		srcGroupName = "any"
	} else if info.Src == conf.BanGongWang || info.Src == conf.BanGongWangV6 {
		srcGroupName = info.Src
	} else if srcAddressGroup := h.getAddressGroup(info.Src); srcAddressGroup != nil {
		srcGroupName = srcAddressGroup.Name
	} else {
		srcGroupName = h.geneSrcGroupName(groupName)
		commands = append(commands, h.geneAddressCmd(srcGroupName, info.Src))
	}
	l.Info("2. 生成目标地址组策略命令--------------------------->")
	if info.Dst == "0.0.0.0/0" {
		dstGroupName = "any"
	} else if dstAddressGroup := h.getAddressGroup(info.Dst); dstAddressGroup != nil {
		dstGroupName = dstAddressGroup.Name
	} else {
		dstGroupName = h.geneDstGroupName(groupName)
		commands = append(commands, h.geneAddressCmd(dstGroupName, info.Dst))
	}

	l.Info("3. 生成端口策略命令-------------------------->")
	if portNames, portCmd = h.genePortCmd(info); portCmd != "" {
		commands = append(commands, portCmd)
	}

	l.Info("4. 生成策略---------------------------------->")
	commands = append(commands, h.genePolicyCmd(info.Direction, groupName, srcGroupName, dstGroupName, portNames))
	return strings.Join(commands, "\n"), nil
}

func (h *HuaWeiHandler) CheckNat(info *model.TTaskInfo) (err error) {
	return
}

type huaWeiParse struct {
	base
	policyText        string
	addressGroupText  string
	addressObjectText string
	portGroupText     string
	portObjectText    string
}

func (h *huaWeiParse) parse() error {
	if e := h.device.UpdateParseStatus(ParseStatusInit); e != nil {
		return e
	}
	h.addLog("<-------开始解析设备策略------->")
	h.addLog("1. 获取完整配置信息-------------->")
	if e := h.getConfig(); e != nil {
		return e
	}

	h.addLog("2. 解析端口信息----------------->")
	services := h.parseServiceText()
	if e := h.saveServices(services); e != nil {
		return e
	}

	h.addLog("3. 解析地址组信息--------------->")
	addressSets := h.parseAddressGroupText()
	if e := h.saveAddressGroup(addressSets); e != nil {
		return e
	}

	h.addLog("4. 解析策略信息----------------->")
	policies := h.parsePolicyText()
	if e := h.savePolicies(policies, addressSets, services); e != nil {
		return e
	}

	//h.addLog("5. 解析黑名单组地址--------------->")
	//h.parseBlacklistGroupAddress()
	return nil
}
func (h *huaWeiParse) getConfig1() {
	fb, err := os.ReadFile("C:\\Users\\34607\\.wind\\profiles\\default.v10\\terminal\\logs\\南京\\nj-huawei-policy.log")
	if err != nil {
		h.error = err
		return
	}
	h.policyText = string(fb)
	fb, err = os.ReadFile("C:\\Users\\34607\\.wind\\profiles\\default.v10\\terminal\\logs\\南京\\nj-huawei-address-group.log")
	if err != nil {
		h.error = err
		return
	}
	h.addressGroupText = string(fb)
	fb, err = os.ReadFile("C:\\Users\\34607\\.wind\\profiles\\default.v10\\terminal\\logs\\南京\\nj-huawei-address-object.log")
	if err != nil {
		h.error = err
		return
	}
	h.addressObjectText = string(fb)
	fb, err = os.ReadFile("C:\\Users\\34607\\.wind\\profiles\\default.v10\\terminal\\logs\\南京\\nj-huawei-service-group.log")
	if err != nil {
		h.error = err
		return
	}
	h.portGroupText = string(fb)
	fb, err = os.ReadFile("C:\\Users\\34607\\.wind\\profiles\\default.v10\\terminal\\logs\\南京\\nj-huawei-service-object.log")
	if err != nil {
		h.error = err
		return
	}
	h.portObjectText = string(fb)
}
func (h *huaWeiParse) getConfig() error {
	h.addLog("2. 获取配置----------------->")
	commands := []*netApi2.Command{
		{Id: 1, Cmd: "dis current-configuration configuration policy-security"},
		{Id: 2, Cmd: "dis ip address-set type group"},
		{Id: 3, Cmd: "dis ip address-set type object"},
		{Id: 4, Cmd: "dis ip service-set type group"},
		{Id: 5, Cmd: "dis ip service-set type object"},
	}
	result, e := h.send(commands)
	if e != nil {
		return e
	}
	for _, item := range result {
		item.Result = strings.ReplaceAll(item.Result, "---- More ----", "")
		item.Result = strings.ReplaceAll(item.Result, "\u001B[42D                                          \u001B[42D", "")
		switch item.Id {
		case 1:
			h.policyText = item.Result
		case 2:
			h.addressGroupText = item.Result
		case 3:
			h.addressObjectText = item.Result
		case 4:
			h.portGroupText = item.Result
		case 5:
			h.portObjectText = item.Result
		}
	}
	return nil
}

type huaweiServiceObjectItem struct {
	protocol string
	start    string
	end      string
}
type huaweiServiceObject struct {
	name  string
	items []*huaweiServiceObjectItem
}

func (h *huaWeiParse) parseServiceText() []*huaweiServiceObject {
	/*
		Service-set Name: TCP-10081
		Type: object
		Item number(s): 1
		Reference number(s): 3
		Item(s):
		 service 0 protocol tcp source-port 0 to 65535 destination-port 10081
		 service protocol tcp source-port 8000 destination-port 7000
		 service 1 protocol tcp source-port 0 to 65535 destination-port 10081 to 10085
	*/
	results := make([]*huaweiServiceObject, 0)
	var service *huaweiServiceObject
	for i, line := range strings.Split(h.portObjectText, "\r") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines := splitLineBySpace(line)
		if len(lines) < 2 {
			continue
		}
		// 排除无效行
		validM := map[string]bool{"Service-set": true, "Type:": true, "service": true}
		if _, ok := validM[lines[0]]; !ok {
			continue
		}
		// Service-set Name: TCP-10081
		if lines[0] == "Service-set" {
			service = &huaweiServiceObject{
				name: lines[2],
			}
			results = append(results, service)
			continue
		}
		if service == nil {
			h.addLog("无效的service object, text: %s, num: %d", line, i+1)
			continue
		}

		if lines[0] == "service" {
			item := &huaweiServiceObjectItem{
				protocol: lines[3],
			}
			// service 1 protocol tcp source-port 0 to 65535 destination-port 10081 to 10085
			dstLines := splitLineBySpace(strings.Split(line, "destination-port")[1])
			if len(dstLines) == 1 {
				item.start = dstLines[0]
				item.end = dstLines[0]
			} else { // 10081 to 10085
				item.start = dstLines[0]
				item.end = dstLines[2]
			}
			service.items = append(service.items, item)
		}
	}
	h.addLog("解析到<%d>个service object", len(results))
	return results
}

// 解析端口
func (h *huaWeiParse) saveServices(data []*huaweiServiceObject) error {
	bulks := make([]*model.TDevicePort, 0)
	for _, service := range data {
		for _, v := range service.items {
			start, _ := portToInt(v.start)
			end, _ := portToInt(v.end)
			bulks = append(bulks, &model.TDevicePort{
				DeviceId: h.DeviceId,
				Name:     service.name,
				Protocol: v.protocol,
				Start:    start,
				End:      end,
			})
		}
	}
	return h.savePort(bulks)
}

type huaweiAddressSetItem struct {
	addressType string
	address     string
}
type huaweiAddressSet struct {
	name  string
	items []*huaweiAddressSetItem
}

// 解析address-set
func (h *huaWeiParse) parseAddressGroupText() []*huaweiAddressSet {
	/*
		Address-set: IPV6-BanGongWang
		Type: object
		Item number(s): 4
		Reference number(s): 0
		Item(s):
		 address 0 240E:E5:8000:19::3 128
		 address 1 218.2.210.74 mask 32
		 address 2 218.94.158.244 mask 255.255.255.255
		 address 0 range 172.20.96.11 172.20.96.13
		Address-set: IPV6-BanGongWang
		Type: group
		Item number(s): 4
		Reference number(s): 0
		Item(s):
		 address 0 240E:E5:8000:19::3 128
		 address 1 address-set 180.167.148.2/32
	*/
	groups := make([]*huaweiAddressSet, 0)
	objects := make([]*huaweiAddressSet, 0)
	var addressSet *huaweiAddressSet
	for i, line := range strings.Split(h.addressObjectText+h.addressGroupText, "\r") {
		line = strings.TrimSpace(line)
		validLines := map[string]bool{"Address-set:": true, "Type:": true, "address": true}
		if line == "" {
			continue
		}
		lines := splitLineBySpace(line)
		// 跳过无效行
		if _, ok := validLines[lines[0]]; !ok {
			continue
		}
		// Address-set: IPV6-BanGongWang
		if lines[0] == "Address-set:" {
			addressSet = &huaweiAddressSet{
				name: lines[1],
			}
			continue
		}
		if addressSet == nil {
			h.addLog(fmt.Sprintf("配置缺失, line: %s, num: %d", line, i+1))
			continue
		}
		// Type: object
		if lines[0] == "Type:" {
			switch lines[1] {
			case "object":
				objects = append(objects, addressSet)
			case "group":
				groups = append(groups, addressSet)
			}
		}
		/*
		  address 0 240E:E5:8000:19::3 128
		  address 1 218.2.210.74 mask 32
		  address 2 218.94.158.244 mask 255.255.255.255
		  address 0 range 172.20.96.11 172.20.96.13
		*/
		if lines[0] == "address" {
			if len(lines) < 4 {
				h.addLog(fmt.Sprintf("不符合解析条件的地址: %s", line))
				continue
			}
			// address 0 range 172.20.96.11 172.20.96.13  添加范围地址，只能解析同C段的地址
			if strings.Contains(line, "range") {
				addresses := h.getRangeAddress(lines[3], lines[4])
				for _, v := range addresses {
					addressSet.items = append(addressSet.items, &huaweiAddressSetItem{
						address: v,
					})
				}
				continue
			}
			item := &huaweiAddressSetItem{}
			addressSet.items = append(addressSet.items, item)
			// address 1 address-set 180.167.148.2/32
			if strings.Contains(line, "address-set") {
				item.addressType = "address-set"
				item.address = lines[3]
				continue
			}
			if strings.Contains(line, "mask") {
				mask := lines[4]
				addr := lines[2]
				// 255.255.255.255
				if strings.Contains(mask, ".") {
					item.address = ipMaskSimple(addr, mask)
				} else { // 32
					item.address = fmt.Sprintf("%s/%s", addr, mask)
				}
				continue
			}
			// address 0 240E:E5:8000:19::3 128
			item.address = fmt.Sprintf("%s/%s", lines[2], lines[3])
		}
	}
	objectM := map[string]*huaweiAddressSet{}
	for _, v := range objects {
		objectM[v.name] = v
	}
	result := make([]*huaweiAddressSet, 0)
	result = append(result, objects...)
	// 循环地址组，地址组内包含地址object，需要解构object里的地址到group里
	for _, v := range groups {
		for _, item := range v.items {
			aSet := &huaweiAddressSet{
				name: v.name,
			}
			result = append(result, aSet)
			// 解构地址
			if item.addressType == "address-set" {
				// 根据地址名从objectM中拿到详细的地址
				ob, ok := objectM[item.address]
				if !ok {
					h.addLog(fmt.Sprintf("未获取到地址object: %s", item.address))
					continue
				}
				// 把object下的地址加到组内
				aSet.items = append(aSet.items, ob.items...)
			} else {
				aSet.items = append(aSet.items, item)
			}
		}
	}
	h.addLog(fmt.Sprintf("解析到%d条组地址", len(result)))
	return result
}

// 解析地址和地址组
// 地址存在地址名和地址组名
// 先解析地址名保存起来
// 再解析地址组直接关联地址保存起来，主要用于策略关联
func (h *huaWeiParse) saveAddressGroup(data []*huaweiAddressSet) error {
	bulks := make([]*model.TDeviceAddressGroup, 0)
	for _, v := range data {
		for _, item := range v.items {
			bulks = append(bulks, &model.TDeviceAddressGroup{
				DeviceId:    h.DeviceId,
				Name:        v.name,
				Address:     item.address,
				AddressType: item.addressType,
			})
		}
	}
	return h.saveGroup(bulks)
}

// 解析黑名单组
func (h *huaWeiParse) parseBlacklistGroupAddress() {
	deviceGroups, e := h.getBlacklistDeviceGroup()
	if e != nil {
		h.error = e
		return
	}
	if deviceGroups == nil {
		return
	}
	h.addLog("需要解析<%d>个黑名单地址组")
	addressSets := h.parseAddressGroupText()
	addressSetM := make(map[string]*huaweiAddressSet)
	for _, v := range addressSets {
		addressSetM[v.name] = v
	}
	for _, v := range deviceGroups {
		fmt.Println("解析地址组--->", v.Name)
		tx := database.DB.Begin()
		group, ok := addressSetM[v.Name]
		fmt.Println("组内多少个地址--->", len(group.items))
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
			for _, item := range group.items {
				bulks = append(bulks, &model.TBlacklistDeviceGroupAddress{
					DeviceId:      v.DeviceId,
					DeviceGroupId: v.Id,
					Ip:            item.address,
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

type huaweiPolicy struct {
	name               string
	action             string
	sourceZone         string
	destinationZone    string
	sourceAddress      []*huaweiAddressSetItem
	destinationAddress []*huaweiAddressSetItem
	commands           []string
	line               int
	services           []string
}

func (h *huaWeiParse) parsePolicyText() []*huaweiPolicy {
	/*
	 security-policy
	 rule name YWJS-87034-Pre
	  source-zone untrust
	  destination-zone trust
	  source-address address-set YWJS-87034
	  destination-address address-set YWJS-87034-Det
	  destination-address 58.217.201.11 mask 255.255.255.255
	  service TCP-3307
	  service TCP-8001
	  action permit
	*/
	results := make([]*huaweiPolicy, 0)
	var hp *huaweiPolicy
	for i, line := range strings.Split(h.policyText, "\r") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, ">") {
			continue
		}
		lines := splitLineBySpace(line)
		if len(lines) < 2 {
			continue
		}
		// 获取策略名
		if strings.HasPrefix(line, "rule") {
			hp = &huaweiPolicy{
				name: lines[2],
				line: i + 1,
			}
			results = append(results, hp)
			continue
		}
		if hp == nil {
			continue
		}
		hp.commands = append(hp.commands, line)
		// source-zone untrust
		if lines[0] == "source-zone" {
			hp.sourceZone = lines[1]
			continue
		}
		// destination-zone trust
		if lines[0] == "destination-zone" {
			hp.destinationZone = lines[1]
			continue
		}
		// source-address address-set YWJS-87034
		// destination-address 58.217.201.11 mask 255.255.255.255
		if lines[0] == "source-address" {
			if lines[1] == "address-set" {
				hp.sourceAddress = append(hp.sourceAddress, &huaweiAddressSetItem{
					address:     lines[2],
					addressType: "address-set",
				})
			} else {
				// source-address range 1.1.1.1 1.1.1.10
				if lines[1] == "range" {
					addresses := h.getRangeAddress(lines[2], lines[3])
					for _, addr := range addresses {
						hp.sourceAddress = append(hp.sourceAddress, &huaweiAddressSetItem{
							address: addr,
						})
					}
					continue
				}
				addr := lines[1]
				// destination-address 58.217.201.11 mask 255.255.255.255
				if lines[2] == "mask" {
					addr = ipMaskSimple(addr, lines[3])
				} else { // destination-address 58.217.201.11 32
					addr = fmt.Sprintf("%s/%s", addr, lines[2])
				}
				hp.sourceAddress = append(hp.sourceAddress, &huaweiAddressSetItem{
					address: addr,
				})
			}
			continue
		}
		// destination-address address-set YWJS-87034-Det
		if lines[0] == "destination-address" {
			if lines[1] == "address-set" {
				hp.destinationAddress = append(hp.destinationAddress, &huaweiAddressSetItem{
					address:     lines[2],
					addressType: "address-set",
				})
			} else {
				// source-address range 1.1.1.1 1.1.1.10
				if lines[1] == "range" {
					addresses := h.getRangeAddress(lines[2], lines[3])
					for _, addr := range addresses {
						hp.destinationAddress = append(hp.destinationAddress, &huaweiAddressSetItem{
							address: addr,
						})
					}
					continue
				}
				addr := lines[1]
				// destination-address 58.217.201.11 mask 255.255.255.255
				if lines[2] == "mask" {
					addr = ipMaskSimple(addr, lines[3])
				} else { // destination-address 58.217.201.11 32
					addr = fmt.Sprintf("%s/%s", addr, lines[2])
				}
				hp.destinationAddress = append(hp.destinationAddress, &huaweiAddressSetItem{
					address: addr,
				})
			}
			continue
		}
		// action permit
		if lines[0] == "action" {
			hp.action = lines[1]
			continue
		}
		// service TCP-8001
		if lines[0] == "service" {
			hp.services = append(hp.services, lines[1])
			continue
		}
	}
	h.addLog("解析到<%d>条策略", len(results))
	return results
}

// 解析策略
func (h *huaWeiParse) savePolicies(policies []*huaweiPolicy, addressSets []*huaweiAddressSet, services []*huaweiServiceObject) error {
	bulks := make([]*model.TDevicePolicy, 0)
	for _, v := range policies {
		direction := h.parseDirection(v.sourceZone, v.destinationZone)
		if len(v.services) == 0 {
			v.services = append(v.services, "any")
		}
		srcGroups := make([]string, 0)
		for _, s := range v.sourceAddress {
			srcGroups = append(srcGroups, s.address)
		}
		dstGroups := make([]string, 0)
		for _, s := range v.destinationAddress {
			dstGroups = append(dstGroups, s.address)
		}
		for _, s := range v.services {
			item := &model.TDevicePolicy{
				DeviceId:  h.DeviceId,
				Name:      v.name,
				Action:    v.action,
				Direction: direction,
				SrcGroup:  strings.Join(srcGroups, ","),
				DstGroup:  strings.Join(dstGroups, ","),
				PortGroup: s,
				Command:   strings.Join(v.commands, "\n"),
				Line:      v.line,
			}
			bulks = append(bulks, item)
		}
	}
	addressSetM := h.makeAddressSetM(addressSets)
	serviceM := h.makeServiceM(services)
	for _, v := range bulks {
		// 取出源地址组的所有地址拼接
		v.Src = h.findAddress(v.SrcGroup, addressSetM)
		// 取出目标地址组的所有地址拼接
		v.Dst = h.findAddress(v.DstGroup, addressSetM)
		// 拼接端口组
		v.Port = h.findPort(v.PortGroup, serviceM)
	}
	return h.savePolicy(bulks)
}

// 根据from-zone和to-zone区分出方向
func (h *huaWeiParse) parseDirection(sourceZone, destinationZone string) string {
	// 解析方向 如果 source-zone in destination-zone out则是出向， source-zone out destination-zone in 则是入向
	switch {
	case sourceZone == h.device.InPolicy && destinationZone == h.device.OutPolicy:
		return "outside"
	case sourceZone == h.device.OutPolicy && destinationZone == h.device.InPolicy:
		return "inside"
	}
	return fmt.Sprintf("%s-%s", sourceZone, destinationZone)
}

// 判断是否是策略行
func (h *huaWeiParse) isPolicyLine(line string) (string, bool) {
	if line == "" {
		return "", false
	}
	prefixes := []string{"rule", "action", "source-zone", "destination-zone", "source-address", "destination-address", "service"}
	for _, prefix := range prefixes {
		if strings.Contains(line, prefix) {
			return prefix + strings.Split(line, prefix)[1], true
		}
	}
	return "", false
}

// 根据地址组从地址组map中找到对应的地址信息
func (h *huaWeiParse) findAddress(groupName string, groupAddress map[string][]string) string {
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

// 根据端口组从所有端口组map中找到对应的端口信息
func (h *huaWeiParse) findPort(portName string, groupPorts map[string][]string) string {
	if portName == "" || portName == "any" {
		return "any"
	}
	portList := make([]string, 0)
	if p, ok := PortMaps[portName]; ok {
		return p
	}
	if p, ok := groupPorts[portName]; ok {
		portList = append(portList, p...)
	} else {
		portList = append(portList, portName)
	}
	return strings.Join(portList, ",")
}

// 拼接同组的地址
func (h *huaWeiParse) makeAddressSetM(data []*huaweiAddressSet) map[string][]string {
	if h.error != nil {
		return nil
	}
	result := make(map[string][]string)
	for _, v := range data {
		for _, item := range v.items {
			result[v.name] = append(result[v.name], item.address)
		}
	}
	return result
}

// 组装端口信息
func (h *huaWeiParse) makeServiceM(data []*huaweiServiceObject) (result map[string][]string) {
	if h.error != nil {
		return
	}
	result = make(map[string][]string)
	for _, v := range data {
		for _, item := range v.items {
			result[v.name] = append(result[v.name], fmt.Sprintf("%s-%s", item.start, item.end))
		}
	}
	return
}
