package policy

import (
	"fmt"
	"netops/database"
	"netops/libs"
	"netops/model"
	"netops/pkg/device"
	"netops/utils"
	"strconv"
)

func FirewallList(deviceId, page, size int, direction, src, dst, port string) (result []*model.TDevicePolicy, total int64, err error) {
	if deviceId == 0 {
		return
	}
	// 第一种查全量情况
	if src == "" && dst == "" && port == "" {
		policies := make([]*model.TDevicePolicy, 0)
		db := database.DB.Model(&model.TDevicePolicy{}).Where("device_id = ?", deviceId)
		if direction != "" {
			db = db.Where("direction = ?", direction)
		}
		if e := db.Count(&total).Scopes(libs.Pagination(page, size)).Find(&policies).Error; e != nil {
			err = fmt.Errorf("查询策略失败, device_id: %d, err: %w", deviceId, e)
			return
		}
		result = policies
		return
	}
	// 校验基本格式
	if ok, e := checkQuery(src, dst, port); !ok {
		err = e
		return
	}
	src, dst, _ = parseParams(src, dst, port)
	// 第二种，根据源目IP地址或者端口查询
	if src != "" || dst != "" {
		// 查询匹配条件的策略信息
		ap, e := device.NewDeviceHandler(deviceId)
		if e != nil {
			err = e
			return
		}
		policies, e := ap.SearchAll(src, dst, port)
		if ap.Error() != nil {
			err = ap.Error()
			return
		}
		start := (page - 1) * size
		end := page * size
		total = int64(len(policies))
		if end > len(policies) {
			end = len(policies)
		}
		result = policies[start:end]
		return
	}
	return
}
func parseParams(srcStr, dstStr, portStr string) (src, dst string, port int) {
	src, _ = utils.ParseIP(srcStr)
	dst, _ = utils.ParseIP(dstStr)
	port, _ = utils.ParsePort(portStr)
	return
}
func checkQuery(src, dst, portStr string) (result bool, err error) {
	if src != "" && !utils.VerifyIP(src) {
		err = fmt.Errorf("源IP<%s>格式不正确", src)
		return
	}
	if dst != "" && !utils.VerifyIP(dst) {
		err = fmt.Errorf("目标IP<%s>格式不正确", dst)
		return
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		err = fmt.Errorf("端口<%s>格式不正确", portStr)
	}
	if port < 0 || port > 65535 {
		err = fmt.Errorf("端口<%d>格式不正确", port)
		return
	}
	result = true
	return
}

func NlbList(deviceId, page, size int, vs, dst, pool, member string) (result []*model.TF5Vs, total int64, err error) {
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = 20
	}
	db := database.DB.Model(&model.TF5Vs{}).Where("device_id = ?", deviceId)
	if vs != "" {
		db = db.Where("name like ?", "%"+vs+"%")
	}
	if pool != "" {
		db = db.Where("pool like ?", "%"+pool+"%")
	}
	if dst != "" {
		db = db.Where("destination like ?", "%"+dst+"%")
	}
	// 如果member存在，则需要对member进行过滤
	if member != "" {
		pools := make([]string, 0)
		if e := database.DB.Model(&model.TF5PoolNode{}).Where("name like ?", "%"+member+"%").Pluck("pool_name", &pools).Error; e != nil {
			err = fmt.Errorf("获取pool member信息异常: <%s>", e.Error())
			return
		}
		db = db.Where("pool in ?", pools)
	}
	result = make([]*model.TF5Vs, 0)
	if e := db.Count(&total).Offset((page - 1) * size).Limit(size).Find(&result).Error; e != nil {
		err = fmt.Errorf("获取vs信息异常: <%s>", e.Error())
		return
	}
	return
}
