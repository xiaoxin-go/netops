package parse

import (
	"go.uber.org/zap"
	"netops/model"
	"strings"
)

func NewParseHandler(region string) (result Parse) {
	switch region {
	case "上海沙箱":
		result = &ShangHai{Base{region: region}}
	case "上海":
		result = &Nanjing{Base{region: region}}
	case "南京":
		result = &Nanjing{Base{region: region}}
	case "贵州":
		result = &Guizhou{Nanjing{Base{region: region}}}
	case "芜湖":
		result = &Nanjing{Base{region: region}}
	}
	return
}

type Parse interface {
	Parse(data [][]string) []*model.TTaskInfo
	ParseNlb(data [][]string) []*model.TTaskInfo
}

type Base struct {
	region string
	Err    error
}

func (p *Base) Parse(data [][]string) []*model.TTaskInfo {
	l := zap.L().With(zap.String("func", "Parse"))
	l.Debug("解析工单配置")
	result := make([]*model.TTaskInfo, 0)
	itemType := "in"
	for index, rows := range data {
		l.Debug("数据信息", zap.Int("index", index), zap.Any("rows", rows))
		if index < 2 || isEmpty(rows) || strings.Contains(rows[0], "访问源") {
			l.Debug("排除空行")
			continue
		}
		if strings.Contains(rows[0], "情景2") {
			itemType = "out"
			continue
		}
		if itemType == "in" {
			result = append(result, p.parseIn(rows))
		} else {
			result = append(result, p.parseOut(rows))
		}
	}
	return result
}
func (p *Base) ParseNlb(data [][]string) []*model.TTaskInfo {
	result := make([]*model.TTaskInfo, 0)
	for index, rows := range data {
		if index < 1 || isEmpty(rows) || strings.Contains(rows[0], "源地址") {
			continue
		}
		item := &model.TTaskInfo{}
		item.Src = strings.TrimSpace(rows[0])
		item.Dst = strings.TrimSpace(rows[1])
		item.Protocol = parseProtocol(rows[2])
		item.DPort = strings.ReplaceAll(strings.TrimSpace(rows[3]), "/", ",")
		item.Node = strings.TrimSpace(rows[4])
		item.NodePort = strings.TrimSpace(rows[5])
		if len(rows) > 6 {
			item.SNat = strings.ToLower(strings.TrimSpace(rows[6]))
		}
		result = append(result, item)
	}
	return result
}
func (p *Base) parseIn(data []string) (result *model.TTaskInfo) {
	result = new(model.TTaskInfo)
	result.Direction = "inside"
	result.Src = strings.TrimSpace(data[0])
	result.StaticIp = strings.TrimSpace(data[1])
	result.StaticPort = strings.TrimSpace(data[2])
	result.Protocol = parseProtocol(data[3])
	result.Dst = strings.TrimSpace(data[4])
	result.DPort = strings.TrimSpace(data[5])
	return
}
func (p *Base) parseOut(data []string) (result *model.TTaskInfo) {
	result = new(model.TTaskInfo)
	result.Direction = "outside"
	result.Src = strings.TrimSpace(data[0])
	result.Dst = strings.TrimSpace(data[1])
	result.DPort = strings.ReplaceAll(strings.TrimSpace(data[2]), "/", ",")
	result.Protocol = parseProtocol(data[3])
	result.OutboundNetworkType = parseMappedType(data[4])
	return
}

func parseProtocol(protocol string) (result string) {
	result = strings.TrimSpace(protocol)
	if result == "" {
		result = "tcp"
	}
	result = strings.ToLower(result)
	return
}
func parseMappedType(mappedType string) (result string) {
	result = strings.ToUpper(strings.TrimSpace(mappedType))
	if result == "" {
		result = "公网"
	}
	return
}

func isEmpty(rows []string) (result bool) {
	// 如果长度小于4，或者前两个都为空，则认为是空行
	if len(rows) < 3 || (strings.TrimSpace(rows[0]) == "" && strings.TrimSpace(rows[1]) == "") {
		return true
	}
	for _, item := range rows {
		if strings.TrimSpace(item) != "" {
			return
		}
	}
	return true
}
