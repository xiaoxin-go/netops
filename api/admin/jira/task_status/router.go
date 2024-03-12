package task_status

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/admin/task_statuses", handler.List)
	e.GET("/admin/task_status", handler.Get)
	e.POST("/admin/task_status", handler.Create)
	e.PUT("/admin/task_status", handler.Update)
	e.DELETE("/admin/task_status", handler.Delete)
}
