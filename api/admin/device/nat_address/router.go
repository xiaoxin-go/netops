package nat_address

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/admin/nat_addresses", handler.List)
	e.GET("/admin/nat_address", handler.Get)
	e.POST("/admin/nat_address", handler.Create)
	e.PUT("/admin/nat_address", handler.Update)
	e.DELETE("/admin/nat_address", handler.Delete)
}
