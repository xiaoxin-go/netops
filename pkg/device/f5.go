package device

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"netops/conf"
	"netops/database"
	"netops/grpc_client/net_api"
	net_api2 "netops/grpc_client/protobuf/net_api"
	"netops/model"
	subnet2 "netops/pkg/subnet"
	"netops/utils"
	"strings"
	"time"
)

type SourceAddressTranslation struct {
	Type string `json:"type"`
}
type ProfilesReference struct {
	Link string `json:"link"`
}
type Persist struct {
	Name          string `json:"name"`
	Partition     string `json:"partition"`
	TmDefault     string `json:"tmDefault"`
	NameReference struct {
		Link string `json:"link"`
	} `json:"nameReference"`
}

type VsResult struct {
	Name                     string                   `json:"name"`
	Partition                string                   `json:"partition"`
	Source                   string                   `json:"source"`
	Destination              string                   `json:"destination"`
	Enabled                  bool                     `json:"enabled"`
	Disable                  bool                     `json:"disable"`
	IpProtocol               string                   `json:"ipProtocol"`
	SourceAddressTranslation SourceAddressTranslation `json:"sourceAddressTranslation"`
	Pool                     string                   `json:"pool"`
	ProfilesReference        ProfilesReference        `json:"profilesReference"`
	Rules                    []string                 `json:"rules"`
	Persist                  []Persist                `json:"persist"`
}

type Profiles struct {
	Name       string `json:"name"`
	Partition  string `json:"partition"`
	FullPath   string `json:"fullPath"`
	Generation int    `json:"generation"`
}

type PoolResult struct {
	Name             string            `json:"name"`
	Partition        string            `json:"partition"`
	Monitor          string            `json:"monitor"`
	MembersReference ProfilesReference `json:"membersReference"`
}

type PoolMemberResult struct {
	Name      string `json:"name"`
	Partition string `json:"partition"`
	State     string `json:"state"`
}

func NewF5Policy(deviceId int) *F5Policy {
	f := &F5Policy{}
	f.DeviceId = deviceId
	f.init()
	return f
}

type F5Policy struct {
	F5Parse
}

func (f *F5Policy) Error() error {
	return f.error
}

func (f *F5Policy) Search(info *model.TTaskInfo) (*model.TF5Vs, error) {
	if f.error != nil {
		return nil, f.error
	}
	f5Vs := &model.TF5Vs{}
	if e := f5Vs.FirstByDestination(fmt.Sprintf("%s:%s", info.Dst, info.DPort)); e != nil {
		return nil, e
	}
	return f5Vs, nil
}
func (f *F5Policy) GetCommand(vs *model.TF5Vs) (command string) {
	return fmt.Sprintf("Name:%s", vs.Name)
}

type sourceAddressTranslation struct {
	Pool string `json:"pool"`
	Type string `json:"type"`
}
type member struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}
type poolCommand struct {
	Name      string    `json:"name"`
	Partition string    `json:"partition"`
	Monitor   string    `json:"monitor"`
	Members   []*member `json:"members"`
}
type vsCommand struct {
	Name                     string                   `json:"name"`
	Partition                string                   `json:"partition"`
	Destination              string                   `json:"destination"`
	Disabled                 bool                     `json:"disabled"`
	IpProtocol               string                   `json:"ipProtocol"`
	Mask                     string                   `json:"mask"`
	Pool                     string                   `json:"pool"`
	SourceAddressTranslation sourceAddressTranslation `json:"sourceAddressTranslation"`
}

func (f *F5Policy) geneVsName(jiraKey, dst string, info *model.TTaskInfo) string {
	return fmt.Sprintf("%s-%d_%s_%s%s", jiraKey, info.Id, dst, strings.ToUpper(info.Protocol), info.DPort)
}
func (f *F5Policy) genePoolName(vsName string) string {
	return fmt.Sprintf("%s_POOL", vsName)
}

func (f *F5Policy) GeneCommand(jiraKey string, info *model.TTaskInfo) error {
	if f.error != nil {
		return f.error
	}
	l := zap.L().With(zap.String("func", "GeneCommand"), zap.String("jira_key", jiraKey))
	l.Debug("生成f5策略--->", zap.Any("info", info))
	// 把ipv4和v6的.和:转成下划线
	dst := strings.ReplaceAll(strings.ReplaceAll(info.Dst, ".", "_"), ":", "_")
	vsName := f.geneVsName(jiraKey, dst, info)
	poolName := f.genePoolName(vsName)
	var mask, destination string
	if utils.GetIpType(info.Dst) == conf.IpTypeV4 {
		mask = "255.255.255.255"
		destination = fmt.Sprintf("%s:%s", info.Dst, info.DPort)
	} else {
		mask = "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"
		destination = fmt.Sprintf("%s.%s", info.Dst, info.DPort)
	}
	l.Debug("生成vs命令--->")
	vsCmd := vsCommand{
		Name:        vsName,
		Partition:   "Common",
		Destination: fmt.Sprintf("/Common/%s", destination),
		Disabled:    false,
		IpProtocol:  info.Protocol,
		Mask:        mask,
		Pool:        fmt.Sprintf("/Common/%s", poolName),
	}
	if info.SNat != "" {
		vsCmd.SourceAddressTranslation = sourceAddressTranslation{Type: info.SNat}
	}
	// 如果snat等于snat则需要取snat的pool
	if info.SNat == "snat" {
		poolName, err := f.getF5SnatPoolByDeviceId(info.Dst, f.DeviceId)
		if err != nil {
			return err
		}
		if poolName == "" {
			return fmt.Errorf("网段<%s> 在设备<%s>下未定义snat", info.Dst, f.device.Name)
		}
		vsCmd.SourceAddressTranslation.Pool = poolName
	}
	l.Debug("生成pool命令--->")
	poolCmd := poolCommand{
		Name:      poolName,
		Partition: "Common",
		Monitor:   fmt.Sprintf("/Common/%s", strings.ToLower(info.Protocol)),
	}
	members := make([]*member, 0)
	for _, v := range strings.Split(info.Node, ",") {
		var memberName string
		if utils.GetIpType(info.Dst) == conf.IpTypeV4 {
			memberName = fmt.Sprintf("%s:%s", v, info.NodePort)
		} else {
			memberName = fmt.Sprintf("%s.%s", v, info.NodePort)
		}
		members = append(members, &member{
			Name:    memberName,
			Address: v,
		})
	}
	poolCmd.Members = members
	l.Debug("命令生成完成", zap.Any("vs_command", vsCmd), zap.Any("pool_command", poolCmd))
	info.VsCommand = vsCommandToString(vsCmd)
	info.PoolCommand = poolCommandToString(poolCmd)
	return nil
}

// GetF5SnatPoolByDeviceId 根据设备ID和vs网段获取对应的snat pool名称
func (f *F5Policy) getF5SnatPoolByDeviceId(dst string, deviceId int) (poolName string, err error) {
	snatPools, e := new(model.TF5SnatPool).FindByDeviceId(deviceId)
	if e != nil {
		return "", e
	}
	minSubnet := ""
	for _, item := range snatPools {
		for _, subnet := range strings.Split(item.Subnet, ",") {
			if result, _ := subnet2.IsNet(subnet, dst); result {
				if minSubnet == "" {
					minSubnet = subnet
					poolName = item.Name
					continue
				}
				// 最小优先，如果s比最小的还小，则替换
				if ok, _ := subnet2.IsNet(minSubnet, subnet); ok {
					minSubnet = subnet
					poolName = item.Name
				}
			}
		}
	}
	return
}

func vsCommandToString(command vsCommand) string {
	m := make(map[string]interface{})
	m["name"] = command.Name
	m["partition"] = command.Partition
	m["destination"] = command.Destination
	m["mask"] = command.Mask
	m["disabled"] = command.Disabled
	m["ipProtocol"] = command.IpProtocol
	m["pool"] = command.Pool
	if command.SourceAddressTranslation.Type != "" {
		m["sourceAddressTranslation"] = command.SourceAddressTranslation
	}
	vb, _ := json.Marshal(m)
	return string(vb)
}
func poolCommandToString(command poolCommand) string {
	vb, _ := json.Marshal(command)
	return string(vb)
}

func (f *F5Policy) SendConfig(info *model.TTaskInfo) error {
	if f.error != nil {
		return f.error
	}
	l := zap.L().With(zap.String("func", "SendConfig"), zap.Int("device_id", f.device.Id))
	l.Info("执行f5配置", zap.Any("info", info))
	l.Info("创建pool--->")
	if e := f.createPool(info.PoolCommand); e != nil {
		l.Error("创建pool失败", zap.Error(e), zap.Any("commands", info.PoolCommand))
		return fmt.Errorf("创建pool失败, err: %w", e)
	}
	if e := f.createVs(info.VsCommand); e != nil {
		l.Error("创建vs失败", zap.Error(e), zap.Any("commands", info.VsCommand))
		return fmt.Errorf("创建vs失败, err: %w", e)
	}
	return nil
}
func (f *F5Policy) createPool(command string) error {
	return f.send("/mgmt/tm/ltm/pool/", "POST", command, nil)
}
func (f *F5Policy) createVs(command string) error {
	return f.send("/mgmt/tm/ltm/virtual/", "POST", command, nil)
}

type F5Parse struct {
	DeviceId   int
	device     *model.TNLBDevice
	error      error
	client     *net_api.Client
	operateLog *model.TPolicyLog
}

func (f *F5Parse) Error() error {
	return f.error
}

func (f *F5Parse) setParseStatus(status string) {
	if err := database.DB.Model(f.device).Update("parse_status", status).Error; err != nil {
		f.error = fmt.Errorf("修改解析状态异常: <%s>", err.Error())
	}
}

func (f *F5Parse) init() {
	nlbDevice := model.TNLBDevice{}
	if err := nlbDevice.FirstById(f.DeviceId); err != nil {
		f.error = err
		return
	}
	f.device = &nlbDevice
	f.connect()
}

// 添加操作日志
func (f *F5Parse) addLog(content string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf("[%s] [%s]\n", now, content)
	if f.operateLog != nil {
		f.operateLog.Content += msg
		database.DB.Save(f.operateLog)
	} else {
		f.operateLog = &model.TPolicyLog{}
		f.operateLog.DeviceId = f.device.Id
		f.operateLog.Content = msg
		f.operateLog.DeviceType = "nlb"
		database.DB.Create(f.operateLog)
	}
}

func (f *F5Parse) clear() {
	database.DB.Delete(&model.TF5Vs{}, "device_id = ?", f.DeviceId)
	database.DB.Delete(&model.TF5Pool{}, "device_id = ?", f.DeviceId)
	database.DB.Delete(&model.TF5PoolNode{}, "device_id = ?", f.DeviceId)
}

func (f *F5Parse) ConnectGrpc() {
	f.connect()
}
func (f *F5Parse) CloseGrpc() {
	if f.client != nil {
		f.client.Close()
	}
}

func (f *F5Parse) connect() {
	region := model.TRegion{}
	if e := region.FirstById(f.device.RegionId); e != nil {
		f.error = e
	}
	f.client = net_api.NewClient(region.ApiServer)
}

func (f *F5Parse) ParseConfig() error {
	if f.error != nil {
		return f.error
	}
	defer f.client.Close()
	f.addLog("初始化解析状态--->")
	if e := f.device.UpdateParseStatus(ParseStatusInit); e != nil {
		return e
	}
	f.addLog("清除历史数据--->")
	f.clear()
	go func() {
		f.addLog("解析vs--->")
		if e := f.parseVs(); e != nil {
			f.parseFailed(e)
			return
		}
		f.addLog("解析pool信息")
		if e := f.parsePool(); e != nil {
			f.parseFailed(e)
			return
		}
		f.parseSuccess()
	}()
	return nil
}

func (f *F5Parse) parseSuccess() {
	f.operateLog.Status = "success"
	f.addLog("策略解析成功!")
	f.setParseStatus(ParseStatusSuccess)
}
func (f *F5Parse) parseFailed(e error) {
	f.operateLog.Status = "failed"
	f.addLog(e.Error())
	_ = f.device.UpdateParseStatus(ParseStatusFailed)
}

func (f *F5Parse) send(uri, method, params string, result any) error {
	req := &net_api2.HttpRequest{
		Url:      fmt.Sprintf("https://%s%s", f.device.Host, uri),
		Method:   method,
		Username: f.device.Username,
		Password: f.device.Password,
		Params:   params,
	}
	message, e := f.client.Http(req)
	if e != nil {
		return e
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal([]byte(message), result); err != nil {
		zap.L().Error("解析GRPC HTTP结果失败", zap.Error(e), zap.String("message", message))
		return fmt.Errorf("解析GRPC HTTP结果失败, err: %w", err)
	}
	return nil
}

func (f *F5Parse) parsePool() error {
	result := struct {
		Items []PoolResult `json:"items"`
	}{}
	if e := f.send("/mgmt/tm/ltm/pool/", "GET", "", &result); e != nil {
		return e
	}
	f5PoolList := make([]*model.TF5Pool, 0)
	for _, v := range result.Items {
		f5Pool := &model.TF5Pool{
			DeviceId:  f.DeviceId,
			Name:      v.Name,
			Partition: v.Partition,
			Monitor:   v.Monitor,
		}
		f5PoolList = append(f5PoolList, f5Pool)
	}
	f.addLog("5. 保存pool信息--------------------->")
	if e := f.savePool(f5PoolList); e != nil {
		f.addLog(e.Error())
		return e
	}
	f.addLog("6. 获取pool member信息------------------>")
	for _, pool := range f5PoolList {
		if e := f.parseMember(pool.Name); e != nil {
			return e
		}
	}
	return nil
}
func (f *F5Parse) parseMember(poolName string) error {
	result := struct {
		Items []*PoolMemberResult `json:"items"`
	}{}
	if e := f.send(fmt.Sprintf("/mgmt/tm/ltm/pool/%s/members", poolName), "GET", "", &result); e != nil {
		return e
	}
	f5PoolNodeList := make([]*model.TF5PoolNode, 0)
	for _, v := range result.Items {
		f5PoolNode := &model.TF5PoolNode{
			DeviceId:  f.DeviceId,
			PoolName:  poolName,
			Name:      v.Name,
			Partition: v.Partition,
			State:     v.State,
		}
		f5PoolNodeList = append(f5PoolNodeList, f5PoolNode)
	}
	if e := f.savePoolNode(f5PoolNodeList); e != nil {
		return e
	}
	return nil
}

func (f *F5Parse) parseVs() error {
	vsTrafficGroup, e := f.parseVsTrafficGroup()
	if e != nil {
		return e
	}
	result := struct {
		Items []*VsResult `json:"items"`
	}{}
	if e := f.send("/mgmt/tm/ltm/virtual/", "GET", "", &result); e != nil {
		return e
	}
	f5VsList := make([]*model.TF5Vs, 0)
	for _, v := range result.Items {
		//profiles := f.getProfiles(strings.TrimPrefix(v.ProfilesReference.Link, "https://localhost"))
		f5Vs := &model.TF5Vs{
			DeviceId:                 f.DeviceId,
			Name:                     v.Name,
			Partition:                v.Partition,
			Source:                   v.Source,
			Destination:              strings.TrimPrefix(v.Destination, fmt.Sprintf("/%s/", v.Partition)),
			SourceAddressTranslation: v.SourceAddressTranslation.Type,
			Enabled:                  v.Enabled == true,
			//ProfilesReference:        profiles.Name,
			Pool:         strings.TrimPrefix(v.Pool, fmt.Sprintf("/%s/", v.Partition)),
			Protocol:     v.IpProtocol,
			Rules:        strings.Join(v.Rules, ","),
			TrafficGroup: strings.TrimPrefix(vsTrafficGroup[v.Name], fmt.Sprintf("/%s/", v.Partition)),
		}
		if len(v.Persist) > 0 {
			f5Vs.Persist = v.Persist[0].Name
		}
		f5VsList = append(f5VsList, f5Vs)
	}
	if e := f.saveVs(f5VsList); e != nil {
		return e
	}
	return nil
}
func (f *F5Parse) parseVsTrafficGroup() (map[string]string, error) {
	result := struct {
		Items []struct {
			Name         string `json:"name"`
			TrafficGroup string `json:"trafficGroup"`
		} `json:"items"`
	}{}
	if e := f.send("/mgmt/tm/ltm/virtual-address/", "GET", "", &result); e != nil {
		return nil, e
	}
	vsGroup := make(map[string]string)
	for _, v := range result.Items {
		vsGroup[v.Name] = v.TrafficGroup
	}
	return vsGroup, nil
}

func (f *F5Parse) getProfiles(uri string) (*Profiles, error) {
	result := struct {
		Items []*Profiles `json:"items"`
	}{}
	if e := f.send(uri, "GET", "", &result); e != nil {
		return nil, e
	}
	if len(result.Items) > 0 {
		return result.Items[0], nil
	}
	return nil, nil
}

// 保存F5vs信息
func (f *F5Parse) saveVs(data []*model.TF5Vs) error {
	if data == nil {
		return nil
	}
	for i := 0; i*100 < len(data); i++ {
		r := (i + 1) * 100
		if (i+1)*100 > len(data) {
			r = len(data)
		}
		bulks := data[i*100 : r]
		if e := new(model.TF5Vs).BulkCreate(bulks); e != nil {
			return e
		}
	}
	return nil
}

// 保存保存F5pool信息
func (f *F5Parse) savePool(data []*model.TF5Pool) error {
	if data == nil {
		return nil
	}
	for i := 0; i*100 < len(data); i++ {
		r := (i + 1) * 100
		if (i+1)*100 > len(data) {
			r = len(data)
		}
		bulks := data[i*100 : r]
		if err := database.DB.Create(&bulks).Error; err != nil {
			return fmt.Errorf("保存POOL信息异常: <%s>", err.Error())
		}
	}
	return nil
}

// 保存保存F5Node信息
func (f *F5Parse) savePoolNode(data []*model.TF5PoolNode) error {
	if data == nil {
		return nil
	}
	for i := 0; i*100 < len(data); i++ {
		r := (i + 1) * 100
		if (i+1)*100 > len(data) {
			r = len(data)
		}
		bulks := data[i*100 : r]
		if err := database.DB.Create(&bulks).Error; err != nil {
			return fmt.Errorf("保存POOLNode信息异常: <%s>", err.Error())
		}
	}
	return nil
}
