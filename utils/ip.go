package utils

import (
	"errors"
	"fmt"
	"net"
	"netops/conf"
	"strconv"
	"strings"
)

// ParseIP 校验IP地址
func ParseIP(ip string) (result string, err error) {
	if !VerifyIP(ip) {
		err = fmt.Errorf("IP<%s>格式不正确", ip)
		return
	}
	if strings.Contains(ip, "/") {
		result = ip
	} else {
		result = AddMask(ip)
	}
	return
}
func VerifyIP(ip string) bool {
	if strings.Contains(ip, "/") {
		_, _, err := net.ParseCIDR(ip)
		if err != nil {
			return false
		}
	} else {
		if net.ParseIP(ip) == nil {
			return false
		}
	}
	return true
}

type RangePort struct {
	Start int
	End   int
}

func ParseRangePort(port string) (result RangePort, err error) {
	if strings.Contains(port, "-") {
		pL := strings.Split(port, "-")
		if len(pL) != 2 {
			err = errors.New("端口格式不正确")
			return
		}
		if result.Start, err = ParsePort(pL[0]); err != nil {
			return
		}
		if result.End, err = ParsePort(pL[1]); err != nil {
			return
		}
		if result.Start > result.End {
			err = errors.New("端口格式不正确")
			return
		}
	} else {
		p, err1 := ParsePort(port)
		if err1 != nil {
			err = err1
			return
		}
		result.Start = p
		result.End = p
	}
	return
}

func ParsePort(port string) (result int, err error) {
	result, err = strconv.Atoi(port)
	if err != nil {
		err = fmt.Errorf("端口类型不正确")
		return
	}
	if !VerifyPort(result) {
		err = fmt.Errorf("端口格式不正确")
		return
	}
	return
}
func VerifyPort(port int) bool {
	if port < 0 || port > 65535 {
		return false
	}
	return true
}

// AddMask 为ip地址添加掩码
func AddMask(ip string) (result string) {
	if GetIpType(ip) == "ipv4" {
		return addMaskIpv4(ip)
	}
	return addMaskIpv6(ip)
}
func addMaskIpv6(ip string) (result string) {
	result = ip + "/128"
	return
}
func addMaskIpv4(ip string) (result string) {
	if ip == "0.0.0.0" {
		result = "0.0.0.0/0"
		return
	}
	subList := strings.Split(ip, ".")
	switch {
	case subList[1] == "0" && subList[2] == "0" && subList[3] == "0":
		result = ip + "/8"
	case subList[2] == "0" && subList[3] == "0":
		result = ip + "/16"
	case subList[3] == "0":
		result = ip + "/24"
	default:
		result = ip + "/32"
	}
	return
}
func GetIpType(ip string) string {
	if strings.Contains(ip, ".") {
		return conf.IpTypeV4
	}
	if strings.Contains(ip, ":") {
		return conf.IpTypeV6
	}
	return ""
}

func IpMask(subnet string) (ip, mask string) {
	if GetIpType(subnet) == "ipv4" {
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
