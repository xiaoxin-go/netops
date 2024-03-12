package task

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"netops/conf"
	"netops/database"
	net_api2 "netops/grpc_client/net_api"
	"netops/grpc_client/protobuf/net_api"
	"netops/model"
	device2 "netops/pkg/device"
	"netops/pkg/parse"
	"netops/pkg/subnet"
	"netops/utils"
	"sort"
	"strconv"
	"strings"
	"time"
)

func NewTaskHandler() *taskHandler {
	return &taskHandler{}
}
func NewTaskHandlerById(id int) *taskHandler {
	h := &taskHandler{}
	h.getTask(id)
	h.init()
	return h
}

type taskHandler struct {
	task          *model.TTask
	implementType *model.TImplementType
	region        *model.TRegion
	traceId       string
	operator      string
	operateLog    *model.TTaskOperateLog
	Err           error
}

func (h *taskHandler) GetTaskType() string {
	return h.task.Type
}

func (h *taskHandler) Task() *model.TTask {
	return h.task
}
func (h *taskHandler) init() {
	if h.task.Type == conf.TaskTypeFirewall {
		h.getImplementType()
	}
	h.getRegion()
}
func (h *taskHandler) getRegion() {
	region := model.TRegion{}
	if e := region.FirstById(h.task.RegionId); e != nil {
		h.Err = e
		return
	}
	h.region = &region
}
func (h *taskHandler) getImplementType() {
	implementType := model.TImplementType{}
	if e := implementType.FirstByName(h.task.ImplementType); e != nil {
		h.Err = e
		return
	}
	h.implementType = &implementType
}
func (h *taskHandler) getTask(id int) (result *model.TTask) {
	task := model.TTask{}
	if e := task.FirstById(id); e != nil {
		h.Err = e
		return
	}
	h.task = &model.TTask{}
	return
}
func (h *taskHandler) SetOperator(operator string) {
	h.operator = operator
}
func (h *taskHandler) GetJiraKey() string {
	return h.task.JiraKey
}
func (h *taskHandler) AddTask(task *model.TTask) error {
	jiraKey := strings.TrimSpace(task.JiraKey)
	// 1. 获取前端传入的工单号
	l := zap.L().With(zap.String("func", "AddTask"), zap.String("jira_key", jiraKey))
	l.Info("创建工单--->")
	l.Info("查询是否存在历史工单--->")
	exists, e := new(model.TTask).ExistsByJiraKey(jiraKey)
	if e != nil {
		return e
	}
	if exists {
		return fmt.Errorf("工单信息已存在, 工单号: %s", jiraKey)
	}
	l.Info("从jira平台获取工单信息-------------->")
	jh := utils.NewJiraHandler()
	issue, e := jh.GetIssue(jiraKey)
	if e != nil {
		return e
	}
	if issue.Key == "" {
		return fmt.Errorf("工单信息不存在，工单号: %s", jiraKey)
	}
	issueToTask(issue, task)
	task.Type = getTaskType(issue)
	l.Info("根据工单区域和环境获取网络区域和模板--->",
		zap.String("region", task.JiraRegion),
		zap.String("environment", task.JiraEnvironment))
	region, err := getTaskRegion(task.JiraRegion, task.JiraEnvironment)
	if err != nil {
		return err
	}
	task.RegionId = region.Id
	l.Info("保存工单信息--->")
	task.Status = conf.TaskStatusInit
	if e := task.Create(); e != nil {
		return e
	}
	h.task = task
	h.getRegion()
	l.Info("添加成功---------------------------->")
	return nil
}

func getTaskType(issue *utils.Issue) string {
	if strings.Contains(issue.Fields.Description, "f5") || strings.Contains(issue.Fields.Description, "F5") || issue.Fields.ImplementContent.Value == "F5开通" {
		return "nlb"
	}
	return "firewall"
}

func issueToTask(issue *utils.Issue, task *model.TTask) {
	task.Assignee = issue.Fields.Assignee.Name
	task.Creator = issue.Fields.Creator.DisplayName
	task.Department = strings.Join([]string{issue.Fields.Department.Value, issue.Fields.Department.Child.Value}, "-")
	task.Description = issue.Fields.Description
	task.JiraRegion = issue.Fields.Region.Value
	task.JiraEnvironment = issue.Fields.Environment.Value
	task.ImplementType = issue.Fields.ImplementContent.Value
	//task.NetworkOpeningRange = issue.Fields.NetworkOpeningRange.Value
	task.JiraStatus = issue.Fields.Status.Name
	task.Summary = issue.Fields.Summary
}
func getTaskRegion(regionName, environment string) (*model.TRegion, error) {
	issueType := model.TIssueType{}
	if e := issueType.FirstByRegionEnvironment(regionName, environment); e != nil {
		return nil, e
	}
	region := model.TRegion{}
	if e := region.FirstById(issueType.RegionId); e != nil {
		return nil, fmt.Errorf("根据工单类型获取属地失败, 工单类型: %s, err: %w", issueType.Description, e)
	}
	return &region, nil
}

// UpdateStatus 更新任务状态
func (h *taskHandler) UpdateStatus(status string) error {
	if h.Err != nil {
		return h.Err
	}
	if e := h.task.UpdateStatus(status); e != nil {
		return e
	}
	return nil
}

// UpdateStatusByOperate 根据操作更新任务状态
func (h *taskHandler) UpdateStatusByOperate(operate string) error {
	if h.Err != nil {
		return h.Err
	}
	taskStatus := model.TTaskStatus{}
	if e := taskStatus.FirstByOperate(operate); e != nil {
		return e
	}
	if e := h.task.UpdateStatus(taskStatus.TaskNextStatus); e != nil {
		return e
	}
	return nil
}

func (h *taskHandler) splitInfos(data *model.TTaskInfo) []*model.TTaskInfo {
	infos := make([]*model.TTaskInfo, 0)
	srcL := strings.Split(data.Src, ",")
	dstL := []string{data.Dst}
	portL := []string{data.DPort}
	if data.StaticIp == "" {
		dstL = strings.Split(data.Dst, ",")
		portL = strings.Split(data.DPort, ",")
	}
	for _, src := range srcL {
		src = strings.Trim(src, " ")
		for _, dst := range dstL {
			dst = strings.Trim(dst, " ")
			for _, port := range portL {
				port = strings.Trim(port, " ")
				ic := *data
				ic.Src = src
				ic.Dst = dst
				ic.DPort = port
				ic.TaskId = h.task.Id
				ic.Status = conf.TaskStatusInit
				infos = append(infos, &ic)
			}
		}
	}
	return infos
}

// AddInfo 添加策略信息
func (h *taskHandler) AddInfo(data *model.TTaskInfo) error {
	if h.Err != nil {
		return h.Err
	}
	l := zap.L().With(zap.Int("task_id", h.task.Id), zap.String("func", "AddInfo"))
	l.Info("添加策略信息--->", zap.Any("data", data))
	l.Info("1. 拆分工作项--->")
	infos := h.splitInfos(data)
	l.Info("2. 校验基本格式--->")
	if e := h.checkInfoStyle(infos); e != nil {
		l.Error(e.Error())
		return e
	}
	l.Info("3. 校验类型--->")
	if e := h.checkInfoType(infos); e != nil {
		l.Error(e.Error())
		return e
	}
	l.Info("4. 去重重复策略并保存策略--->")
	if e := h.excludeSameInfos(infos); e != nil {
		l.Error(e.Error())
		return e
	}
	l.Info("策略添加成功")
	return nil
}

// 添加F5工单策略
func (h *taskHandler) addNlbInfo(data *model.TTaskInfo) error {
	l := zap.L().With(zap.Int("task_id", h.task.Id), zap.String("func", "addNlbInfo"))
	l.Info("添加策略信息--->", zap.Any("data", data))
	l.Info("1. 拆分工作项--->")
	infos := h.splitNlbInfos(data)
	l.Info("2. 校验基本格式--->")
	if e := h.checkNlbInfoStyle(infos); e != nil {
		l.Error(e.Error())
		return e
	}
	l.Info("3. 校验类型--->")
	if e := h.checkNlbInfoType(infos); e != nil {
		l.Error(e.Error())
		return e
	}
	l.Info("4. 去重重复策略并保存策略--->")
	if e := h.excludeSameInfos(infos); e != nil {
		l.Error(e.Error())
		return e
	}
	l.Info("策略添加成功")
	return nil
}

// 分割负载均衡策略
func (h *taskHandler) splitNlbInfos(data *model.TTaskInfo) []*model.TTaskInfo {
	infos := make([]*model.TTaskInfo, 0)
	nodes := strings.Split(data.Node, ";")
	for _, node := range nodes {
		node = strings.TrimSpace(node)
		ic := *data
		ic.Node = node
		ic.TaskId = h.task.Id
		ic.Status = conf.TaskStatusInit
		infos = append(infos, &ic)
	}
	return infos
}

// 根据现有策略，校验当前添加策略是否重复
func (h *taskHandler) checkExists(existsInfos []*model.TTaskInfo, info *model.TTaskInfo) (result bool) {
	for _, i := range existsInfos {
		for _, src := range strings.Split(i.Src, ",") {
			for _, dst := range strings.Split(i.Dst, ",") {
				for _, d := range strings.Split(i.DPort, ",") {
					if src == info.Src && dst == info.Dst && info.Protocol == i.Protocol && d == info.DPort {
						return true
					}
				}
			}
		}
	}
	return
}

// 校验当前添加NLB策略是否重复
func (h *taskHandler) checkNlbExists(existsInfos []*model.TTaskInfo, info *model.TTaskInfo) (result bool) {
	for _, i := range existsInfos {
		if info.Dst == i.Dst && info.DPort == i.DPort && info.NodePort == i.NodePort && info.Protocol == i.Protocol {
			for _, node := range strings.Split(i.Node, ",") {
				if info.Node == node {
					return true
				}
			}
		}
	}
	return
}

// 去除重复策略信息，并保存策略
func (h *taskHandler) excludeSameInfos(data []*model.TTaskInfo) error {
	existInfos, e := new(model.TTaskInfo).FindByTaskId(h.task.Id)
	if e != nil {
		return e
	}
	// 添加信息时对数据进行去重
	bulks := make([]*model.TTaskInfo, 0)
	for _, item := range data {
		// 不存在才添加
		if h.task.Type == conf.TaskTypeFirewall {
			if !h.checkExists(existInfos, item) {
				bulks = append(bulks, item)
			}
		} else {
			if !h.checkNlbExists(existInfos, item) {
				bulks = append(bulks, item)
			}
		}
	}
	if e := new(model.TTaskInfo).BulkCreate(bulks); e != nil {
		return e
	}
	return nil
}

// 排除校验的地址, 如果源地址是办公网，则不校验原地址
func (h *taskHandler) noCheckSrc(src string) bool {
	return src == conf.BanGongWang || src == conf.BanGongWangV6
}

// 校验工单数据格式
func (h *taskHandler) checkInfoStyle(data []*model.TTaskInfo) error {
	for _, item := range data {
		zap.L().Debug("校验工作项信息", zap.Any("item", item))
		// 增加办公网地址的校验, 如果源地址是办公网，则不校验原地址
		if !h.noCheckSrc(item.Src) {
			// 解析源地址
			if src, e := utils.ParseIP(item.Src); e != nil {
				return fmt.Errorf("源地址格式不正确, src: %s, err: %w", item.Src, e)
			} else {
				item.Src = src
			}
		}
		// 解析目标地址
		if dst, e := utils.ParseIP(item.Dst); e != nil {
			return fmt.Errorf("目标地址格式不正确, dst: %s, err: %w", item.Dst, e)
		} else {
			item.Dst = dst
		}
		// IPV4不能访问IPV6
		if (strings.Contains(item.Src, ".") && strings.Contains(item.Dst, ":")) ||
			(strings.Contains(item.Src, ":") && strings.Contains(item.Dst, ".")) {
			return fmt.Errorf("%s->%s, ipv4和ipv6不能相互访问", item.Src, item.Dst)
		}
		// 内部地址存在，端口必须是一对一的
		if item.StaticIp != "" {
			if _, err := utils.ParsePort(item.DPort); err != nil {
				return err
			}
			if staticIp, e := utils.ParseIP(item.StaticIp); e != nil {
				return fmt.Errorf("内部地址格式不正确, 地址: %s, err: %w", item.StaticIp, e)
			} else {
				item.StaticIp = staticIp
			}
			if _, err := utils.ParsePort(item.StaticPort); err != nil {
				return fmt.Errorf("内部端口解析失败, 端口: %s, err: %w", item.StaticPort, err)
			}
		} else {
			// 校验端口格式
			if _, err := utils.ParseRangePort(item.DPort); err != nil {
				return fmt.Errorf("目标端口解析失败, 端口: %s, err: %w", item.DPort, err)
			}
		}
	}
	return nil
}

// CheckInfoStyle 校验工单数据格式
func (h *taskHandler) checkNlbInfoStyle(data []*model.TTaskInfo) error {
	for _, item := range data {
		zap.L().Debug("校验工作项信息", zap.Any("item", item))
		// 校验目标IP
		if !utils.VerifyIP(item.Dst) {
			return fmt.Errorf("目标地址格式不正确, 地址: %s", item.Dst)
		}
		// 校验端口格式
		if _, err := utils.ParsePort(item.DPort); err != nil {
			return fmt.Errorf("目标端口解析失败, 端口: %s, err: %w", item.DPort, err)
		}
		// 校验node格式
		if !utils.VerifyIP(item.Node) {
			return fmt.Errorf("node地址格式不正确, node: %s", item.Node)
		}
		// 校验node端口格式
		if _, err := utils.ParsePort(item.NodePort); err != nil {
			return fmt.Errorf("node端口解析失败, 端口: %s, err: %w", item.NodePort, err)
		}
	}
	return nil
}

// CheckInfoType 校验工单数据类型
func (h *taskHandler) checkInfoType(data []*model.TTaskInfo) error {
	for _, info := range data {
		zap.L().Debug("校验工作项类型", zap.Any("item", info))
		var (
			src *model.TSubnet
			dst *model.TSubnet
			err error
		)
		// 如果是办公网，则不校验源地址类型
		if info.Src == conf.BanGongWang || info.Src == conf.BanGongWangV6 {
			src = &model.TSubnet{Region: "外网"}
		} else if src, err = subnet.GetIPNet(info.Src); err != nil {
			return err
		}
		if dst, err = subnet.GetIPNet(info.Dst); err != nil {
			return err
		}
		if info.Direction == "inside" {
			// 入向源IP必须是公网地址
			if src.Region != "外网" {
				return fmt.Errorf("入向访问源地址不是外网地址, 源地址: %s, 属地: %s", info.Src, src.Region)
			}
			// 入向访问目标必须是属地地址
			if dst.Region != h.region.Name {
				return fmt.Errorf("入向访问目标地址不是%s地址, 目标地址: %s, 属地: %s", h.region.Name, info.Dst, dst.Region)
			}

			if info.StaticIp != "" {
				if dst.NetType != "内网" {
					return fmt.Errorf("入向访问目标地址不是%s内网地址, 源地址: %s, 属地: %s, 类型: %s", h.region.Name, info.Dst, dst.Region, dst.NetType)
				}
				internal, err := subnet.GetIPNet(info.StaticIp)
				if err != nil {
					return err
				}
				if internal.Region != h.region.Name {
					return fmt.Errorf("入向访问内部地址不是%s地址, 内部地址: %s, 属地: %s", h.region.Name, info.StaticIp, internal.Region)
				}
			}
		} else {
			// 出向源IP必须是属地内网地址
			if src.Region != h.region.Name {
				return fmt.Errorf("出向访问源地址不是%s地址, 源地址: %s, 属地: %s", h.region.Name, info.Src, src.Region)
			}
			// 出向访问目标不能是内网地址
			if dst.NetType == "内网" {
				return fmt.Errorf("出向访问目标不能是内网地址, 目标地址: %s, 属地: %s, 类型: %s", info.Dst, dst.Region, dst.NetType)
			}
			// 出向访问不能是本属地地址
			if dst.Region == h.region.Name {
				return fmt.Errorf("出向访问目标不能是%s地址, 目标地址: %s", dst.Region, info.Dst)
			}
			if info.OutboundNetworkType != "" && dst.NetType != info.OutboundNetworkType {
				return fmt.Errorf("出向访问目标地址不是%s地址, 目标地址: %s, 类型: %s", info.OutboundNetworkType, info.Dst, dst.NetType)
			}
		}
	}
	return nil
}

// CheckInfoType 校验工单数据类型
func (h *taskHandler) checkNlbInfoType(data []*model.TTaskInfo) error {
	for _, info := range data {
		dst, err := subnet.GetIPNet(info.Dst)
		if err != nil {
			return err
		}
		// 出向访问目标必须是公网地址
		if dst.Region != h.region.Name && dst.NetType != "外部" {
			return fmt.Errorf("访问目标地址不是%s外部地址, 目标地址: %s, 属地: %s, 类型: %s", h.region.Name, info.Dst, dst.Region)
		}
		node, err := subnet.GetIPNet(info.Node)
		if err != nil {
			return err
		}
		if node.Region != h.region.Name && node.NetType != "内网" {
			return fmt.Errorf("访问node地址不是%s内网地址, node: %s, 属地: %s, 类型: %s", h.region.Name, info.Node, node.Region, node.NetType)
		}
	}
	return nil
}

func (h *taskHandler) isOperate(operate string) (bool, error) {
	taskStatus := model.TTaskStatus{}
	if e := taskStatus.FirstByOperate(operate); e != nil {
		return false, e
	}
	if !strings.Contains(taskStatus.TaskStatus, h.task.Status) {
		return false, fmt.Errorf("拒绝操作, 任务状态必须在%v里, 任务状态: %s", taskStatus.TaskStatus, h.task.Status)
	}
	if !strings.Contains(taskStatus.JiraStatus, h.task.JiraStatus) {
		return false, fmt.Errorf("拒绝操作, 工单状态必须在%v里, 工单状态: %s", taskStatus.TaskStatus, h.task.Status)
	}
	return true, nil
}

func (h *taskHandler) CanOperate(operate string) (bool, error) {
	if h.Err != nil {
		return false, h.Err
	}
	return h.isOperate(operate)
}

// Exec 执行工单，异步执行
func (h *taskHandler) Exec() error {
	if h.Err != nil {
		return h.Err
	}
	l := zap.L().With(
		zap.Int("TaskId", h.task.Id),
		zap.String("func", "exec"),
		zap.String("JiraKey", h.task.JiraKey),
	)
	l.Info("开始执行工单--->")
	h.addLog("开始执行工单--->")
	l.Debug("更新工单状态--->")
	if e := h.updateTaskExecuting(); e != nil {
		h.addLog(e.Error())
		l.Error(e.Error())
		return e
	}
	h.addLog("获取工单需要执行的策略信息--->")
	l.Info("1. 获取工单需要执行的策略信息--->")
	infos, e := new(model.TTaskInfo).FindExecInfoByTaskId(h.task.Id)
	if e != nil {
		l.Error(e.Error())
		h.addLog(e.Error())
		return e
	}
	h.addLog(fmt.Sprintf("当前需要执行%d条策略--->", len(infos)))
	l.Info("2. 将设备一致的策略信息合并到一起----------->", zap.Int("数量", len(infos)))
	h.addLog("组合相同设备策略信息--->")
	deviceInfoM := h.makeDeviceIdSameInfos(infos)
	h.addLog("推送策略--->")
	l.Info("3. 推送策略信息-------------------------->")
	go func() {
		var execErr error
		if h.task.Type == conf.TaskTypeFirewall {
			execErr = h.sendInfos(deviceInfoM)
		} else {
			execErr = h.sendF5Infos(deviceInfoM)
		}
		l.Info("4. 更新任务信息-------------------------->")
		h.addLog("更新任务状态--->")
		if execErr != nil {
			l.Error(execErr.Error())
			h.addLog(execErr.Error())
			if e := h.updateFailed(execErr.Error()); e != nil {
				l.Error(e.Error())
			}
			h.addLog("工单任务执行失败--->")
			return
		} else {
			if e := h.updateSuccess(); e != nil {
				l.Error(e.Error())
				return
			}
			l.Info("5. 更新jira流程------------------------->")
			h.addLog("更新jira流程--->")
			if e := h.UpdateJiraTransitionByOperate(conf.TaskOperateExec); e != nil {
				h.addLog(fmt.Sprintf("更新jira流程失败: %s", e.Error()))
			}
			h.addLog("工单任务执行成功--->")
		}
	}()
	return nil
}

// 执行完成更新设备策略
func (h *taskHandler) updateDevicePolicy(deviceInfos *map[int][]model.TTaskInfo) error {
	h.addLog("更新设备策略----->")
	for deviceId, _ := range *deviceInfos {
		d := model.TFirewallDevice{}
		if e := d.QueryById(deviceId); e != nil {
			return e
		}
		h.addLog(fmt.Sprintf("设备信息: <%d:%s-%s>", deviceId, d.Name, d.Host))
		if h.task.Type == conf.TaskTypeFirewall {
			parser, err := device2.NewDeviceHandler(deviceId)
			if err != nil {
				h.addLog(fmt.Sprintf("更新设备策略异常: error: <%s>", err.Error()))
				return err
			}
			parser.ParseConfig()
		} else {
			parser := device2.NewF5Policy(deviceId)
			if e := parser.ParseConfig(); e != nil {
				zap.L().Error(e.Error())
				h.addLog(fmt.Sprintf("更新设备策略失败, err: %s", e.Error()))
				return e
			}
		}
	}
	h.addLog(fmt.Sprintf("<-------设备策略更新成功-------->"))
	return nil
}

// 将设备ID一致的策略信息组装在一起
func (h *taskHandler) makeDeviceIdSameInfos(infos []*model.TTaskInfo) map[int][]*model.TTaskInfo {
	result := make(map[int][]*model.TTaskInfo)
	for _, i := range infos {
		result[i.DeviceId] = append(result[i.DeviceId], i)
	}
	return result
}

// 发送工单信息
func (h *taskHandler) sendInfos(deviceInfos map[int][]*model.TTaskInfo) error {
	for deviceId, infos := range deviceInfos {
		result, err := h.send(deviceId, infos)
		if err != nil {
			zap.L().Error("调用GRPC接口执行失败------------->", zap.Any("result", result), zap.Error(err))
			return err
		}
		for _, cmd := range result {
			if e := h.updateInfo(int(cmd.Id), cmd.Status, cmd.Result); e != nil {
				h.addLog(e.Error())
			}
		}
	}
	return nil
}

// 调用F5API推送F5配置
func (h *taskHandler) sendF5Infos(deviceInfos map[int][]*model.TTaskInfo) error {
	for deviceId, infos := range deviceInfos {
		parser := device2.NewF5Policy(deviceId)
		for _, info := range infos {
			if e := parser.SendConfig(info); e != nil {
				_ = h.updateInfo(info.Id, conf.TaskStatusFailed, e.Error())
				return fmt.Errorf("工单执行失败, infoId: %d, message: %w", info.Id, e)
			}
			_ = h.updateInfo(info.Id, conf.TaskStatusSuccess, "")
			time.Sleep(time.Second * 5)
		}
		parser.CloseGrpc()
	}
	return nil
}

func (h *taskHandler) updateReady() error {
	if e := h.task.UpdateStatus(conf.TaskStatusReady); e != nil {
		return e
	}
	return nil
}

// 执行任务信息
func (h *taskHandler) updateTaskExecuting() error {
	h.task.Status = conf.TaskStatusExecuting
	now := time.Now()
	h.task.ExecuteTime = &now
	if e := h.task.Save(); e != nil {
		return fmt.Errorf("更新任务状态失败, err: %s", e.Error())
	}
	return nil
}

// 更新任务失败
func (h *taskHandler) updateFailed(message string) error {
	h.task.Status = conf.TaskStatusFailed
	h.task.ErrorInfo = message
	now := time.Now()
	h.task.ExecuteEndTime = &now
	h.task.ExecuteUseTime = int(h.task.ExecuteEndTime.Sub(*h.task.ExecuteTime).Seconds())
	if e := h.task.Save(); e != nil {
		return e
	}
	return nil
}

// 更新任务成功
func (h *taskHandler) updateSuccess() error {
	h.task.Status = conf.TaskStatusSuccess
	now := time.Now()
	h.task.ExecuteEndTime = &now
	h.task.ExecuteUseTime = int(h.task.ExecuteEndTime.Sub(*h.task.ExecuteTime).Seconds())
	if e := h.task.Save(); e != nil {
		return e
	}
	return nil
}

// UpdateJiraTransitionByOperate 更新jira流程
func (h *taskHandler) UpdateJiraTransitionByOperate(operate string) error {
	if h.Err != nil {
		return h.Err
	}
	// 1. 根据当前任务状态获取下一流程状态
	// |  状态  	|  当前jira流程 	| 		下一流程				| 对应操作
	// - ready 	-  编写方案		- 执行方审批&送安全审批|执行风险一号位审批 	- 审核通过
	// - success  -  网络运维实施	- 验收中		- 	执行工单
	taskStatus := model.TTaskStatus{}
	if err := taskStatus.FirstByOperate(operate); err != nil {
		return err
	}
	// 2. 循环当前工单状态要下送的流程，以&分割，表示先送给A再送到B
	var err error
	for _, i := range strings.Split(taskStatus.JiraNextStatus, "&") {
		err = h.updateJiraTransitionByName(i)
	}
	if err != nil {
		return err
	}
	if taskStatus.Assignee != "" {
		return h.updateJiraAssignee(taskStatus.Assignee)
	}
	return nil
}

// AddJiraComment 添加评论
func (h *taskHandler) AddJiraComment(content string) error {
	jh := utils.NewJiraHandler()
	return jh.AddComment(h.task.JiraKey, content)
}

// 分配Jira经办人
func (h *taskHandler) updateJiraAssignee(operator string) error {
	jh := utils.NewJiraHandler()
	return jh.UpdateAssignee(h.task.JiraKey, operator)
}

// 根据流程名更新jira流程
func (h *taskHandler) updateJiraTransitionByName(name string) error {
	// 1. 实例化jira=
	jh := utils.NewJiraHandler()
	// 2. 获取工单信息
	issue, e := jh.GetIssue(h.task.JiraKey)
	if e != nil {
		return e
	}
	// 3. 若工单经办人不是网络自动化，则修改经办人分配给自己
	if issue.Fields.Assignee.Name != jh.GetUsername() {
		if e := jh.UpdateAssignee(h.task.JiraKey, jh.GetUsername()); e != nil {
			return e
		}
	}
	transitions, e := jh.GetTransition(h.task.JiraKey)
	if e != nil {
		return e
	}
	var nextId int
	// 循环工单可以往下送的状态，只要包含（比如定义的流程为 送安全审批|技术一号位审核  to.Name=送安全审批）则直接送到此流程
	for _, item := range transitions {
		if strings.Contains(name, item.To.Name) {
			nextId, _ = strconv.Atoi(item.Id)
			break
		}
	}
	if nextId == 0 {
		zap.L().Error(fmt.Sprintf("从当前工单可以分配的流程中未获取到<%s>流程", name))
		zap.L().Info(fmt.Sprintf("当前工单流程: <%+v>", transitions))
		return fmt.Errorf("从当前工单可以分配的流程<%s>中未获取到<%s>流程", transitions, name)
	}
	// 6. 更改工单流程状态
	if e := jh.UpdateTransition(h.task.JiraKey, nextId); e != nil {
		return e
	}
	if e := h.SyncJiraStatus(); e != nil {
		return e
	}
	return nil
}

// 调用grpc接口发送配置
func (h *taskHandler) send(deviceId int, infos []*model.TTaskInfo) (result []*net_api.Command, err error) {
	d := model.TFirewallDevice{}
	if e := d.FirstById(deviceId); e != nil {
		return nil, e
	}
	deviceType := model.TDeviceType{}
	if e := deviceType.FirstById(d.DeviceTypeId); e != nil {
		return nil, e
	}
	commands := make([]*net_api.Command, 0)
	for _, i := range infos {
		commands = append(commands, &net_api.Command{Id: int32(i.Id), Cmd: i.Command})
	}
	client := net_api2.NewClient(h.region.ApiServer)
	result, e := client.Config(&net_api.ConfigRequest{
		DeviceType:     deviceType.Name,
		Host:           d.Host,
		Username:       d.Username,
		Password:       d.Password,
		EnablePassword: d.EnablePassword,
		Port:           22,
		Commands:       commands,
	})
	if e != nil {
		return nil, e
	}
	return result, nil
}

// 更新策略信息
func (h *taskHandler) updateInfo(infoId int, status, result string) error {
	taskInfo := model.TTaskInfo{}
	taskInfo.Id = infoId
	return taskInfo.UpdateStatusAndResult(status, result)
}

// GeneFirewallConfig 生成防火墙策略配置
func (h *taskHandler) GeneFirewallConfig() error {
	if h.Err != nil {
		return h.Err
	}
	l := zap.L().With(
		zap.String("func", "GeneFirewallConfig"),
		zap.String("jira_key", h.task.JiraKey),
		zap.Int("task_id", h.task.Id),
	)
	l.Info("<------------------生成配置------------------>")
	l.Info("1. 获取工单要开通策略信息-------------------->")
	infos, e := new(model.TTaskInfo).FindByTaskId(h.task.Id)
	if e != nil {
		return e
	}
	infos, e = h.splitInfoList(infos)
	if e != nil {
		return e
	}
	l.Info("2. 获取策略关联设备信息---------------------->")
	if e := h.makeInfosDevice(infos); e != nil {
		return e
	}
	l.Info("3. 校验策略nat映射信息---------------------->")
	if e := h.checkInfosNat(infos); e != nil {
		return e
	}
	l.Info("4. 获取出向策略nat映射pool信息------------------>")
	if e := h.makeInfosNatPool(infos); e != nil {
		return e
	}
	// 获取工单信息策略，若存在策略，则获取策略并更新数据库，未开通策略的加入到列表，后面重新整合并生成命令
	l.Info("5. 循环单条策略信息，保存已开通策略命令，并返回未开通策略信息----------->")
	denyInfos, e := h.getInfoPolicyAndDenyInfos(infos)
	if e != nil {
		return e
	}
	l.Info("6. 生成并组装未开通的策略命令--------------------->")
	newInfos, e := h.saveDenyInfos(denyInfos)
	if e != nil {
		return e
	}
	if e := h.geneDenyConfig(newInfos); e != nil {
		return e
	}
	l.Info("<------------------生成配置结束------------------>")
	return nil
}

// GeneNlbConfig 生成负载均衡策略配置
func (h *taskHandler) GeneNlbConfig() error {
	if h.Err != nil {
		return h.Err
	}
	l := zap.L().With(zap.String("func", "GeneNlbConfig"), zap.Int("task_id", h.task.Id), zap.String("jira_key", h.task.JiraKey))
	l.Info("<------------------生成NLB配置------------------>")
	l.Info("1. 获取工单要开通策略信息-------------------->")
	infos, e := new(model.TTaskInfo).FindByTaskId(h.task.Id)
	if e != nil {
		return e
	}
	l.Info("2. 获取策略关联设备信息---------------------->")
	if e := h.makeInfosDevice(infos); e != nil {
		return e
	}
	l.Info("3. 循环单条策略信息，保存已开通策略命令，并返回未开通策略信息----------->")
	denyInfos, e := h.getInfoPolicyAndDenyInfos(infos)
	if e != nil {
		return e
	}
	l.Info("4. 生成并组装未开通的策略命令--------------------->")
	newInfos, e := h.saveDenyInfos(denyInfos)
	if e != nil {
		return e
	}
	if e := h.geneF5DenyConfig(newInfos); e != nil {
		return e
	}
	l.Info("5. 更新任务状态--------------------------->")
	if e := h.updateReady(); e != nil {
		return e
	}
	l.Info("<------------------生成配置结束------------------>")
	return nil
}

// 拆分策略列表
func (h *taskHandler) splitInfoList(infos []*model.TTaskInfo) ([]*model.TTaskInfo, error) {
	results := make([]*model.TTaskInfo, 0)
	for _, item := range infos {
		for _, src := range strings.Split(item.Src, ",") {
			for _, dst := range strings.Split(item.Dst, ",") {
				for _, port := range strings.Split(item.DPort, ",") {
					info := &model.TTaskInfo{
						TaskId:              item.TaskId,
						Src:                 src,
						Dst:                 dst,
						DPort:               port,
						Direction:           item.Direction,
						StaticIp:            item.StaticIp,
						StaticPort:          item.StaticPort,
						OutboundNetworkType: item.OutboundNetworkType,
						Protocol:            item.Protocol,
						Status:              item.Status,
					}
					if e := info.Create(); e != nil {
						return nil, e
					}
					results = append(results, info)
				}
			}
		}
		// 删除拆分前的工作项信息
		database.DB.Delete(item)
	}
	return results, nil
}

// 循环策略列表，获取策略的设备信息
func (h *taskHandler) makeInfosDevice(infos []*model.TTaskInfo) error {
	for _, info := range infos {
		if e := h.getInfoDevice(info); e != nil {
			return e
		}
	}
	return nil
}

// 获取策略设备信息
func (h *taskHandler) getInfoDevice(info *model.TTaskInfo) error {
	var (
		deviceId int
		err      error
	)
	if h.task.Type == conf.TaskTypeFirewall {
		deviceId, err = h.getInfoDeviceId(info)
	} else {
		deviceId, err = h.getInfoNlbDeviceId(info)
	}
	if err != nil {
		return err
	}
	if deviceId == 0 {
		zap.L().Warn("未获取到工单对应的网络设备", zap.Any("info", info), zap.Any("task", h.task))
		return fmt.Errorf("未获取到工单对应的设备, 请确认工单属地和策略对应的网段是否关联网络设备")
	}
	info.DeviceId = deviceId
	return nil
}

// 校验nat信息是否已映射
func (h *taskHandler) checkInfosNat(infos []*model.TTaskInfo) error {
	for _, info := range infos {
		parser, err := device2.NewDeviceHandler(info.DeviceId)
		if err != nil {
			return err
		}
		if err := parser.CheckNat(info); err != nil {
			return err
		}
	}
	return nil
}

// 循环策略列表，获取出向策略nat映射信息
func (h *taskHandler) makeInfosNatPool(infos []*model.TTaskInfo) error {
	for _, info := range infos {
		if e := h.getInfoNatPool(info); e != nil {
			return e
		}
	}
	return nil
}

// 获取出向策略nat映射信息
func (h *taskHandler) getInfoNatPool(info *model.TTaskInfo) error {
	// 入向策略或者是CN2的不需要获取
	if info.Direction == "inside" || !strings.Contains(info.OutboundNetworkType, "CN2") {
		return nil
	}

	nat, err := h.getDeviceNatByDeviceId(info.Dst, info.DeviceId, info.OutboundNetworkType)
	if err != nil {
		return err
	}
	info.PoolName = nat.StaticName
	info.NatName = nat.NatName
	return nil
}

// 根据目标地址和出向网络信息获取nat映射名称
func (h *taskHandler) getDeviceNatByDeviceId(dst string, deviceId int, outboundNetworkType string) (result *model.TDeviceNatAddress, err error) {
	d := model.TFirewallDevice{}
	if e := d.FirstById(deviceId); e != nil {
		return nil, e
	}
	natAddresses, e := new(model.TDeviceNatAddress).FindByDeviceIdOutboundNetworkType(deviceId, outboundNetworkType)
	if e != nil {
		return nil, e
	}
	minSubnet := ""
	for _, na := range natAddresses {
		for _, v := range strings.Split(na.Subnet, ",") {
			if r, _ := subnet.IsNet(v, dst); r {
				if minSubnet == "" {
					minSubnet = v
					result = na
					continue
				}
				// 最小优先，如果s比最小的还小，则替换
				if ok, _ := subnet.IsNet(minSubnet, v); ok {
					minSubnet = v
					result = na
				}
			}
		}
	}
	if result == nil {
		return nil, fmt.Errorf("未获取到nat映射，请检查策略类型和出向网络类型是否正确\n若配置正确，请在配置管理->设备配置->Nat映射地址菜单中添加对应的地址映射的nat\ndst: %s, 关联设备: %s, 出向网络类型: %s\n", dst, d.Name, outboundNetworkType)
	}
	return
}

// 获取已开通策略命令，并返回未开通策略信息
func (h *taskHandler) getInfoPolicyAndDenyInfos(infos []*model.TTaskInfo) ([]*model.TTaskInfo, error) {
	denyInfos := make([]*model.TTaskInfo, 0)
	for _, info := range infos {
		if h.task.Type == conf.TaskTypeFirewall {
			if e := h.saveInfoPolicy(info); e != nil {
				return nil, e
			}
		} else {
			if e := h.saveInfoNlbPolicy(info); e != nil {
				return nil, e
			}
		}
		if info.Action == "deny" { // 先获取已有配置的策略
			denyInfos = append(denyInfos, info)
		}
	}
	return denyInfos, nil
}

// 获取是否已有负载均衡配置
func (h *taskHandler) saveInfoNlbPolicy(info *model.TTaskInfo) error {
	parser := device2.NewF5Policy(info.DeviceId)
	vs, e := parser.Search(info)
	if e != nil {
		return e
	}
	if vs != nil {
		info.Action = "permit"
		info.Command = parser.GetCommand(vs)
		if err := database.DB.Save(info).Error; err != nil {
			zap.L().Error(fmt.Sprintf("保存工单策略信息异常: <%s>, info: <%+v>", err.Error(), *info))
			return fmt.Errorf("保存工单策略信息异常: <%s>", err.Error())
		}
	} else {
		info.Action = "deny"
	}
	return nil
}

// 获取策略信息是否开通
func (h *taskHandler) saveInfoPolicy(info *model.TTaskInfo) error {
	// 1. 先查询策略是否开通
	// 获取设备信息
	parser, err := device2.NewDeviceHandler(info.DeviceId)
	if err != nil {
		return err
	}
	dp, e := parser.Search(info)
	if e != nil {
		return e
	}
	if dp != nil {
		zap.L().Info(fmt.Sprintf("匹配到的策略: <%+v>", *dp))
		info.Action = dp.Action
		info.Command = parser.GetCommand(dp)
		if e := info.Save(); e != nil {
			return e
		}
	} else {
		info.Action = "deny"
	}
	return nil
}

// 保存拼装的未开通策略信息并删除单条的未开通策略信息
func (h *taskHandler) saveDenyInfos(denyInfos []*model.TTaskInfo) ([]*model.TTaskInfo, error) {
	if len(denyInfos) == 0 {
		return nil, nil
	}
	for _, v := range denyInfos {
		zap.L().Debug("拼接策略", zap.Any("denyInfo", v))
	}
	// 获取未开通的信息ID
	denyInfoIds := make([]int, 0)
	for _, info := range denyInfos {
		denyInfoIds = append(denyInfoIds, info.Id)
	}
	var newDenyInfos []*model.TTaskInfo
	if h.task.Type == conf.TaskTypeFirewall {
		newDenyInfos = h.makeNewDenyInfo(denyInfos)
	} else {
		newDenyInfos = h.makeNewNlbDenyInfo(denyInfos)
	}
	if err := database.DB.Create(&newDenyInfos).Error; err != nil {
		zap.L().Error(fmt.Sprintf("保存组装信息异常: <%s>", err.Error()))
		return nil, fmt.Errorf("保存组装信息异常，请先重试后找管理员处理！")
	}
	// 删除旧的deny数据
	database.DB.Delete(&model.TTaskInfo{}, denyInfoIds)
	return newDenyInfos, nil
}

// 生成新配置
func (h *taskHandler) geneDenyConfig(denyInfos []*model.TTaskInfo) error {
	for _, info := range denyInfos {
		parser, err := device2.NewDeviceHandler(info.DeviceId)
		if err != nil {
			return err
		}
		info.Action = "deny"
		command, e := parser.GeneCommand(h.task.JiraKey, info)
		if e != nil {
			return e
		}
		info.Command = command
		info.Status = conf.TaskStatusReady
		if e := info.Save(); e != nil {
			return e
		}
	}
	return nil
}

// 生成F5配置
func (h *taskHandler) geneF5DenyConfig(denyInfos []*model.TTaskInfo) error {
	for _, info := range denyInfos {
		parser := device2.NewF5Policy(info.DeviceId)
		if parser.Error() != nil {
			return parser.Error()
		}
		info.Action = "deny"
		if e := parser.GeneCommand(h.task.JiraKey, info); e != nil {
			return e
		}
		info.Status = conf.TaskStatusReady
		if e := info.Save(); e != nil {
			return e
		}
	}
	return nil
}

// 组装工作项信息，设备ID，源IP相等的目标IP组合在一起
func (h *taskHandler) makeNewDenyInfo(denyInfos []*model.TTaskInfo) (result []*model.TTaskInfo) {
	// 整合设备ID-目标IP-端口一致的源IP信息
	srcInfos := make(map[string]*model.TTaskInfo)
	for _, item := range denyInfos {
		k := fmt.Sprintf("%d-%s-%s-%s", item.DeviceId, item.Dst, item.Protocol, item.DPort)
		if item.StaticIp != "" {
			k = fmt.Sprintf("%s-%s-%s", k, item.StaticIp, item.StaticPort)
		}
		if item.PoolName != "" {
			k = fmt.Sprintf("%s-%s", k, item.PoolName)
		}
		if v, ok := srcInfos[k]; ok {
			// 因为顺序原因，会导致错乱而无法合并，所以需要每次进行重新排序
			srcL := strings.Split(v.Src, ",")
			srcL = append(srcL, item.Src)
			sort.Strings(srcL)
			v.Src = strings.Join(srcL, ",")
		} else {
			srcInfos[k] = item
		}
	}
	// 整合设备ID源IP和端口一致的目标IP信息
	dstInfos := make(map[string]*model.TTaskInfo)
	for _, item := range srcInfos {
		k := fmt.Sprintf("%d-%s-%s-%s", item.DeviceId, item.Src, item.Protocol, item.DPort)
		if item.StaticIp != "" {
			k = fmt.Sprintf("%s-%s-%s", k, item.StaticIp, item.StaticPort)
		}
		if item.PoolName != "" {
			k = fmt.Sprintf("%s-%s", k, item.PoolName)
		}
		if v, ok := dstInfos[k]; ok {
			if item.StaticIp != "" {
				continue
			}
			// 因为顺序原因，会导致错乱而无法合并，所以需要每次进行重新排序
			dstL := strings.Split(v.Dst, ",")
			dstL = append(dstL, item.Dst)
			sort.Strings(dstL)
			v.Dst = strings.Join(dstL, ",")
		} else {
			fmt.Println("不存在--->")
			dstInfos[k] = item
			fmt.Println(dstInfos)
		}
	}
	// 整合设备ID-源IP-目标地址相同的端口信息
	portInfos := make(map[string]*model.TTaskInfo)
	for _, item := range dstInfos {
		k := fmt.Sprintf("%d-%s-%s-%s", item.DeviceId, item.Src, item.Dst, item.Protocol)
		if item.StaticIp != "" {
			k = fmt.Sprintf("%s-%s-%s", k, item.StaticIp, item.StaticPort)
		}
		if item.PoolName != "" {
			k = fmt.Sprintf("%s-%s", k, item.PoolName)
		}
		if v, ok := portInfos[k]; ok {
			if item.StaticPort != "" {
				continue
			}
			v.DPort = fmt.Sprintf("%s,%s", v.DPort, item.DPort)
		} else {
			item.Id = 0
			portInfos[k] = item
		}
	}
	for _, info := range portInfos {
		result = append(result, info)
	}
	return
}

// 组装工作项信息，设备ID，源IP相等的目标IP组合在一起
func (h *taskHandler) makeNewNlbDenyInfo(denyInfos []*model.TTaskInfo) (result []*model.TTaskInfo) {
	var k string
	// 根据设备ID目标地址，目标端口，node_port整合地址
	dstInfos := make(map[string]*model.TTaskInfo)
	for _, item := range denyInfos {
		k = fmt.Sprintf("%d-%s-%s-%s-%s", item.DeviceId, item.Dst, item.DPort, item.NodePort, item.Protocol)
		if v, ok := dstInfos[k]; ok {
			v.Node = fmt.Sprintf("%s,%s", v.Node, (*item).Node)
		} else {
			dstInfos[k] = item
		}
	}
	for _, info := range dstInfos {
		result = append(result, info)
	}
	return
}

// 获取工单详情关联设备信息
func (h *taskHandler) getInfoDeviceId(data *model.TTaskInfo) (int, error) {
	// 获取工单实施类型
	subnetDevices, e := new(model.TFirewallSubnet).FindByRegionIdImplementTypeId(h.task.RegionId, h.implementType.Id)
	if e != nil {
		return 0, e
	}
	switch data.Direction {
	case "inside":
		return h.getDeviceIdByDeviceNets(data.Direction, data.Dst, subnetDevices), nil
	case "outside":
		return h.getDeviceIdByDeviceNets(data.Direction, data.Src, subnetDevices), nil
	}
	return 0, fmt.Errorf("未知的策略方向, direction: %s", data.Direction)
}

func (h *taskHandler) getDeviceIdByDeviceNets(direction, ip string, deviceNets []*model.TFirewallSubnet) (deviceId int) {
	minSubnet := ""
	for _, dn := range deviceNets {
		var sNet string
		if direction == "inside" {
			sNet = dn.OuterSubnet
		} else {
			sNet = dn.InnerSubnet
		}
		if result, _ := subnet.IsNet(sNet, ip); result {
			if minSubnet == "" {
				minSubnet = sNet
				deviceId = dn.DeviceId
				continue
			}
			// 最小优先，如果s比最小的还小，则替换
			if ok, _ := subnet.IsNet(minSubnet, sNet); ok {
				minSubnet = sNet
				deviceId = dn.DeviceId
			}
		}
	}
	return
}

// 获取Nlb工单详情关联设备信息
func (h *taskHandler) getInfoNlbDeviceId(data *model.TTaskInfo) (int, error) {
	// 获取工单实施类型
	subnets, e := new(model.TNLBSubnet).FindByRegionId(h.task.RegionId)
	if e != nil {
		return 0, e
	}
	return h.getNlbDeviceIdByNlbSubnets(data.Dst, subnets), nil
}
func (h *taskHandler) getNlbDeviceIdByNlbSubnets(ip string, deviceNets []*model.TNLBSubnet) (deviceId int) {
	minSubnet := ""
	for _, dn := range deviceNets {
		sNet := dn.Subnet
		if result, _ := subnet.IsNet(sNet, ip); result {
			if minSubnet == "" {
				minSubnet = sNet
				deviceId = dn.DeviceId
				continue
			}
			// 最小优先，如果s比最小的还小，则替换
			if ok, _ := subnet.IsNet(minSubnet, sNet); ok {
				minSubnet = sNet
				deviceId = dn.DeviceId
			}
		}
	}
	return
}

func (h *taskHandler) GetTaskInfos() error {
	if h.Err != nil {
		return h.Err
	}
	l := zap.L().With(zap.String("func", "GetTaskInfos"), zap.String("jira_key", h.task.JiraKey), zap.Int("task_id", h.task.Id))
	jh := utils.NewJiraHandler()
	issue, e := jh.GetIssue(h.task.JiraKey)
	if e != nil {
		return e
	}
	l.Debug("获取最新附件，以xlsx结尾并且ID最大的", zap.Any("attachment", issue.Fields.Attachment))
	attachmentId := getAttachmentId(issue.Fields.Attachment)
	if attachmentId == "0" {
		return fmt.Errorf("未获取到工单附件")
	}
	l.Info("3.2 获取工单附件信息 -->")
	attachmentResult, e := jh.GetAttachment(attachmentId)
	if e != nil {
		return e
	}
	l.Info(fmt.Sprintf("附件信息 ---> %+v", attachmentResult))
	attachmentByte, e := jh.ReadAttachment(attachmentResult.Content)
	if e != nil {
		return e
	}
	xlsx := utils.Xlsx{}
	xFile := xlsx.OpenBinary(attachmentByte)
	attachmentData := xlsx.ReadSheetWithIndex(xFile, 0)
	parseH := parse.NewParseHandler(h.region.Name)
	if parseH == nil {
		l.Warn("未找到对应的解析算法--!")
		return fmt.Errorf("未获取到对应工单属地和环境的解析方法")
	}
	l.Debug(fmt.Sprintf("附件内容: <%+v>", attachmentData))
	l.Info("3.3 解析工单信息 -->")
	// 清除旧的工单信息
	if e := new(model.TTaskInfo).DeleteByTaskId(h.task.Id); e != nil {
		return e
	}
	if h.task.Type == conf.TaskTypeFirewall {
		infos := parseH.Parse(attachmentData)
		for _, info := range infos {
			if e := h.AddInfo(info); e != nil {
				return e
			}
		}
	} else {
		infos := parseH.ParseNlb(attachmentData)
		for _, info := range infos {
			if e := h.addNlbInfo(info); e != nil {
				return e
			}
		}
	}
	return nil
}

// 添加操作日志
func (h *taskHandler) addLog(content string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	if h.operateLog != nil {
		h.operateLog.Content += fmt.Sprintf("[%s] [%s]\n", now, content)
		database.DB.Save(h.operateLog)
	} else {
		h.operateLog = &model.TTaskOperateLog{TaskId: h.task.Id, Operator: h.operator}
		h.operateLog.Content = fmt.Sprintf("[%s] [%s]\n", now, content)
		database.DB.Create(h.operateLog)
	}
}

// SyncJiraStatus 同步jira状态
func (h *taskHandler) SyncJiraStatus() error {
	if h.Err != nil {
		return h.Err
	}
	jh := utils.NewJiraHandler()
	issue, e := jh.GetIssue(h.task.JiraKey)
	if e != nil {
		return e
	}
	if h.task.JiraStatus == issue.Fields.Status.Name {
		return nil
	}
	if e := h.task.UpdateJiraStatus(issue.Fields.Status.Name); e != nil {
		return e
	}
	return nil
}

// SyncJiraRegionEnvironment 同步jira属地环境信息
func (h *taskHandler) SyncJiraRegionEnvironment() error {
	if h.Err != nil {
		return h.Err
	}
	jh := utils.NewJiraHandler()
	issue, e := jh.GetIssue(h.task.JiraKey)
	if e != nil {
		return e
	}
	if h.task.JiraRegion == issue.Fields.Region.Value && h.task.JiraEnvironment == issue.Fields.Environment.Value && h.task.ImplementType == issue.Fields.ImplementContent.Value {
		return nil
	}
	h.task.JiraRegion = issue.Fields.Region.Value
	h.task.JiraEnvironment = issue.Fields.Environment.Value
	h.task.ImplementType = issue.Fields.ImplementContent.Value
	region, err := getTaskRegion(h.task.JiraRegion, h.task.JiraEnvironment)
	if err != nil {
		return err
	}
	h.task.Type = getTaskType(issue)
	h.task.RegionId = region.Id
	if e := h.task.Save(); e != nil {
		return e
	}
	return nil
}
func (h *taskHandler) addInfoAttachmentToJira() error {
	infos, e := new(model.TTaskInfo).FindByTaskId(h.task.Id)
	if e != nil {
		return e
	}
	var (
		titles []map[string]string
		data   []map[string]interface{}
	)
	if h.task.Type == "firewall" {
		titles = []map[string]string{
			{"title": "方向", "key": "direction"},
			{"title": "源地址", "key": "src"},
			{"title": "目标地址", "key": "dst"},
			{"title": "目标端口", "key": "dport"},
			{"title": "协议", "key": "protocol"},
			{"title": "出向网络类型", "key": "outbound_network_type"},
			{"title": "内部地址", "key": "static_ip"},
			{"title": "内部端口", "key": "static_port"},
		}
	} else {
		titles = []map[string]string{
			{"title": "目标地址", "key": "dst"},
			{"title": "目标端口", "key": "dport"},
			{"title": "协议", "key": "protocol"},
			{"title": "Node", "key": "node"},
			{"title": "NodePort", "key": "node_port"},
			{"title": "Snat", "key": "snat"},
		}
	}
	// 把工单策略struct转为map
	b, err := json.Marshal(&infos)
	if err != nil {
		return fmt.Errorf("转换工单策略信息为byte异常, err: %w", err)
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return fmt.Errorf("转换工单策略byte为map异常, err: %w", err)
	}
	xlsx := utils.Xlsx{}
	buffer := xlsx.NewFileToBuffer(titles, data)
	jh := utils.NewJiraHandler()
	filename := fmt.Sprintf("netops-%s.xlsx", utils.LocalTimeToString())
	if e := jh.AddAttachment(h.task.JiraKey, filename, buffer.Bytes()); e != nil {
		return e
	}
	return nil
}

func getAttachmentId(attachmentList []utils.Attachment) string {
	if len(attachmentList) == 0 {
		return "0"
	}
	var attachmentId int
	for _, attachment := range attachmentList {
		if !strings.HasSuffix(attachment.FileName, ".xlsx") {
			continue
		}
		currentAttachmentId, _ := strconv.Atoi(attachment.Id)
		if currentAttachmentId > attachmentId {
			attachmentId = currentAttachmentId
		}
	}
	return strconv.Itoa(attachmentId)
}
