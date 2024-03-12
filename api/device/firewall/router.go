package firewall

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/device/firewalls", handler.List)
	e.GET("/device/firewall", handler.Get)
	e.POST("/device/firewall", handler.Create)
	e.PUT("/device/firewall", handler.Update)
	e.DELETE("/device/firewall", handler.Delete)
	e.POST("/device/firewall/policy", handler.Policy)
	e.GET("/device/firewall/policy_log", handler.PolicyLog)
	e.POST("/device/firewall/backup", handler.Backup)
}
