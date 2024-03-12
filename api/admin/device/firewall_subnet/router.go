package firewall_subnet

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/admin/firewall_subnets", handler.List)
	e.GET("/admin/firewall_subnet", handler.Get)
	e.POST("/admin/firewall_subnet", handler.Create)
	e.PUT("/admin/firewall_subnet", handler.Update)
	e.DELETE("/admin/firewall_subnet", handler.Delete)
}
