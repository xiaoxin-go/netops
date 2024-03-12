package f5_snat_pool

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/admin/f5_snat_pools", handler.List)
	e.GET("/admin/f5_snat_pool", handler.Get)
	e.POST("/admin/f5_snat_pool", handler.Create)
	e.PUT("/admin/f5_snat_pool", handler.Update)
	e.DELETE("/admin/f5_snat_pool", handler.Delete)
}
