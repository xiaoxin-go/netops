package parse

import (
	"netops/model"
	"strings"
)

type Nanjing struct {
	Base
}

func (p *Nanjing) Parse(data [][]string) (result []*model.TTaskInfo) {
	result = make([]*model.TTaskInfo, 0)
	itemType := "in"
	for index, item := range data {
		if index < 2 || isEmpty(item) || strings.Contains(item[0], "访问源") {
			continue
		}
		if strings.Contains(item[0], "情景2") {
			itemType = "out"
			continue
		}
		if itemType == "in" {
			result = append(result, p.parseIn(item))
		} else {
			result = append(result, p.parseOut(item))
		}
	}
	return
}
func (p *Nanjing) parseIn(data []string) (result *model.TTaskInfo) {
	result = new(model.TTaskInfo)
	result.Direction = "inside"
	result.Src = strings.TrimSpace(data[0])
	result.Dst = strings.TrimSpace(data[1])
	result.DPort = strings.ReplaceAll(strings.TrimSpace(data[2]), "/", ",")
	if len(data) < 4 {
		result.Protocol = parseProtocol("")
	} else {
		result.Protocol = parseProtocol(data[3])
	}
	return
}

// 解析南京出向策略
func (p *Nanjing) parseOut(data []string) (result *model.TTaskInfo) {
	result = new(model.TTaskInfo)
	result.Direction = "outside"
	// 如果写了地址，就不用应用名获取地址了
	if strings.TrimSpace(data[1]) != "" {
		result.Src = strings.TrimSpace(data[1])
	}
	result.Dst = strings.TrimSpace(data[2])
	result.DPort = strings.ReplaceAll(strings.TrimSpace(data[3]), "/", ",")
	if len(data) < 5 {
		result.Protocol = parseProtocol("")
	} else {
		result.Protocol = parseProtocol(data[4])
	}
	if len(data) < 6 {
		result.OutboundNetworkType = parseMappedType("")
	} else {
		result.OutboundNetworkType = parseMappedType(data[5])
	}

	return
}
