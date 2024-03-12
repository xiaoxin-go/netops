package nlb_subnet

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/admin/nlb_subnets", handler.List)
	e.GET("/admin/nlb_subnet", handler.Get)
	e.POST("/admin/nlb_subnet", handler.Create)
	e.PUT("/admin/nlb_subnet", handler.Update)
	e.DELETE("/admin/nlb_subnet", handler.Delete)
}
