package subnet

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/admin/subnets", handler.List)
	e.GET("/admin/subnet", handler.Get)
	e.POST("/admin/subnet", handler.Create)
	e.PUT("/admin/subnet", handler.Update)
	e.DELETE("/admin/subnet", handler.Delete)
}
