package subnet

import (
	"fmt"
	"github.com/pkg/errors"
	"net"
	"netops/model"
	"netops/utils"
	"strings"
)

func GetIPNet(ip string) (result *model.TSubnet, err error) {
	return getIpNet(ip)
}
func getIpNet(ip string) (*model.TSubnet, error) {
	ipType := utils.GetIpType(ip)
	subnets, e := new(model.TSubnet).PluckSubnet()
	if e != nil {
		return nil, e
	}
	var minSubnet string
	for _, subnet := range subnets {
		subnet = strings.TrimSpace(subnet)
		subnetType := utils.GetIpType(subnet)
		// 判断同一种类型的地址
		if subnetType != ipType {
			continue
		}
		ok, _ := IsNet(subnet, ip)
		if ok {
			// 如果是IPV4直接取匹配到的第一个网段即可，IPV6需要再次循环取最小的，原因是IPV6无法根据大小进行排序
			if minSubnet == "" {
				minSubnet = subnet
				continue
			}
			// 和最小的对比，如果比最小的还小，则替换最小的IP
			if ok, _ := IsNet(minSubnet, subnet); ok {
				minSubnet = subnet
			}
		}
	}
	if minSubnet != "" {
		subnet := model.TSubnet{}
		if e := subnet.FirstBySubnet(minSubnet); e != nil {
			return nil, e
		}
		return &subnet, nil
	}
	return nil, fmt.Errorf("获取匹配到地址所属的网段, ip: %s", ip)
}
func IsNet(subnet, ip string) (result bool, err error) {
	return isNet(subnet, ip)
}
func isNet(subnet, ip string) (ok bool, err error) {
	if subnet == ip {
		ok = true
		return
	}
	_, subnetNet, err := net.ParseCIDR(subnet)
	if err != nil {
		err = fmt.Errorf("转换网段<%s>异常: <%s>", subnet, err.Error())
		return
	}
	var bitMask string
	switch utils.GetIpType(ip) {
	case "ipv4":
		bitMask = "/32"
	case "ipv6":
		bitMask = "/128"
	}

	switch {
	case !strings.Contains(ip, "/"):
		ok = subnetNet.Contains(net.ParseIP(ip))
	case strings.Contains(ip, bitMask):
		ok = subnetNet.Contains(net.ParseIP(strings.Split(ip, bitMask)[0]))
	default:
		_, ipNet, err1 := net.ParseCIDR(ip)
		if err1 != nil {
			err = fmt.Errorf("转换IP<%s>异常: <%s>", ip, err.Error())
			return
		}
		ipOnes, _ := ipNet.Mask.Size()
		subnetOnes, _ := subnetNet.Mask.Size()
		ok = subnetOnes <= ipOnes && subnetNet.Contains(ipNet.IP)
	}
	return
}

// InBlacklistWhitelist 校验IP是否在白名单中
func InBlacklistWhitelist(ip string, ipType string) (bool, error) {
	whitelists, e := new(model.TBlacklistWhitelist).PluckSubnetByIpType(ipType)
	if e != nil {
		return false, e
	}
	// 白名单为空时, 直接返回不在白名单中
	if len(whitelists) == 0 {
		return false, nil
	}

	for _, whitelist := range whitelists {
		ok, e := isNet(whitelist, ip)
		if e != nil {
			return false, errors.Wrap(e, fmt.Sprintf("校验地址<IP:%s>是否在白名单中失败", ip))
		}
		if ok {
			return true, nil
		}
	}

	return false, nil
}
