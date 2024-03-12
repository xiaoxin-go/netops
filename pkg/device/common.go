package device

import (
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"net"
	"netops/conf"
	"netops/database"
	"netops/grpc_client/net_api"
	net_api2 "netops/grpc_client/protobuf/net_api"
	"netops/model"
	"netops/pkg/subnet"
	"netops/utils"
	"sort"
	"strconv"
	"strings"
	"time"
)

var PortMaps = map[string]string{
	"www":           "80",
	"http":          "80",
	"https":         "443",
	"ssh":           "22",
	"ftp":           "21",
	"sip":           "5060",
	"imap":          "143",
	"imap4":         "143",
	"smtp":          "25",
	"pop3":          "110",
	"ftp-data":      "20",
	"sqlnet":        "1433",
	"domain":        "53",
	"ntp":           "123",
	"netbios-ssn":   "139",
	"dns-udp":       "53",
	"dns-tcp":       "53",
	"netbios-dgm":   "138",
	"junos-https":   "443",
	"junos-ssh":     "22",
	"junos-ftp":     "21",
	"junos-tcp-any": "any",
}

var ProtocolMaps = map[string]string{
	"6":  "tcp",
	"17": "udp",
}

const (
	ParseStatusInit    = "init"
	ParseStatusFailed  = "failed"
	ParseStatusSuccess = "success"
)

type base struct {
	DeviceId      int
	error         error
	device        *model.TFirewallDevice
	operateLog    *model.TPolicyLog
	region        *model.TRegion
	deviceType    *model.TDeviceType
	backupCommand string
}

func (b *base) GeneCreateGroupCmd(groupName string) (result string) {
	return ""
}
func (b *base) Error() error {
	return b.error
}
func (b *base) ParseInvalidPolicy() {
	return
}

func (b *base) send(commands []*net_api2.Command) ([]*net_api2.Command, error) {
	client := net_api.NewClient(b.region.ApiServer)
	result, e := client.Show(&net_api2.ConfigRequest{
		DeviceType:     b.deviceType.Name,
		Host:           b.device.Host,
		Username:       b.device.Username,
		Password:       b.device.Password,
		EnablePassword: b.device.EnablePassword,
		Port:           int32(b.device.Port),
		Commands:       commands})
	if e != nil {
		return nil, e
	}
	return result, nil
}

func (b *base) geneBlackAddrName(addr string) string {
	return fmt.Sprintf("blacklist_%s", addr)
}

func (b *base) getRegion() {
	region := model.TRegion{}
	if e := region.FirstById(b.device.RegionId); e != nil {
		b.error = e
		return
	}
	b.region = &region
}
func (b *base) getDeviceType() {
	deviceType := model.TDeviceType{}
	if err := deviceType.FirstById(b.device.DeviceTypeId); err != nil {
		b.error = err
		return
	}
	b.deviceType = &deviceType
}
func (b *base) init() {
	device := model.TFirewallDevice{}
	if err := device.FirstById(b.DeviceId); err != nil {
		b.error = err
		return
	}
	b.device = &device
	b.getRegion()
	b.getDeviceType()
}

func (b *base) setParseStatus(status string) {
	if err := database.DB.Model(b.device).Update("parse_status", status).Error; err != nil {
		b.error = fmt.Errorf("修改解析状态异常: <%s>", err.Error())
	}
}

func (b *base) groupName(jiraKey string, info *model.TTaskInfo) (result string) {
	return fmt.Sprintf("%s-%d", jiraKey, info.Id)
}

// 获取range地址，范围地址  172.1.1.1-172.1.1.3
func (b *base) getRangeAddress(startIp, endIp string) []string {
	// 如果存在掩码，就把地址分割出来
	if strings.Contains(startIp, "/") {
		startIp = strings.Split(startIp, "/")[0]
		endIp = strings.Split(endIp, "/")[0]
	}
	startIpChars := strings.Split(startIp, ".")
	endIpChars := strings.Split(endIp, ".")
	if startIpChars[2] != endIpChars[2] {
		b.addLog(fmt.Sprintf("范围地址<%s-%s>超出C段", startIp, endIp))
	}
	endIpEnd := endIpChars[3]
	ipPre := strings.Join(startIpChars[0:3], ".")
	start, _ := strconv.Atoi(startIpChars[3])
	end, _ := strconv.Atoi(endIpEnd)
	ipList := make([]string, 0)
	for i := start; i <= end; i++ {
		ip := fmt.Sprintf("%s.%d/32", ipPre, i)
		ipList = append(ipList, ip)
	}
	return ipList
}

// 将IP地址切分，并转换成int
// input: "172.1.1.1"
// output: [172, 1, 1, 1]
func ipToOnes(ip string) []int {
	result := make([]int, 0)
	for _, v := range strings.Split(ip, ".") {
		i, _ := strconv.Atoi(v)
		result = append(result, i)
	}
	return result
}

// 获取deny策略名
func (b *base) getInfoDenyPolicyName(info *model.TTaskInfo) (result string) {
	if info.Direction == "inside" {
		result = b.device.InDenyPolicyName
	} else {
		result = b.device.OutDenyPolicyName
	}
	return
}
func (b *base) geneSrcGroupName(groupName string) string {
	return fmt.Sprintf("%s-SRC", groupName)
}
func (b *base) geneDstGroupName(groupName string) string {
	return fmt.Sprintf("%s-DST", groupName)
}
func (b *base) genePortGroupName(groupName string) string {
	return fmt.Sprintf("%s-SERVICE", groupName)
}

// 返回组名
func (b *base) getPortNames(port, protocol string) (results []string, err error) {
	p, err := utils.ParseRangePort(port)
	if err != nil {
		return
	}
	ports := make([]model.TDevicePort, 0)
	db := database.DB.Where("device_id = ? and protocol like ? and start <= ? and end >= ?", b.DeviceId,
		"%"+protocol+"%", p.Start, p.End).Find(&ports)
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

// 查询策略
func (b *base) search(info *model.TTaskInfo) *model.TDevicePolicy {
	log := zap.L().With(zap.Int("infoId", info.Id))
	log.Info("<-------单条策略查询------->")
	log.Info(fmt.Sprintf("TaskInfo: <%+v>", info))
	log.Info(fmt.Sprintf("Device: <%+v>", b.device))
	log.Info("1. 根据源目地址模糊匹配符合条件的策略---------->")
	var (
		result    *model.TDevicePolicy
		db        *gorm.DB
		portNames = []string{"any"}
	)
	db = database.DB.Where("device_id = ? and dst like ? and protocol in ? and direction = ? and action = ?",
		b.DeviceId, "%"+info.Dst+"%", []string{info.Protocol, "ip"}, info.Direction, "permit")
	if info.Direction == "inside" && (info.Src == conf.BanGongWang || info.Src == conf.BanGongWangV6) {
		db = db.Where("src_group = ?", info.Src)
	} else {
		db = db.Where("src like ?", "%"+info.Src+"%")
	}

	var err error
	if info.Protocol != "ip" {
		zap.L().Info("根据端口和协议获取端口所在的组--------->")
		if portNames, err = b.getPortNames(info.DPort, info.Protocol); err != nil {
			b.error = err
			return nil
		}
		log.Info(fmt.Sprintf("port:<%s> -> portNames:<%+v>", info.DPort, portNames))
		db = db.Where("port = ? or port = ? or port in ?", "any", info.DPort, portNames)
	}
	policies := make([]*model.TDevicePolicy, 0)
	db = db.Find(&policies)
	if db.Error != nil {
		b.error = fmt.Errorf("查询策略表异常: <%s>", db.Error.Error())
		return nil
	}
	if len(policies) > 0 {
		log.Info("匹配到策略信息------------------>")
		result = policies[0]
		log.Info(fmt.Sprintf("匹配到的策略: <%+v>", *result))
	} else {
		log.Info("2. 根据基本信息未匹配到相应的策略，开始进行网段的匹配------->")
		// 如果没有匹配的策略，则获取所有地址组，根据地址组来查询
		// 先获取源地址为网段的策略信息
		log.Info("先根据基本条件进行过滤------------>")
		subnetPolicies := make([]*model.TDevicePolicy, 0)
		db = database.DB.Where("action = ? and device_id = ? and direction = ?",
			"permit", b.DeviceId, info.Direction)
		if info.Protocol != "ip" {
			db = db.Where("port = ? or port = ? or port in ?", "any", info.DPort, portNames)
		}
		db = db.Find(&subnetPolicies)
		if db.Error != nil {
			b.error = fmt.Errorf("查询策略地址组异常: <%s>", db.Error.Error())
		}
		// 根据源目地址找到符合条件的策略
		log.Info("3. 根据源目地址从策略表中匹配策略--------->")
		policies = b.getSubnetPolicy(subnetPolicies, info.Src, info.Dst)
		if len(policies) > 0 {
			// 匹配出最小的策略
			log.Info("根据匹配到的网段策略, 获取范围最小的策略信息-------->")
			policy := b.getPolicy(policies, info.Src, info.Dst)
			log.Info(fmt.Sprintf("匹配到的策略: <%+v>", policy))
			result = policy
		}
	}
	log.Info("<-------策略查询结束------->")
	return nil
}

func (b *base) SearchAll(src, dst, port string) ([]*model.TDevicePolicy, error) {
	if b.error != nil {
		return nil, b.error
	}
	l := zap.L().With(zap.String("func", "SearchAll"))
	l.Debug("策略查询--->", zap.String("src", src), zap.String("dst", dst), zap.String("port", port))
	policies := make([]*model.TDevicePolicy, 0)
	l.Debug("2. 根据设备和方向获取所有策略--->")
	// 先获取源地址为网段的策略信息
	subnetPolicies := make([]*model.TDevicePolicy, 0)
	db := database.DB.Where("device_id = ?", b.DeviceId)
	if port != "" {
		l.Debug("根据端口和协议获取端口所在的组--->")
		var portNames []string
		var err error
		if portNames, err = b.getPortNames(port, ""); err != nil {
			return nil, err
		}
		l.Debug("根据port和port_names过滤", zap.String("port", port), zap.Any("port_names", portNames))
		db = db.Where("port = ? or port = ? or port_group in ?", "any", port, portNames)
	}
	if e1 := db.Order("line").Find(&subnetPolicies).Error; e1 != nil {
		return nil, fmt.Errorf("查询策略表异常, err: %w", e1)
	}
	l.Debug("3. 根据源目地址获取符合条件的策略------------->")
	policies = b.getSubnetPolicy(subnetPolicies, src, dst)
	l.Debug(fmt.Sprintf("匹配到的策略条数: <%d>", len(policies)))
	l.Debug("<-------查询结束------->")
	return nil, nil
}

// 获取IP地址所在的组，获取条件，组包含的地址必须与传入的多个地址完全一致
func (b *base) getAddressGroup(address string) *model.TDeviceAddressGroup {
	ips := strings.Split(address, ",")
	// 1. 获取包含第一个地址的所有组
	groups := make([]*model.TDeviceAddressGroup, 0)
	if database.DB.Where("device_id = ? and address = ?", b.DeviceId, ips[0]).Find(&groups).Error != nil || len(groups) == 0 {
		return nil
	}
	// 2. 根据组名获取所有组包含的所有地址
	gns := make([]string, 0)
	for _, v := range groups {
		gns = append(gns, v.Name)
	}
	groups = make([]*model.TDeviceAddressGroup, 0)
	if database.DB.Where("device_id = ? and name in ?", b.DeviceId, gns).Find(&groups).Error != nil || len(groups) == 0 {
		return nil
	}
	// 3. 将这些相同的组和地址拼接起来
	gms := make(map[string][]string, 0)
	for _, v := range groups {
		if _, ok := gms[v.Name]; ok {
			gms[v.Name] = append(gms[v.Name], v.Address)
		} else {
			gms[v.Name] = []string{v.Address}
		}
	}
	// 4. 循环这些组，对组内地址进行排序，内容完全一致则选用此组
	// gm={"YWJS-1234-1-SRC": "1.1.1.1/32,1.1.1.2/32", "YWJS-1234-2-DST", "1.1.1.2/32,1.1.1.3/32"}
	sort.Strings(ips)
	for n, v := range gms {
		sort.Strings(v)
		if strings.Join(v, ",") == strings.Join(ips, ",") {
			return &model.TDeviceAddressGroup{Name: n}
		}
	}
	return nil
}

// 获取端口所在的组，端口组内地址和端口必须一致，端口可能为多个，处理逻辑复杂
func (b *base) getDevicePortName(port, protocol string) string {
	// 获取所有包含端口的组
	portNames := make([]string, 0)
	portStrList := make([]string, 0)
	for _, p := range strings.Split(port, ",") {
		dps := make([]*model.TDevicePort, 0)
		rp, _ := utils.ParseRangePort(p)
		portStrList = append(portStrList, fmt.Sprintf("%d-%d", rp.Start, rp.End))
		// 获取包含端口的所有组
		if database.DB.Where("device_id = ? and start = ? and end = ? and protocol = ?", b.DeviceId, rp.Start, rp.End, protocol).Find(&dps).Error != nil {
			return ""
		}
		for _, v := range dps {
			portNames = append(portNames, v.Name)
		}
	}
	// 获取所有组内的端口
	dps := make([]*model.TDevicePort, 0)
	if database.DB.Where("device_id = ? and name in ?", b.DeviceId, portNames).Find(&dps).Error != nil {
		return ""
	}
	// 拼接组和端口
	portNameMaps := make(map[string][]string, 0)
	for _, v := range dps {
		if _, ok := portNameMaps[v.Name]; ok {
			portNameMaps[v.Name] = append(portNameMaps[v.Name], fmt.Sprintf("%d-%d", v.Start, v.End))
		} else {
			portNameMaps[v.Name] = []string{fmt.Sprintf("%d-%d", v.Start, v.End)}
		}
	}
	// 循环地址组包含的端口，排序后对比，取出第一个完全一致的地址组
	sort.Strings(portStrList)
	for portName, v := range portNameMaps {
		sort.Strings(v)
		if strings.Join(portStrList, ",") == strings.Join(v, ",") {
			return portName
		}
	}
	return ""
}

// 保存解析好的地址组对应地址到数据库
func (b *base) saveGroup(data []*model.TDeviceAddressGroup) error {
	if data == nil || len(data) == 0 {
		return nil
	}
	tx := database.DB.Begin()
	if e := tx.Delete(&model.TDeviceAddressGroup{}, "device_id = ?", b.device.Id).Error; e != nil {
		tx.Rollback()
		return fmt.Errorf("清除历史地址组配置失败, err: %w", e)
	}
	for i := 0; i*100 < len(data); i++ {
		r := (i + 1) * 100
		if (i+1)*100 > len(data) {
			r = len(data)
		}
		groups := data[i*100 : r]
		if e := tx.Create(&groups).Error; e != nil {
			tx.Rollback()
			return fmt.Errorf("保存设备地址组失败, err: %w", e)
		}
	}
	if e := tx.Commit().Error; e != nil {
		return fmt.Errorf("保存地址组commit失败: %w", e)
	}
	return nil
}

// 保存黑名单组地址
func (b *base) saveGroupAddress(tx *gorm.DB, data []*model.TBlacklistDeviceGroupAddress) error {
	if data == nil || len(data) == 0 {
		return nil
	}
	for i := 0; i*100 < len(data); i++ {
		r := (i + 1) * 100
		if (i+1)*100 > len(data) {
			r = len(data)
		}
		groups := data[i*100 : r]
		if e := tx.Create(&groups).Error; e != nil {
			zap.L().Error("保存黑名单地址组失败", zap.Error(e))
			return fmt.Errorf("保存黑名单组地址失败, err: %w", e)
		}
	}
	return nil
}

// 保存解析好的端口信息到数据库
func (b *base) savePort(data []*model.TDevicePort) error {
	if data == nil {
		return nil
	}
	tx := database.DB.Begin()
	if e := new(model.TDevicePort).DeleteByDeviceId(b.DeviceId, tx); e != nil {
		tx.Rollback()
		return e
	}
	for i := 0; i*100 < len(data); i++ {
		r := (i + 1) * 100
		if (i+1)*100 > len(data) {
			r = len(data)
		}
		ports := data[i*100 : r]
		if e := tx.Create(&ports).Error; e != nil {
			tx.Rollback()
			zap.L().Error("保存调和端口失败", zap.Error(e))
			return fmt.Errorf("保存设备端口失败, err: %w", e)
		}
	}
	if e := tx.Commit().Error; e != nil {
		zap.L().Error("保存端口commit失败", zap.Error(e))
		return fmt.Errorf("保存端口commit失败, err: %w", e)
	}
	return nil
}

// 保存解析好的策略信息到数据库
func (b *base) savePolicy(data []*model.TDevicePolicy) error {
	if data == nil || len(data) == 0 {
		return nil
	}
	tx := database.DB.Begin()
	if e := new(model.TDevicePolicy).DeleteByDeviceId(b.DeviceId, tx); e != nil {
		tx.Rollback()
		return e
	}
	for i := 0; i*100 < len(data); i++ {
		r := (i + 1) * 100
		if (i+1)*100 > len(data) {
			r = len(data)
		}
		policies := data[i*100 : r]
		if err := tx.Create(&policies).Error; err != nil {
			tx.Rollback()
			zap.L().Error("保存策略信息失败", zap.Error(err))
			return fmt.Errorf("保存策略信息失败, err: %w", err)
		}
	}
	if e := tx.Commit().Error; e != nil {
		zap.L().Error("保存策略信息commit失败", zap.Error(e))
		return fmt.Errorf("保存策略信息失败, err: %w", e)
	}
	return nil
}

// 保存策略命中数
func (b *base) savePolicyHitCount(data []*model.TDevicePolicyHitCount) error {
	if data == nil || len(data) == 0 {
		return nil
	}
	tx := database.DB.Begin()
	if e := tx.Delete(&model.TDevicePolicyHitCount{}, "device_id = ?", b.device.Id).Error; e != nil {
		tx.Rollback()
		zap.L().Error("删除旧策略命中数失败", zap.Error(e), zap.Int("device_id", b.device.Id))
		return fmt.Errorf("删除旧策略命中数失败, err: %w", e)
	}
	for i := 0; i*100 < len(data); i++ {
		r := (i + 1) * 100
		if (i+1)*100 > len(data) {
			r = len(data)
		}
		policies := data[i*100 : r]
		if err := tx.Create(&policies).Error; err != nil {
			zap.L().Error("保存策略命中数失败", zap.Error(err))
			return fmt.Errorf("保存策略命中数失败, err: %w", err)
		}
	}
	return nil
}

func (b *base) SearchNat(info *model.TTaskInfo) *model.TDeviceNat {
	return nil
}

// 保存nat
func (b *base) saveNat(data []*model.TDeviceNat) error {
	if data == nil || len(data) == 0 {
		return nil
	}
	tx := database.DB.Begin()
	if e := tx.Delete(&model.TDeviceNat{}, "device_id = ?", b.DeviceId).Error; e != nil {
		tx.Rollback()
		zap.L().Error("清除历史nat数据失败", zap.Error(e))
		return fmt.Errorf("清除历史nat信息失败, err: %w", e)
	}
	b.addLog("保存nat------------->")
	for i := 0; i*100 < len(data); i++ {
		r := (i + 1) * 100
		if (i+1)*100 > len(data) {
			r = len(data)
		}
		nats := data[i*100 : r]
		if e := tx.Create(&nats).Error; e != nil {
			tx.Rollback()
			return fmt.Errorf("保存Nat信息异常: <%w>", e)
		}
	}
	if e := tx.Commit().Error; e != nil {
		return fmt.Errorf("保存nat信息异常")
	}
	return nil
}

// 根据传入的设备策略，获取包含源目地址的策略
func (b *base) getSubnetPolicy(policies []*model.TDevicePolicy, src, dst string) (result []*model.TDevicePolicy) {
	srcPolicies := make([]*model.TDevicePolicy, 0)
	result = make([]*model.TDevicePolicy, 0)
	for _, sp := range policies {
		// 如果源地址是办公网，则判断src_group是办公网的即可
		if (src == conf.BanGongWang && sp.SrcGroup == conf.BanGongWang) || (src == conf.BanGongWangV6 && sp.SrcGroup == conf.BanGongWangV6) {
			srcPolicies = append(srcPolicies, sp)
			continue
		}
		// 如果源IP等于空，或者匹配到网段则加入源IP策略列表中
		if src == "" {
			srcPolicies = append(srcPolicies, sp)
			continue
		}
		for _, s := range strings.Split(sp.Src, ",") {
			if strings.Contains(s, "/32") && s != src { // 如果是32位地址，则直接判断相等
				continue
			}
			if ok, _ := subnet.IsNet(s, src); !ok { // 不是32位地址，则判断是否在此网段中
				continue
			}
			srcPolicies = append(srcPolicies, sp)
			break // 当前条策略，匹配到一个就可以
		}
	}
	// 通过源IP策略列表继续过滤目标IP地址
	for _, sp := range srcPolicies {
		// 如果目标IP等于空，或者匹配到网段则加入源IP策略列表中
		if dst == "" {
			result = append(result, sp)
			continue
		}
		for _, d := range strings.Split(sp.Dst, ",") {
			// 完全匹配地址
			if strings.Contains(d, "/32") && d != dst { // 如果是32位地址，则直接判断相等
				continue
			}
			// 匹配网段
			if ok, _ := subnet.IsNet(d, dst); !ok { // 不是32位地址，则判断是否在此网段中
				continue
			}
			result = append(result, sp)
			break
		}
	}
	return
}

// 根据符合条件的策略信息获取范围最小的策略信息
func (b *base) getPolicy(policies []*model.TDevicePolicy, src, dst string) (result *model.TDevicePolicy) {
	result = &model.TDevicePolicy{}
	for _, v := range policies {
		// 循环源地址，取出当前行最小的源地址
		minSrc := b.getMinSubnet(src, v.Src)
		// 循环目标地址，取出当前行最小的目标地址
		minDst := b.getMinSubnet(dst, v.Dst)
		if result.Src == "" {
			result = v
			result.Src = minSrc
			result.Dst = minDst
			continue
		}

		// 和上一条策略对比范围，如果比比上一条符合的策略网段大，则跳过
		if ok, _ := subnet.IsNet(result.Src, minSrc); !ok {
			continue
		}
		// 如果最小的地址和源地址不相等（说明比上条更小），则取当前策略
		if minSrc != result.Src {
			result = v
			result.Src = minSrc
			result.Dst = minDst
		} else {
			// 否则对比目标网段，取目标网段最小的策略
			if ok, _ := subnet.IsNet(result.Dst, minDst); ok {
				result = v
				result.Src = minSrc
				result.Direction = minDst
			}
		}
	}
	return
}

// 根据传入的设备策略，获取包含源目地址的策略
func (b *base) getSubnetNat(nats []*model.TDeviceNat, src, dst string) (result []*model.TDeviceNat) {
	srcNats := make([]*model.TDeviceNat, 0) // 源地址匹配的nat策略信息
	result = make([]*model.TDeviceNat, 0)
	for _, nat := range nats {
		// 如果源IP等于空，或者匹配到网段则加入源IP策略列表中
		if src != "" {
			for _, s := range strings.Split(nat.Network, ",") {
				if strings.Contains(s, "/32") && s != src { // 如果是32位地址，则直接判断相等
					continue
				}
				if ok, _ := subnet.IsNet(s, src); !ok { // 不是32位地址，则判断是否在此网段中
					continue
				}
				srcNats = append(srcNats, nat)
				break
			}
		} else {
			srcNats = append(srcNats, nat)
		}
	}
	// 通过源地址匹配的nat策略列表继续过滤目标IP地址
	for _, nat := range srcNats {
		// 如果目标IP等于空，或者匹配到网段则加入源IP策略列表中
		if dst != "" {
			for _, d := range strings.Split(nat.Destination, ",") {
				// 完全匹配地址
				if strings.Contains(d, "/32") && d != dst { // 如果是32位地址，则直接判断相等
					continue
				}
				// 匹配网段
				if ok, _ := subnet.IsNet(d, dst); !ok { // 不是32位地址，则判断是否在此网段中
					continue
				}
				result = append(result, nat)
				break
			}
		} else {
			result = append(result, nat)
		}
	}
	return
}

// 根据符合条件的策略信息获取范围最小的策略信息
func (b *base) getNat(nats []*model.TDeviceNat, src, dst string) (result *model.TDeviceNat) {
	result = &model.TDeviceNat{}
	for _, nat := range nats {
		// 循环源地址，取出当前行最小的源地址网段
		minSrc := b.getMinSubnet(src, nat.Network)
		// 循环目标地址，取出当前行最小的目标地址网段
		minDst := b.getMinSubnet(dst, nat.Destination)
		if result.Network == "" {
			result = nat
			result.Network = minSrc
			result.Destination = minDst
			continue
		}
		// 和上一条策略对比范围，如果比比上一条符合的策略网段大，则跳过
		if ok, _ := subnet.IsNet(result.Network, minSrc); !ok {
			continue
		}
		// 如果最小的地址和源地址不相等（说明比上条更小），则取当前策略
		if minSrc != result.Network {
			result = nat
			result.Network = minSrc
			result.Destination = minDst
		} else {
			// 否则对比目标网段，取目标网段最小的策略
			if ok, _ := subnet.IsNet(result.Destination, minDst); ok {
				result = nat
				result.Network = minSrc
				result.Direction = minDst
			}
		}
	}
	return
}

// 获取单条策略里最小的IP网段
func (b *base) getMinSubnet(ip, subnets string) (minSubnet string) {
	for _, s := range strings.Split(subnets, ",") {
		// 不是源地址的网段排除掉
		if ok, _ := subnet.IsNet(s, ip); !ok {
			continue
		}
		if minSubnet == "" {
			minSubnet = s
		}
		// 和最小的对比，如果比最小的还小，则替换最小的IP
		if ok, _ := subnet.IsNet(minSubnet, s); ok {
			minSubnet = s
		}
	}
	return
}

// 添加操作日志
func (b *base) addLog(content string, args ...any) {
	content = fmt.Sprintf(content, args...)
	now := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf("[%s] [%s]\n", now, content)
	if b.operateLog != nil {
		b.operateLog.Content += msg
		database.DB.Save(b.operateLog)
	} else {
		b.operateLog = &model.TPolicyLog{}
		b.operateLog.DeviceId = b.device.Id
		b.operateLog.Content = msg
		b.operateLog.DeviceType = "firewall"
		database.DB.Create(b.operateLog)
	}
}

// 获取黑名单地址组
func (b *base) getBlacklistDeviceGroup() ([]*model.TBlacklistDeviceGroup, error) {
	blackDeviceIds := make([]int, 0)
	if e := database.DB.Model(&model.TBlacklistDevice{}).Where("device_id = ? and enabled = 1", b.device.Id).Pluck("device_id", &blackDeviceIds).Error; e != nil {
		return nil, fmt.Errorf("获取黑名单设备信息异常: <%w>", e)
	}
	fmt.Println("黑名单设备------->", len(blackDeviceIds))
	if len(blackDeviceIds) == 0 {
		return nil, nil
	}
	deviceGroups := make([]*model.TBlacklistDeviceGroup, 0)
	if e := database.DB.Where("device_id in ?", blackDeviceIds).Find(&deviceGroups).Error; e != nil {
		return nil, fmt.Errorf("获取设备<%d>黑名单地址组信息异常: <%w>", b.device.Id, e)
	}
	return deviceGroups, nil
}

// 转换子网掩码为简写
func ipMaskSimple(ip, mask string) string {
	ip = fmt.Sprintf("%s/%d", ip, maskSimple(mask))
	return ip
}

func maskSimple(mask string) int {
	bs := make([]byte, 4)
	for i, v := range strings.Split(mask, ".") {
		vi, _ := strconv.Atoi(v)
		bs[i] = byte(vi)
	}
	ones, _ := net.IPv4Mask(bs[0], bs[1], bs[2], bs[3]).Size()
	return ones
}

func ipMask(subnet string) (ip, mask string) {
	if utils.GetIpType(subnet) == "ipv4" {
		return ipv4Mask(subnet)
	}
	ips := strings.Split(subnet, "/")
	return ips[0], ips[1]
}

// 地址简写转换为子网掩码
func ipv4Mask(subnet string) (ip, mask string) {
	if !strings.Contains(subnet, "/") {
		subnet = fmt.Sprintf("%s/32", subnet)
	}
	_, n, err := net.ParseCIDR(subnet)
	if err != nil {
		fmt.Printf("n: <%s>, err: <%s>, subnet: <%s>\n", n, err, subnet)
		return
	}
	mask = fmt.Sprintf("%d.%d.%d.%d", n.Mask[0], n.Mask[1], n.Mask[2], n.Mask[3])
	ip = n.IP.String()
	return
}

// 转换端口为int
func portToInt(port string) (result int, err error) {
	if p, ok := PortMaps[port]; ok {
		port = p
	}
	result, err = strconv.Atoi(port)
	return
}

// 根据空格分割单条数据，处理中间多个空格
func splitLineBySpace(line string) []string {
	return splitLine(line, " ")
}

// 根据符号分割
func splitLine(line, char string) []string {
	lineSplits := strings.Split(line, char)
	// 对行进行处理，避免中间出现多个空格
	lines := make([]string, 0)
	for _, v := range lineSplits {
		v = strings.TrimSpace(v)
		if v != "" {
			lines = append(lines, v)
		}
	}
	return lines
}

// 判断是否是any
func isAny(str string) bool {
	m := map[string]bool{
		"any":  true,
		"any4": true,
		"any6": true,
	}
	return m[str]
}
