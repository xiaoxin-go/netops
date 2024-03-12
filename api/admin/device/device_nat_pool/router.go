package device_nat_pool

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/admin/device_nat_pools", handler.List)
	e.GET("/admin/device_nat_pool", handler.Get)
	e.POST("/admin/device_nat_pool", handler.Create)
	e.PUT("/admin/device_nat_pool", handler.Update)
	e.DELETE("/admin/device_nat_pool", handler.Delete)
}
