package task_template

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/admin/task_templates", handler.List)
	e.GET("/admin/task_template", handler.Get)
	e.POST("/admin/task_template", handler.Create)
	e.PUT("/admin/task_template", handler.Update)
	e.DELETE("/admin/task_template", handler.Delete)
}
