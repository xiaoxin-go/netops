package routers

import (
	"github.com/gin-gonic/gin"
	"netops/api/admin/device/device_nat_pool"
	"netops/api/admin/device/device_type"
	"netops/api/admin/device/f5_snat_pool"
	"netops/api/admin/device/firewall_subnet"
	"netops/api/admin/device/nat_address"
	"netops/api/admin/device/nlb_subnet"
	"netops/api/admin/jira/implement_type"
	"netops/api/admin/jira/issue_type"
	"netops/api/admin/jira/task_status"
	"netops/api/admin/platform/api"
	"netops/api/admin/platform/menu"
	"netops/api/admin/platform/outbound_network_type"
	"netops/api/admin/platform/region"
	"netops/api/admin/platform/subnet"
	"netops/api/admin/platform/task_template"
	"netops/api/auth"
	"netops/api/device/backup"
	"netops/api/device/firewall"
	"netops/api/device/nlb"
	firewall2 "netops/api/policy/firewall"
	"netops/api/policy/firewall_nat"
	nlb2 "netops/api/policy/nlb"
	"netops/api/system/log"
	"netops/api/system/role"
	"netops/api/system/user"
	"netops/api/task"
	"netops/api/task_info"
	"netops/api/tools/invalid_policy_task"
	"netops/api/tools/public_whitelist"
)

type Option func(engine *gin.RouterGroup)

var options = make([]Option, 0)

func Include(opts ...Option) {
	options = append(options, opts...)
}

func IncludeRouter() {
	Include(auth.Routers)

	Include(user.Routers)
	Include(role.Routers)
	Include(menu.Routers)
	Include(log.Routers)

	Include(task.Routers)
	Include(task_info.Routers)

	Include(firewall.Routers)
	Include(nlb.Routers)
	Include(backup.Routers)

	Include(public_whitelist.Routers)
	Include(invalid_policy_task.Routers)

	Include(firewall2.Routers)
	Include(nlb2.Routers)
	Include(firewall_nat.Routers)

	Include(api.Routers)
	Include(subnet.Routers)
	Include(region.Routers)
	Include(outbound_network_type.Routers)
	Include(task_template.Routers)

	Include(task_status.Routers)
	Include(issue_type.Routers)
	Include(implement_type.Routers)

	Include(firewall_subnet.Routers)
	Include(nlb_subnet.Routers)
	Include(f5_snat_pool.Routers)
	Include(device_type.Routers)
	Include(device_nat_pool.Routers)
	Include(nat_address.Routers)
}

// Init 初始化
func Init(r *gin.RouterGroup) *gin.RouterGroup {
	IncludeRouter()
	for _, opt := range options {
		opt(r)
	}
	return r
}
