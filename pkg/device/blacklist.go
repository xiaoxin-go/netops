package device

import (
	"fmt"
	"netops/conf"
	"netops/utils"
	"strings"
)

// GeneShowCmd 生成查询命令
func (a *AsaHandler) GeneShowCmd(groupName, subnet string) string {
	addr, _ := utils.IpMask(subnet)
	return fmt.Sprintf("show object-group id %s | in %s", groupName, addr)
}
func (a *AsaHandler) GenePermitCmd(groupNames []string, subnet string) string {
	addr, _ := utils.IpMask(subnet)
	commands := make([]string, 0)
	for _, groupName := range groupNames {
		commands = append(commands, fmt.Sprintf("object-group network %s", groupName))
		commands = append(commands, fmt.Sprintf("no network-object host %s", addr))
	}
	return strings.Join(commands, "\n")
}
func (a *AsaHandler) GeneDenyCmd(groupName, subnet string) string {
	addr, _ := utils.IpMask(subnet)
	commands := []string{
		fmt.Sprintf("object-group network %s", groupName),
		fmt.Sprintf("network-object host %s", addr),
	}
	return strings.Join(commands, "\n")
}

func (a *AsaHandler) GeneCreateGroupCmd(ipType, policyName, groupName string) (result string) {
	return ""
}

func (a *AsaHandler) ParseBlacklistGroupAddress() {

}

func (s *SrxHandler) getSecurityZone() string {
	return s.device.OutPolicy
}

func (s *SrxHandler) GeneShowCmd(groupName, subnet string) string {
	addr, _ := utils.IpMask(subnet)
	// show configuration | display set|match SH_blacklist_office_2023_72 | match 185.225.75.247
	return fmt.Sprintf("show configuration | display set|match %s | match %s", groupName, addr)
}

func (s *SrxHandler) GenePermitCmd(groupNames []string, subnet string) string {
	zone := s.getSecurityZone()
	addrName := s.geneBlackAddrName(subnet)
	commands := make([]string, 0)
	for _, groupName := range groupNames {
		commands = append(commands, fmt.Sprintf("delete security zones security-zone %s address-book address-set %s address %s\n", zone, groupName, addrName))
		commands = append(commands, fmt.Sprintf("delete security zones security-zone %s address-book address %s %s", zone, addrName, subnet))
	}
	return strings.Join(commands, "\n")
}
func (s *SrxHandler) GeneDenyCmd(groupName, subnet string) (result string) {
	zone := s.getSecurityZone() // 策略方向，untrust
	addrName := s.geneBlackAddrName(subnet)
	commands := []string{
		fmt.Sprintf("set security zones security-zone %s address-book address %s %s\n", zone, addrName, subnet),
		fmt.Sprintf("set security zones security-zone %s address-book address-set %s address %s", zone, groupName, addrName),
	}
	return strings.Join(commands, "\n")
}
func (s *SrxHandler) GeneCreateGroupCmd(ipType, policyName, groupName string) (result string) {
	zone := s.getSecurityZone()
	result += fmt.Sprintf("set security zones security-zone %s address-book address-set %s address 1.1.1.1\n", zone, groupName)
	result += fmt.Sprintf("set security policies from-zone %s to-zone %s policy %s match source-address %s", zone, s.device.InPolicy, policyName, groupName)
	return
}

func (h *HuaWeiHandler) geneIpObjectName(ip, mask string) string {
	return fmt.Sprintf("blacklist-%s-%s", ip, mask)
}

func (h *HuaWeiHandler) GeneShowCmd(groupName, subnet string) string {
	addr, _ := utils.IpMask(subnet)
	return fmt.Sprintf("display current-configuration | in %s", addr)
}

// GenePermitCmd 华为解封， 先从地址组内删除，再删除地址名
func (h *HuaWeiHandler) GenePermitCmd(groupNames []string, subnet string) string {
	addr, mask := utils.IpMask(subnet)
	ipObject := h.geneIpObjectName(addr, mask)
	commands := make([]string, 0)
	for _, groupName := range groupNames {
		commands = append(commands, fmt.Sprintf("ip address-set %s type group", groupName)) // 进入地址组
		commands = append(commands, fmt.Sprintf("undo address address-set %s", ipObject))   // 从地址组内删除
	}
	commands = append(commands, "quit")
	commands = append(commands, fmt.Sprintf("undo address-set %s", ipObject)) // 删除地址名
	return strings.Join(commands, "\n")
}

// GeneDenyCmd 华为设备封堵，先创建地址名，再加入地址组
func (h *HuaWeiHandler) GeneDenyCmd(groupName, subnet string) string {
	addr, mask := utils.IpMask(subnet)
	ipObject := h.geneIpObjectName(addr, mask)
	commands := []string{
		fmt.Sprintf("ip address-set %s type object", ipObject),
		fmt.Sprintf("address %s mask %s", addr, mask),
		fmt.Sprintf("ip address-set %s type group", groupName),
		fmt.Sprintf("address address-set %s", ipObject),
	}
	return strings.Join(commands, "\n")
}
func (h *HuaWeiHandler) GeneCreateGroupCmd(ipType, policyName, groupName string) string {
	commands := []string{
		fmt.Sprintf("ip address-set %s type group", groupName),
		"security-policy",
		fmt.Sprintf("rule name %s", policyName),
		fmt.Sprintf("source-address address-set %s", groupName),
	}
	return strings.Join(commands, "\n")
}
func (h *H3cHandler) GeneShowCmd(groupName, subnet string) string {
	addr, _ := utils.IpMask(subnet)
	return fmt.Sprintf("display current-configuration | in %s", addr)
}
func (h *H3cHandler) GenePermitCmd(groupNames []string, ip string) string {
	addr, mask := utils.IpMask(ip)
	ipType := utils.GetIpType(ip)
	if ipType == conf.IpTypeV4 {
		ipType = "ip"
	}
	commands := make([]string, 0)
	for _, groupName := range groupNames {
		commands = append(commands, fmt.Sprintf("object-group %s address %s", ipType, groupName))
		commands = append(commands, fmt.Sprintf("undo network subnet %s %s", addr, mask))
	}
	return strings.Join(commands, "\n")
}
func (h *H3cHandler) GeneDenyCmd(groupName, subnet string) string {
	addr, mask := utils.IpMask(subnet)
	ipType := utils.GetIpType(subnet)
	if ipType == conf.IpTypeV4 {
		ipType = "ip"
	}
	commands := []string{
		fmt.Sprintf("object-group %s address %s", ipType, groupName),
		fmt.Sprintf("network subnet %s %s", addr, mask),
	}
	return strings.Join(commands, "\n")
}
func (h *H3cHandler) GeneCreateGroupCmd(ipType, policyName, groupName string) (result string) {
	if ipType == conf.IpTypeV4 {
		ipType = "ip"
	}
	realPolicyName := policyName
	if ipType == conf.IpTypeV6 {
		realPolicyName = fmt.Sprintf("%s_%s", "Ipv6", policyName)
	}
	inPolicyChars := strings.Split(h.device.InPolicy, " ") // source-zone Untrust destination-zone Trust
	commands := []string{
		fmt.Sprintf("security-policy %s", ipType),
		fmt.Sprintf("rule name %s", realPolicyName),
		fmt.Sprintf("source-zone %s", inPolicyChars[1]),
		fmt.Sprintf("destination-zone %s", inPolicyChars[3]),
		fmt.Sprintf("source-ip %s %s", ipType, groupName),
	}
	return strings.Join(commands, "\n")
}
