package device

import (
	"errors"
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

type srxPortItem struct {
	start int
	end   int
}
type srxPort struct {
	name     string
	protocol string
	items    []*srxPortItem
}

func NewSrxHandler(deviceId int) *SrxHandler {
	result := &SrxHandler{}
	result.DeviceId = deviceId
	return result
}

type SrxHandler struct {
	srxParse
}

func (s *SrxHandler) init() {
	s.backupCommand = "show conf"
	s.base.init()
}

// ParseConfig 获取并解析配置
func (s *SrxHandler) ParseConfig() {
	s.addLog("<-------开始解析设备策略------->")
	if e := s.parse(); e != nil {
		s.operateLog.Status = "failed"
		s.addLog(e.Error())
		_ = s.device.UpdateParseStatus(ParseStatusFailed)
		return
	}
	_ = s.device.UpdateParseStatus(ParseStatusSuccess)
	s.addLog("<-------解析策略完成------->")
}
func (s *SrxHandler) search(info *model.TTaskInfo) (*model.TDevicePolicy, error) {
	l := zap.L().With(zap.String("func", "Search"), zap.Int("info_id", info.Id))
	l.Debug("策略查询--->", zap.Any("device", s.device), zap.Any("info", info))
	var (
		result    *model.TDevicePolicy
		db        *gorm.DB
		portNames = []string{"any"}
		err       error
	)
	l.Debug("1. 根据源目地址模糊匹配符合条件的策略---------->")
	db = database.DB.Where("device_id = ? and dst like ? and direction = ? and valid = ?",
		s.DeviceId, "%"+info.Dst+"%", info.Direction, true)
	// 如果是办公网，直接对组名
	if info.Direction == "inside" && (info.Src == conf.BanGongWang || info.Src == conf.BanGongWangV6) {
		db = db.Where("src_group = ?", info.Src)
	} else {
		// 正常逻辑校验规则，源地址包含，目标地址包含，协议相同，方向相同，并且已开通的
		db = db.Where("src like ?", "%"+info.Src+"%")
	}

	if info.Protocol != "ip" {
		l.Debug("根据端口和协议获取端口所在的组--------->")
		if portNames, err = s.getPortNames(info.DPort, info.Protocol); err != nil {
			return nil, err
		}
		l.Debug("根据port和port_names过滤", zap.String("port", info.DPort), zap.Any("port_names", portNames))
		db = db.Where("port = ? or port = ? or port_group in ?", "any", info.DPort, portNames)
	}
	policies := make([]*model.TDevicePolicy, 0)
	if e1 := db.Find(&policies).Error; e1 != nil {
		return nil, fmt.Errorf("查询策略表异常: <%s>", e1.Error())
	}
	if len(policies) > 0 {
		l.Debug("匹配到策略信息------------------>")
		result = policies[0]
		// 如果匹配到的第一条是deny, 说明未开通
		if result.Action == "deny" {
			return nil, nil
		}
		l.Debug("匹配到的策略--->", zap.Any("result", result))
		return result, nil
	} else {
		l.Debug("2. 根据基本信息未匹配到相应的策略，开始进行网段的匹配------->")
		// 如果没有匹配的策略，则获取所有地址组，根据地址组来查询
		// 先获取源地址为网段的策略信息
		l.Debug("先根据基本条件进行过滤------------>")
		subnetPolicies := make([]*model.TDevicePolicy, 0)
		db = database.DB.Where("device_id = ? and direction = ? and valid = ?", s.DeviceId, info.Direction, true)
		if info.Protocol != "ip" {
			// 匹配端口等于目标端口，或者端口是any，或者策略在端口存在的组
			db = db.Where("port = ? or port = ? or port_group in ?", "any", info.DPort, portNames)
		}
		if e1 := db.Find(&subnetPolicies).Error; e1 != nil {
			return nil, fmt.Errorf("查询策略地址组异常, err: %w", e1)
		}
		// 根据源目地址找到符合条件的策略
		l.Info("3. 根据源目地址从策略表中匹配策略--------->")
		policies = s.getSubnetPolicy(subnetPolicies, info.Src, info.Dst)
		if len(policies) > 0 {
			if info.Src == conf.BanGongWang || info.Src == conf.BanGongWangV6 {
				result = policies[0]
			} else {
				// 匹配出最小的策略
				l.Debug("根据匹配到的网段策略, 获取范围最小的策略信息-------->")
				result = s.getPolicy(policies, info.Src, info.Dst)
				l.Debug("匹配到的策略--->", zap.Any("result", result))
			}
			if result.Action == "deny" {
				return nil, nil
			}
			return result, nil
		}
	}
	l.Debug("<-------策略查询结束------->")
	return nil, nil
}
func (s *SrxHandler) Search(info *model.TTaskInfo) (*model.TDevicePolicy, error) {
	if s.error != nil {
		return nil, s.error
	}
	return s.search(info)
}
func (s *SrxHandler) GetCommand(dp *model.TDevicePolicy) string {
	return dp.Command
}

// 生成地址组策略
func (s *SrxHandler) geneAddressCmd(zone, name, address string) string {
	return fmt.Sprintf("set security zones security-zone %s address-book address %s %s\n", zone, name, address)
}

// 获取存在的端口名
// 获取组名
func (s *SrxHandler) getPortName(p int, protocol string) *model.TDevicePort {
	result := &model.TDevicePort{}
	if database.DB.Where("device_id = ? and protocol like ? and start = ? and end = ?", s.DeviceId, "%"+protocol+"%",
		p, p).First(result).Error != nil {
		return nil
	}
	return result
}

// 生成port策略
func (s *SrxHandler) genePortCmd(info *model.TTaskInfo) (portNames []string, portCmd string) {
	if info.Protocol != "ip" { // 协议为ip的端口直接any，否则要分割端口生成端口策略
		portNames = make([]string, 0)
		portCmd = ""
		// 将多个端口和范围端口拆分为单个端口
		for _, p := range strings.Split(info.DPort, ",") {
			rp, _ := utils.ParseRangePort(p)
			if rp.Start == 0 && rp.End == 65535 {
				return []string{"any"}, ""
			}
			for v := rp.Start; v <= rp.End; v++ {
				// TODO 需要确认端口具体生成策略，多个端口若一个端口存在组怎么处理
				if port := s.getPortName(v, info.Protocol); port == nil {
					portName := s.genePortName(info.Protocol, v)
					portCmd += fmt.Sprintf("set applications application %s protocol %s\n", portName, info.Protocol)
					portCmd += fmt.Sprintf("set applications application %s destination-port %d-%d\n", portName, v, v)
					portNames = append(portNames, portName)
				} else {
					portNames = append(portNames, port.Name)
				}
			}
		}
		return
	}
	return []string{"any"}, ""
}

// 生成策略命令
func (s *SrxHandler) genePolicyCmd(direction, name, denyPolicyName string, srcGroups, dstGroups, portNames []string) string {
	var zones string
	switch direction {
	case "inside":
		zones = fmt.Sprintf("from-zone %s to-zone %s", s.device.OutPolicy, s.device.InPolicy)
	case "outside":
		zones = fmt.Sprintf("from-zone %s to-zone %s", s.device.InPolicy, s.device.OutPolicy)
	default:
		s.error = fmt.Errorf("未知的策略方向")
	}
	policyCmd := ""
	for _, v := range srcGroups {
		policyCmd += fmt.Sprintf("set security policies %s policy %s match source-address %s\n", zones, name, v)
	}
	for _, v := range dstGroups {
		policyCmd += fmt.Sprintf("set security policies %s policy %s match destination-address %s\n", zones, name, v)
	}
	for _, v := range portNames {
		policyCmd += fmt.Sprintf("set security policies %s policy %s match application %s\n", zones, name, v)
	}
	policyCmd += fmt.Sprintf("set security policies %s policy %s then permit\n", zones, name)
	if denyPolicyName != "" {
		policyCmd += fmt.Sprintf("insert security policies %s policy %s before policy %s\n", zones, name, denyPolicyName)
	}
	return policyCmd
}

// 生成端口组名
func (s *SrxHandler) genePortName(protocol string, port int) string {
	if protocol == "ip" {
		return "any"
	}
	return fmt.Sprintf("%s-%d", strings.ToUpper(protocol), port)
}

func (s *SrxHandler) searchNat(info *model.TTaskInfo) *model.TDeviceSrxNat {
	result := &model.TDeviceSrxNat{}
	if err := database.DB.Where("device_id = ? and direction = ? and src like ? and pool = ? and dst like ?",
		s.DeviceId, info.Direction, "%"+info.Src+"%", info.PoolName, "%"+info.Dst+"%").First(result).Error; errors.Is(err, gorm.ErrRecordNotFound) { // 如果为空则为空
		return nil
	} else if err != nil { // 否则获取失败报个错
		s.error = fmt.Errorf("获取nat配置信息异常: %s", err.Error())
		return nil
	}
	return result
}

// 生成nat策略
func (s *SrxHandler) geneNatCmd(name string, info *model.TTaskInfo) string {
	if nat := s.searchNat(info); nat != nil {
		zap.L().Info(fmt.Sprintf("nat策略已存在: <%+v>", nat))
		info.ExistsConfig = nat.Command
	} else {
		zap.L().Info("生成nat策略------------------------------->")
		natCmd := ""
		for _, src := range strings.Split(info.Src, ",") {
			natCmd += fmt.Sprintf("set security nat source rule-set %s rule %s match source-address %s\n", info.NatName, name, src)
		}
		for _, dst := range strings.Split(info.Dst, ",") {
			natCmd += fmt.Sprintf("set security nat source rule-set %s rule %s match destination-address %s\n", info.NatName, name, dst)
		}
		// nat不做端口映射
		//for _, p := range strings.Split(info.DPort, ",") {
		//	natCmd += fmt.Sprintf("set security nat source rule-set %s rule %s match destination-port %s\n", info.NatName, name, p)
		//}
		natCmd += fmt.Sprintf("set security nat source rule-set %s rule %s then source-nat pool %s\n", info.NatName, name, info.PoolName)
		newNatName := s.getNewNatName()
		if newNatName != "" {
			natCmd += fmt.Sprintf("insert security nat source rule-set %s rule %s before rule %s", info.NatName, name, newNatName)
		}
		return natCmd
	}
	return ""
}

// 获取最新的nat配置
func (s *SrxHandler) getNewNatName() string {
	nat := model.TDeviceSrxNat{}
	if err := database.DB.Where("device_id = ?", s.DeviceId).First(&nat).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.error = fmt.Errorf("获取最新的nat地址异常: <%s>", err.Error())
	}
	return nat.Rule
}

// 获取地址所在的地址名
func (s *SrxHandler) getAddressGroup(address string) *model.TDeviceAddressGroup {
	group := &model.TDeviceAddressGroup{}
	if err := database.DB.Where("device_id = ? and address = ?", s.DeviceId, address).First(group).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else if err != nil {
		s.error = fmt.Errorf("根据设备ID<%d>, address获取地址组异常: <%s>", s.DeviceId, err.Error())
		return nil
	}
	return group
}

// GeneCommand 生成策略命令
func (s *SrxHandler) GeneCommand(jiraKey string, info *model.TTaskInfo) (string, error) {
	if s.error != nil {
		return "", s.error
	}
	l := zap.L().With(zap.Int("info_id", info.Id))
	l.Info("<------------------------生成策略命令------------------------>")
	l.Info(fmt.Sprintf("info: <%+v>", *info))
	l.Info("1. 获取当前开通策略关联设备的出入向策略名---------->")
	var (
		denyPolicyName = s.getInfoDenyPolicyName(info)
		name           = s.groupName(jiraKey, info)
		addressCmd     = ""
		portCmd        = ""
		srcGroups      = make([]string, 0)
		dstGroups      = make([]string, 0)
		portNames      = make([]string, 0)
		commands       = make([]string, 0)
	)
	l.Info("2. 生成源地址组策略命令---------------------->")
	// 如果办公网，地址组直接设置为办公网
	if info.Src == conf.BanGongWang || info.Src == conf.BanGongWangV6 {
		srcGroups = append(srcGroups, info.Src)
	} else {
		for _, v := range strings.Split(info.Src, ",") {
			if group := s.getAddressGroup(v); group != nil {
				srcGroups = append(srcGroups, group.Name)
			} else {
				// 如果是入向，则源地址的zone使用outPolicy  untrust
				if info.Direction == "inside" {
					addressCmd += s.geneAddressCmd(s.device.OutPolicy, v, v)
				} else { // 否则使用inPolicy  trust
					addressCmd += s.geneAddressCmd(s.device.InPolicy, v, v)
				}
				srcGroups = append(srcGroups, v)
			}
		}
	}

	l.Info("3. 生成目的地址组策略命令--------------------->")
	for _, v := range strings.Split(info.Dst, ",") {
		group := s.getAddressGroup(v)
		if group != nil {
			dstGroups = append(dstGroups, group.Name)
		} else {
			// 如果是入向，则目标地址的zone使用inPolicy  trust
			if info.Direction == "inside" {
				addressCmd += s.geneAddressCmd(s.device.InPolicy, v, v)
			} else { // 否则使用outPolicy  untrust
				addressCmd += s.geneAddressCmd(s.device.OutPolicy, v, v)
			}

			dstGroups = append(dstGroups, v)
		}
	}
	commands = append(commands, addressCmd)
	l.Info("4. 生成端口策略命令-------------------------->")
	if portNames, portCmd = s.genePortCmd(info); portCmd != "" {
		commands = append(commands, portCmd)
	}
	l.Info("5. 生成策略命令----------------------------->")
	policyCmd := s.genePolicyCmd(info.Direction, name, denyPolicyName, srcGroups, dstGroups, portNames)
	commands = append(commands, policyCmd)

	// 如果是出向访问并且出向网络类型不为空，则生成nat地址转换策略
	l.Info("6. 生成nat策略命令----------------------------->")
	if info.Direction == "outside" && info.PoolName != "" {
		if natCmd := s.geneNatCmd(name, info); natCmd != "" {
			commands = append(commands, natCmd)
		}
	}
	l.Info("<------------------------命令生成结束------------------------>")
	return strings.Trim(strings.Join(commands, "\n"), "\n"), nil
}

func (s *SrxHandler) parseAddressLine(line string) (direction, name, address string) {
	line = strings.TrimSpace(strings.ReplaceAll(line, "\\r", ""))
	lines := strings.Split(line, " ")
	direction = lines[4]
	name = lines[7]
	// 取地址名和地址
	address = lines[8]
	return
}

// 获取组名
func (s *SrxHandler) getPortNames(port, protocol string) (results []string, err error) {
	p, err := utils.ParseRangePort(port)
	if err != nil {
		return
	}
	if p.Start > 0 || p.End < 65535 {
		ports := make([]model.TDevicePort, 0)
		db := database.DB.Where("device_id = ? and protocol like ? and start <= ? and end >= ?", s.DeviceId, "%"+protocol+"%",
			p.Start, p.End).Find(&ports)
		if db.Error != nil {
			return nil, fmt.Errorf("根据port<%s>获取端口组异常: <%s>", port, db.Error.Error())
		}
		results = make([]string, 0)
		for _, p := range ports {
			results = append(results, p.Name)
		}
	}
	results = append(results, "any")
	return results, nil
}

func (s *SrxHandler) CheckNat(info *model.TTaskInfo) (err error) {
	return
}

type srxParse struct {
	base
	portText             string
	addressSetText       string
	addressText          string
	policyText           string
	natRuleSetText       string
	natPoolText          string
	deactivatePolicyText string
}

func (s *srxParse) parse() error {
	if e := s.device.UpdateParseStatus(ParseStatusInit); e != nil {
		return e
	}
	s.addLog("1. 获取完整配置信息--------------->")
	if e := s.getConfig(); e != nil {
		return e
	}

	s.addLog("2. 解析端口信息------------------>")
	ports := s.parsePortText()
	if e := s.savePorts(ports); e != nil {
		return e
	}

	s.addLog("3. 解析地址信息----------------->")
	addresses := s.parseAddressText()

	s.addLog("4. 解析地址组信息---------------->")
	addressSets := s.parseAddressSetText(addresses)
	if e := s.saveAddressSets(addresses, addressSets); e != nil {
		return e
	}

	s.addLog("5. 解析策略信息------------------>")
	policies := s.parsePolicyText()
	if e := s.savePolicies(policies, addresses, addressSets, ports); e != nil {
		return e
	}

	s.addLog("6. 解析nat信息------------------>")
	pools := s.parseNatPoolText()
	if e := s.saveNatPools(pools); e != nil {
		return e
	}
	if e := s.saveNatRuleSets(s.parseNatRuleSetText()); e != nil {
		return e
	}

	//s.addLog("7. 解析黑名单信息--->")
	//s.parseBlacklistGroupAddress(addressSets)
	return nil
}

func (s *srxParse) getConfig() error {
	commands := []*netApi2.Command{
		{Id: 1, Cmd: "show configuration security policies | display set"},
		{Id: 2, Cmd: "show configuration security zones | display set | match address-set"},
		{Id: 3, Cmd: "show configuration security zones | display set | match address | except address-set"},
		{Id: 4, Cmd: "show configuration applications | display set"},
		{Id: 5, Cmd: "show configuration security nat | display set | match rule-set"},
		{Id: 6, Cmd: "show configuration security nat | display set | match pool | match address"},
		{Id: 7, Cmd: "show configuration | display set | match deactivate"},
	}
	result, err := s.send(commands)
	if err != nil {
		return err
	}
	for _, v := range result {
		//fileName := fmt.Sprintf("%d.log", v.Id)
		//f, _ := os.Create(fileName)
		//f.WriteString(v.Result)
		//f.Close()
		switch v.Id {
		case 1:
			s.policyText = v.Result
		case 2:
			s.addressSetText = v.Result
		case 3:
			s.addressText = v.Result
		case 4:
			s.portText = v.Result
		case 5:
			s.natRuleSetText = v.Result
		case 6:
			s.natPoolText = v.Result
		case 7:
			s.deactivatePolicyText = v.Result
		}
	}
	return nil
}
func (s *srxParse) getConfig1() {
	fmt.Println("获取设备策略-------------->")
	fb, err := os.ReadFile("C:\\Users\\34607\\.wind\\profiles\\default.v10\\terminal\\logs\\南京\\南京SRX.log")
	if err != nil {
		s.error = err
		return
	}
	//newTexts := make([]string, 0)
	//for _, line := range strings.Split(string(fb), "\r") {
	//	newTexts = append(newTexts, strings.Split(line, "]")[1])
	//}

	s.policyText = string(fb)
	s.addressText = s.policyText
	s.addressSetText = s.policyText
	s.portText = s.policyText
	s.natRuleSetText = s.policyText
	s.natPoolText = s.policyText
}

type srxAddressSet struct {
	name  string
	zone  string
	items []string
}

func (s *srxParse) parseAddressSetText(addresses []*srxAddress) []*srxAddressSet {
	/*
		set security zones security-zone untrust address-book address-set blacklist address 134.224.69.89
		set security zones security-zone trust address-book address-set minsheng-bill address 172.28.159.58/32
		解析逻辑：
		循环每行，拆解zone，地址名，若zone和地址名与上一个定义的不一致，说明是新组，然后根据地址名从地址map中获取具体地址
	*/
	results := make([]*srxAddressSet, 0)
	addressM := s.makeAddressM(addresses)
	var addrSet *srxAddressSet
	for _, line := range strings.Split(s.addressSetText, "\r") {
		line = strings.TrimSpace(line)
		// 排除无效的行
		if !strings.HasPrefix(line, "set security zones security-zone") {
			continue
		}
		lines := splitLineBySpace(line)
		if len(lines) < 10 || lines[6] != "address-set" {
			continue
		}
		name := lines[7]
		zone := lines[4]
		// 如果是第一行，或者后面切组，方向和组名其中之一不相等，说明是新组
		if addrSet == nil || name != addrSet.name || zone != addrSet.zone {
			addrSet = &srxAddressSet{
				name: name,
				zone: zone,
			}
			results = append(results, addrSet)
		}
		// 如果地址组的zone在地址中不存在，说明配置有异常
		if _, ok := addressM[addrSet.zone]; !ok {
			s.addLog("地址组zone在地址map里未定义: %s", addrSet.zone)
			continue
		}
		// 从地址中根据zone和地址名取出具体地址信息添加到items中
		if addr, ok := addressM[addrSet.zone][lines[9]]; ok {
			addrSet.items = append(addrSet.items, addr...)
		} else {
			s.addLog("地址组地址在地址map里未定义: %s, %s", addrSet.zone, lines[9])
		}
	}
	s.addLog("解析到%d个地址组", len(results))
	return results
}

func (s *srxParse) saveAddressSets(addresses []*srxAddress, data []*srxAddressSet) error {
	bulks := make([]*model.TDeviceAddressGroup, 0)
	for _, v := range addresses {
		for _, addr := range v.items {
			bulks = append(bulks, &model.TDeviceAddressGroup{
				DeviceId:    s.device.Id,
				Name:        v.name,
				Address:     addr,
				Zone:        v.zone,
				AddressType: "address",
			})
		}
	}
	for _, v := range data {
		for _, addr := range v.items {
			bulks = append(bulks, &model.TDeviceAddressGroup{
				DeviceId:    s.device.Id,
				Name:        v.name,
				Zone:        v.zone,
				Address:     addr,
				AddressType: "address-set",
			})
		}
	}
	return s.saveGroup(bulks)
}

// 组装地址组map map[zone][name][]addr   {"trust": {"blacklist": ["1.1.1.1/32", “1.1.1.2/32"]}}}
func (s *srxParse) makeAddressSetM(data []*srxAddressSet) map[string]map[string][]string {
	result := make(map[string]map[string][]string)
	for _, v := range data {
		if _, ok := result[v.zone]; !ok {
			result[v.zone] = make(map[string][]string)
		}
		result[v.zone][v.name] = v.items
	}
	return result
}

type srxAddress struct {
	zone  string
	name  string
	items []string
}

func (s *srxParse) parseAddressText() []*srxAddress {
	/*
		set security zones security-zone trust address-book address 172.28.159.58/32 172.28.159.58/32
		set security zones security-zone trust address-book address 172.28.0.0/16 172.28.0.0/16
		set security zones security-zone trust address-book address 172.28.29.42/32 range-address 172.28.29.42 to 172.28.29.42
	*/
	results := make([]*srxAddress, 0)
	for _, line := range strings.Split(s.addressText, "\r") {
		line = strings.TrimSpace(line)
		// 排除无效的行
		if !strings.HasPrefix(line, "set security zones security-zone") {
			continue
		}
		lines := splitLineBySpace(line)
		if len(lines) < 9 || lines[6] != "address" {
			continue
		}
		item := &srxAddress{
			zone: lines[4],
			name: lines[7],
		}
		if lines[8] == "range-address" {
			item.items = append(item.items, s.getRangeAddress(lines[9], lines[11])...)
		} else {
			item.items = append(item.items, lines[8])
		}
		results = append(results, item)
	}
	s.addLog("解析到%d个地址", len(results))
	return results
}

// 组装地址map map[zone]map[name]value  {"trust": {"1.1.1.1": "1.1.1.1/32"}}
func (s *srxParse) makeAddressM(addresses []*srxAddress) map[string]map[string][]string {
	result := make(map[string]map[string][]string)
	for _, v := range addresses {
		if _, ok := result[v.zone]; !ok {
			result[v.zone] = make(map[string][]string)
			result[v.zone]["any"] = []string{"0.0.0.0/0"} // 添加any与0.0.0.0的映射
		}
		result[v.zone][v.name] = v.items
	}
	return result
}

type srxPolicy struct {
	fromZone             string
	toZone               string
	policy               string
	action               string
	sourceAddresses      []string
	destinationAddresses []string
	applications         []string
	commands             []string
	line                 int
	deactivate           bool
}

func (s *srxParse) parsePolicyText() []*srxPolicy {
	if s.error != nil {
		return nil
	}
	/*
		源对from-zone  src地址从from-zone的direction中获取
		目标对应to-zone  dst地址从to-zone的direction中获取
		set security policies from-zone trust to-zone untrust policy YWJS-86600-13175 match source-address 172.20.104.240/32
		set security policies from-zone trust to-zone untrust policy YWJS-86600-13175 match destination-address 132.33.88.128/32
		set security policies from-zone trust to-zone untrust policy YWJS-86600-13175 match application TCP-7001
		set security policies from-zone trust to-zone untrust policy YWJS-86600-13175 then permit
		deactivate security policies from-zone untrust to-zone trust policy hw-deny-116-228-151-188
	*/
	results := make([]*srxPolicy, 0)
	var item *srxPolicy
	for i, line := range strings.Split(s.policyText, "\r") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "then log") {
			continue
		}
		if !(strings.HasPrefix(line, "set security policies") || strings.HasPrefix(line, "deactivate security policies")) {
			continue
		}
		lines := splitLineBySpace(line)
		// 如果长度小于11，并且不是deactivate的策略跳过
		if len(lines) < 11 && lines[0] != "deactivate" {
			continue
		}
		policyName := lines[8]
		// 如果是第一行，或者换了个策略，则重新声明新的策略对象
		if item == nil || policyName != item.policy {
			item = &srxPolicy{
				policy:   policyName,
				fromZone: lines[4],
				toZone:   lines[6],
				line:     i + 1,
			}
			results = append(results, item)
		}
		if item == nil {
			s.addLog("无效的策略, text: %s, line: %d", line, i+1)
			continue
		}

		// 添加策略命令
		item.commands = append(item.commands, line)

		// 增加策略是否deactivate掉
		if lines[0] == "deactivate" {
			item.deactivate = true
			continue
		}

		switch lines[10] {
		case "source-address": // 添加源地址
			item.sourceAddresses = append(item.sourceAddresses, lines[11])
		case "destination-address": // 添加目标地址
			item.destinationAddresses = append(item.destinationAddresses, lines[11])
		case "application": // 添加端口
			item.applications = append(item.applications, lines[11])
		default:
			// 获取action
			if lines[9] == "then" {
				item.action = lines[10]
			}
		}
	}
	s.addLog("解析到%d条策略", len(results))
	return results
}

func (s *srxParse) savePolicies(policies []*srxPolicy, addresses []*srxAddress, addressesSet []*srxAddressSet, ports []*srxPort) error {
	addressM := s.makeAddressM(addresses)
	addressSetM := s.makeAddressSetM(addressesSet)
	portsM := s.makePortsM(ports)
	bulks := make([]*model.TDevicePolicy, 0)
	directionAnyM := map[string]*srxPolicy{}
	for _, v := range policies {
		// 如果没有端口，则设置为any
		if len(v.applications) == 0 {
			v.applications = []string{"any"}
		}
		// 解析方向 如果 from-zone in to-zone out则是出向， from-zone out to-zone in 则是入向
		direction := s.parseDirection(v.fromZone, v.toZone)
		// 根据from-zone to-zone获取第一个any地址，往下的策略则都是失效策略，一个方向只存在一个绝对any策略
		if strings.Join(v.sourceAddresses, "") == "any" && strings.Join(v.destinationAddresses, "") == "any" &&
			strings.Join(v.applications, "") == "any" {
			if _, ok := directionAnyM[direction]; !ok {
				directionAnyM[direction] = v
			}
		}
		valid := true
		// 判断是否是无效策略, any策略已经出现，并且与本策略名不一致
		if r, ok := directionAnyM[direction]; ok && r.policy != v.policy {
			valid = false
		}
		// 如果策略被deactivate掉，说明是无效策略
		if v.deactivate {
			valid = false
		}
		for _, p := range v.applications {
			item := &model.TDevicePolicy{
				DeviceId:  s.device.Id,
				Name:      v.policy,
				Action:    v.action,
				SrcGroup:  strings.Join(v.sourceAddresses, ","),
				DstGroup:  strings.Join(v.destinationAddresses, ","),
				Command:   strings.Join(v.commands, "\n"),
				Direction: direction,
				Line:      v.line,
				Valid:     valid,
			}
			// 转换源地址
			srcList := make([]string, 0)
			for _, addr := range v.sourceAddresses {
				// 源地址需要通过from-zone和地址从map中获取对应的地址信息
				srcList = append(srcList, getAddress(v.fromZone, addr, addressM, addressSetM))
			}
			// 转换目标地址
			dstList := make([]string, 0)
			for _, addr := range v.destinationAddresses {
				// 目标地址需要通过to-zone和地址从map中获取对应的地址信息
				dstList = append(dstList, getAddress(v.toZone, addr, addressM, addressSetM))
			}
			item.Src = strings.Join(srcList, ",")
			item.Dst = strings.Join(dstList, ",")

			// 转换端口
			item.PortGroup = p
			// 先从端口组里拿端口
			if r, ok := portsM[p]; ok { // 从端口组里取到的是个列表
				item.Port = strings.Join(r, ",")
			} else {
				// 再从端口对象里拿
				if r, ok := PortMaps[p]; ok {
					item.Port = r
				} else { // 否则可能定义的是直接的端口
					item.Port = p
				}
			}
			bulks = append(bulks, item)
		}
	}
	if e := s.savePolicy(bulks); e != nil {
		return e
	}
	s.addLog("更新设备any策略--->")
	// 保存设备出向any和入向any策略
	s.device.InPermitPolicyName = ""
	s.device.InDenyPolicyName = ""
	s.device.OutDenyPolicyName = ""
	s.device.OutPermitPolicyName = ""
	for direction, p := range directionAnyM {
		switch direction {
		case "inside":
			switch p.action {
			case "permit":
				s.device.InPermitPolicyName = p.policy
			case "deny":
				s.device.InDenyPolicyName = p.policy
			}
		case "outside":
			switch p.action {
			case "permit":
				s.device.OutPermitPolicyName = p.policy
			case "deny":
				s.device.OutDenyPolicyName = p.policy
			}
		}
	}
	if e := s.device.Save(nil); e != nil {
		s.addLog(e.Error())
		return e
	}
	return nil
}

func (s *srxParse) parseBlacklistGroupAddress(addressSets []*srxAddressSet) {
	s.addLog("解析黑名单地址组信息--->")
	deviceGroups, e := s.getBlacklistDeviceGroup()
	if e != nil {
		s.error = e
		return
	}
	if deviceGroups == nil {
		return
	}
	s.addLog("黑名单组有%d个", len(deviceGroups))
	addressSetM := s.makeAddressSetM(addressSets)
	// 获取出向策略区域内的地址组
	groupAddresses := addressSetM[s.device.OutPolicy]
	fmt.Println("解析到黑名单组数量--->", len(groupAddresses))
	for _, v := range deviceGroups {
		fmt.Println("解析地址组--->", v.Name)
		tx := database.DB.Begin()
		ips, ok := groupAddresses[v.Name]
		fmt.Println("组内多少个地址--->", len(ips))
		// 删除地址组内地址
		if e := tx.Delete(&model.TBlacklistDeviceGroupAddress{}, "device_group_id = ?", v.Id).Error; e != nil {
			tx.Rollback()
			s.addLog(fmt.Sprintf("清除组内地址异常, %s", e.Error()))
			continue
		}
		// 如果设备中无此地址组，则需要删除些地址组
		if !ok {
			if e := tx.Delete(v).Error; e != nil {
				tx.Rollback()
				s.addLog(fmt.Sprintf("删除无效地址组异常: <%s>", e.Error()))
				continue
			}
		} else {
			// 添加新的地址
			bulks := make([]*model.TBlacklistDeviceGroupAddress, 0)
			for _, ip := range ips {
				bulks = append(bulks, &model.TBlacklistDeviceGroupAddress{
					DeviceId:      v.DeviceId,
					DeviceGroupId: v.Id,
					Ip:            ip,
					IpType:        v.IpType,
				})
			}
			if e := s.saveGroupAddress(tx, bulks); e != nil {
				tx.Rollback()
				continue
			}
		}
		if e := tx.Commit().Error; e != nil {
			s.addLog(fmt.Sprintf("保存地址组<%s>地址信息异常: <%s>", v.Name, e.Error()))
		}
	}
}

// 根据from-zone和to-zone区分出方向
func (s *srxParse) parseDirection(fromZone, toZone string) string {
	// 解析方向 如果 from-zone in to-zone out则是出向， from-zone out to-zone in 则是入向
	switch {
	case fromZone == s.device.InPolicy && toZone == s.device.OutPolicy:
		return "outside"
	case fromZone == s.device.OutPolicy && toZone == s.device.InPolicy:
		return "inside"
	}
	return fmt.Sprintf("%s-%s", fromZone, toZone)
}

// 地址地址名或地址组中取出地址信息
func getAddress(zone, addr string, addressM map[string]map[string][]string, addressSetM map[string]map[string][]string) string {
	// 先从地址组中获取
	if _, ok := addressSetM[zone]; ok {
		if r, ok := addressSetM[zone][addr]; ok {
			return strings.Join(r, ",")
		}
	}
	// 再从地址中获取
	if _, ok := addressM[zone]; ok {
		if r, ok := addressM[zone][addr]; ok {
			return strings.Join(r, ",")
		}
	}
	return addr
}

// 获取无效策略
func (s *srxParse) getDeactivatePolicyNames() map[string]bool {
	/*
		deactivate security policies from-zone untrust to-zone trust policy hw-deny-172-24-248-151
		deactivate security policies from-zone untrust to-zone trust policy BanGongWang
	*/
	result := make(map[string]bool, 0)
	for _, line := range strings.Split(s.deactivatePolicyText, "\r") {
		line = strings.Trim(strings.TrimSpace(line), "\n")
		if !strings.HasPrefix(line, "deactivate") {
			continue
		}
		lineSplit := splitLineBySpace(line)
		result[lineSplit[len(lineSplit)-1]] = true
	}
	return result
}

func (s *srxParse) parsePortText() []*srxPort {
	if s.error != nil {
		return nil
	}
	/*
		set applications application TCP-8022 protocol tcp
		set applications application TCP-8022 destination-port 8022
		set applications application TCP_D8021_8050 destination-port 8021-8050
		set applications application TCP-30151 term TCP-30151 protocol tcp
		set applications application TCP-30151 term TCP-30151 destination-port 30151-30151
	*/
	results := make([]*srxPort, 0)
	var p *srxPort
	for i, line := range strings.Split(s.portText, "\r") {
		line = strings.Trim(strings.TrimSpace(line), "\n")
		if !strings.Contains(line, "set applications application") {
			continue
		}
		lines := splitLineBySpace(line)
		// set applications application TCP-8022 protocol tcp
		if lines[4] == "protocol" {
			p = &srxPort{
				name:     lines[3],
				protocol: lines[5],
			}
			results = append(results, p)
			continue
		}
		// set applications application TCP-30151 term TCP-30151 protocol tcp
		if lines[4] == "term" && lines[6] == "protocol" {
			p = &srxPort{
				name:     lines[3],
				protocol: lines[7],
			}
			results = append(results, p)
			continue
		}
		if p == nil {
			s.addLog("无效的端口策略, text: %s, num: %d", line, i+1)
			continue
		}
		// set applications application TCP-8022 destination-port 8022
		// set applications application TCP_D8021_8050 destination-port 8021-8050
		if lines[4] == "destination-port" || (lines[4] == "term" && lines[6] == "destination-port") {
			portStr := lines[len(lines)-1]
			// 8021-8050
			if strings.Contains(portStr, "-") {
				ps := strings.Split(portStr, "-")
				start, _ := portToInt(ps[0])
				end, _ := portToInt(ps[1])
				p.items = append(p.items, &srxPortItem{start: start, end: end})
			} else {
				start, _ := portToInt(portStr)
				p.items = append(p.items, &srxPortItem{start: start, end: start})
			}
			continue
		}
	}
	s.addLog("共解析到%d个端口组---------->", len(results))
	return results
}

// 解析端口
func (s *srxParse) savePorts(ports []*srxPort) error {
	bulks := make([]*model.TDevicePort, 0)
	for _, v := range ports {
		for _, item := range v.items {
			bulks = append(bulks, &model.TDevicePort{
				DeviceId: s.DeviceId,
				Name:     v.name,
				Protocol: v.protocol,
				Start:    item.start,
				End:      item.end,
			})
		}
	}
	return s.savePort(bulks)
}

// 组装端口
func (s *srxParse) makePortsM(ports []*srxPort) map[string][]string {
	results := make(map[string][]string)
	for _, v := range ports {
		for _, item := range v.items {
			results[v.name] = append(results[v.name], fmt.Sprintf("%d-%d", item.start, item.end))
		}
	}
	return results
}

type srxNatPool struct {
	natType   string // 分为source和destination 在解析rule-set时匹配对应的
	pool      string
	addresses []string
	commands  []string
	port      string
}

// 解析nat pool策略
func (s *srxParse) parseNatPoolText() []*srxNatPool {
	/*
		set security nat source pool snat_DeskVpn address 58.213.99.10/32
		set security nat source pool snat_pool address 58.213.99.13/32 to 58.213.99.14/32
		set security nat destination pool pool_30 address 172.31.15.22/32
		set security nat destination pool pool_30 address port 21
	*/
	results := make([]*srxNatPool, 0)
	var natPool *srxNatPool
	for i, line := range strings.Split(s.natPoolText, "\r") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "set security nat") {
			continue
		}
		lines := splitLineBySpace(line)
		// 排除无效行
		if len(lines) < 8 || lines[4] != "pool" {
			continue
		}
		if lines[7] != "port" {
			natPool = &srxNatPool{
				natType: lines[3],
				pool:    lines[5],
			}
		}
		if natPool == nil {
			s.addLog("无效nat, text: %s, num: %d", line, i+1)
			continue
		}
		natPool.commands = append(natPool.commands, line)
		// 添加端口
		if lines[7] == "port" {
			natPool.port = lines[8]
		} else {
			// 获取地址，只有为地址时，才往结果里添加
			if len(lines) > 8 && lines[8] == "to" {
				natPool.addresses = append(natPool.addresses, s.getRangeAddress(lines[7], lines[9])...)
			} else {
				natPool.addresses = append(natPool.addresses, lines[7])
			}
			results = append(results, natPool)
		}
	}
	s.addLog("解析到%d个nat pool", len(results))
	return results
}

// 保存nat pool
func (s *srxParse) saveNatPools(data []*srxNatPool) error {
	bulks := make([]*model.TDeviceNatPool, 0)
	for _, v := range data {
		bulks = append(bulks, &model.TDeviceNatPool{
			DeviceId: s.device.Id,
			Name:     v.pool,
			NatType:  v.natType,
			Command:  strings.Join(v.commands, "\n"),
			Address:  strings.Join(v.addresses, ","),
			Port:     v.port,
		})
	}
	return s.saveNatPool(bulks)
}

type srxNatRuleSet struct {
	fromZone             string
	toZone               string
	natType              string
	ruleSet              string
	rule                 string
	sourceAddresses      []string
	destinationAddresses []string
	destinationPorts     []*srxPortItem
	protocol             string
	pool                 string
	commands             []string
}

// 解析nat rule-set策略
func (s *srxParse) parseNatRuleSetText() []*srxNatRuleSet {
	/*
		set security nat destination rule-set xxx rule 103 match source-address 0.0.0.0/0
		set security nat destination rule-set xxx rule 103 match destination-address 10.10.10.10/32
		set security nat destination rule-set xxx rule 103 match destination-port 16384 to 32768
		set security nat destination rule-set xxx rule 103 match protocol udp
		set security nat destination rule-set xxx rule 103 then destination-nat pool pool_YXXX-5111_UDP16384-32768
	*/
	results := make([]*srxNatRuleSet, 0)
	var item *srxNatRuleSet
	var fromZone, toZone string
	for i, line := range strings.Split(s.natRuleSetText, "\r") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "set security nat") {
			continue
		}
		lines := splitLineBySpace(line)
		// 排除无效行
		if len(lines) < 9 || lines[4] != "rule-set" {
			continue
		}
		// 获取from zone和to zone
		switch lines[6] {
		case "from":
			fromZone = lines[8]
		case "to":
			toZone = lines[8]
		case "rule":
			if len(lines) < 11 {
				s.addLog("无效nat, text: %s, num: %d", line, i+1)
				continue
			}
			// 当切换新的rule时，声明新的对象
			if item == nil || item.rule != lines[7] || item.ruleSet != lines[5] {
				item = &srxNatRuleSet{
					natType:  lines[3],
					fromZone: fromZone,
					toZone:   toZone,
					ruleSet:  lines[5],
					rule:     lines[7],
				}
				results = append(results, item)
			}
			item.commands = append(item.commands, line)
			// 解析源地址
			switch lines[9] {
			case "source-address":
				item.sourceAddresses = append(item.sourceAddresses, lines[10])
			case "destination-address":
				item.destinationAddresses = append(item.destinationAddresses, lines[10])
			case "protocol":
				item.protocol = lines[10]
			case "source-nat":
				if lines[10] == "pool" {
					item.pool = lines[11]
				} else {
					item.pool = lines[10]
				}
			case "destination-nat":
				item.pool = lines[11]
			case "destination-port":
				// 如果存在to，说明要与上一个端口相连接，把上一个端口的end设置为此
				if lines[10] == "to" {
					end, _ := portToInt(lines[11])
					item.destinationPorts[len(item.destinationPorts)-1].end = end
				} else {
					start, _ := portToInt(lines[10])
					item.destinationPorts = append(item.destinationPorts, &srxPortItem{start: start, end: start})
				}
			default:
				s.addLog("无效nat, text: %s, num: %d", line, i+1)
			}
		}
	}
	return results
}

// 保存nat rule-set策略
func (s *srxParse) saveNatRuleSets(data []*srxNatRuleSet) error {
	bulks := make([]*model.TDeviceSrxNat, 0)
	for _, v := range data {
		ports := make([]string, 0)
		for _, p := range v.destinationPorts {
			ports = append(ports, fmt.Sprintf("%d-%d", p.start, p.end))
		}
		item := &model.TDeviceSrxNat{
			DeviceId:  s.device.Id,
			Direction: s.parseDirection(v.fromZone, v.toZone),
			Command:   strings.Join(v.commands, "\n"),
			Src:       strings.Join(v.sourceAddresses, ","),
			Dst:       strings.Join(v.destinationAddresses, ","),
			Protocol:  s.changeProtocol(v.protocol),
			DstPort:   strings.Join(ports, ","),
			Pool:      v.pool,
			Rule:      v.rule,
			NatType:   v.natType,
		}
		bulks = append(bulks, item)
	}
	return s.saveNatRuleSet(bulks)
}

// 转换协议
func (s *srxParse) changeProtocol(protocol string) string {
	if r, ok := ProtocolMaps[protocol]; ok {
		return r
	}
	return protocol
}

// 保存解析好的nat pool信息
func (s *srxParse) saveNatPool(data []*model.TDeviceNatPool) error {
	if data == nil || len(data) == 0 {
		return nil
	}
	tx := database.DB.Begin()
	if e := tx.Delete(&model.TDeviceNatPool{}, "device_id = ?", s.device.Id).Error; e != nil {
		tx.Rollback()
		return fmt.Errorf("清除nat pool信息失败, err: %w", e)
	}
	for i := 0; i*100 < len(data); i++ {
		r := (i + 1) * 100
		if (i+1)*100 > len(data) {
			r = len(data)
		}
		groups := data[i*100 : r]
		if err := tx.Create(&groups).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("保存设备nat pool信息异常: <%s>", err.Error())
		}
	}
	if e := tx.Commit().Error; e != nil {
		return fmt.Errorf("保存nat pool信息异常: <%w>", e)
	}
	return nil
}

// 保存解析好的nat pool信息
func (s *srxParse) saveNatRuleSet(data []*model.TDeviceSrxNat) error {
	if data == nil || len(data) == 0 {
		return nil
	}
	tx := database.DB.Begin()
	if e := tx.Delete(&model.TDeviceSrxNat{}, "device_id = ?", s.device.Id).Error; e != nil {
		tx.Rollback()
		return fmt.Errorf("清除nat rule-set信息异常: <%w>", e)
	}
	for i := 0; i*100 < len(data); i++ {
		r := (i + 1) * 100
		if (i+1)*100 > len(data) {
			r = len(data)
		}
		groups := data[i*100 : r]
		if err := tx.Create(&groups).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("保存设备nat rule-set信息异常: <%w>", err)
		}
	}
	if e := tx.Commit().Error; e != nil {
		return fmt.Errorf("保存nat rule-set信息异常: <%w>", e)
	}
	return nil
}
