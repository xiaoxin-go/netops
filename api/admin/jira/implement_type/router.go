package implement_type

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/admin/implement_types", handler.List)
	e.GET("/admin/implement_type", handler.Get)
	e.POST("/admin/implement_type", handler.Create)
	e.PUT("/admin/implement_type", handler.Update)
	e.DELETE("/admin/implement_type", handler.Delete)
}
