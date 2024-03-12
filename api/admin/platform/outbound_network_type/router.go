package outbound_network_type

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/admin/outbound_network_types", handler.List)
	e.GET("/admin/outbound_network_type", handler.Get)
	e.POST("/admin/outbound_network_type", handler.Create)
	e.PUT("/admin/outbound_network_type", handler.Update)
	e.DELETE("/admin/outbound_network_type", handler.Delete)
}
