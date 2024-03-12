package device_type

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/admin/device_types", handler.List)
	e.GET("/admin/device_type", handler.Get)
	e.POST("/admin/device_type", handler.Create)
	e.PUT("/admin/device_type", handler.Update)
	e.DELETE("/admin/device_type", handler.Delete)
}
