package issue_type

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/admin/issue_types", handler.List)
	e.GET("/admin/issue_type", handler.Get)
	e.POST("/admin/issue_type", handler.Create)
	e.PUT("/admin/issue_type", handler.Update)
	e.DELETE("/admin/issue_type", handler.Delete)
}
