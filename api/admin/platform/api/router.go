package api

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/admin/apis", handler.List)
	e.GET("/admin/api", handler.Get)
	e.POST("/admin/api", handler.Create)
	e.PUT("/admin/api", handler.Update)
	e.DELETE("/admin/api", handler.Delete)
}
