package nlb

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/device/nlbs", handler.List)
	e.GET("/device/nlb", handler.Get)
	e.POST("/device/nlb", handler.Create)
	e.PUT("/device/nlb", handler.Update)
	e.DELETE("/device/nlb", handler.Delete)
	e.POST("/device/nlb/policy", handler.Policy)
	e.GET("/device/nlb/policy_log", handler.PolicyLog)
}
