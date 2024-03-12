package task_info

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/task/infos", handler.List)
	e.GET("/task/info", handler.Get)
	e.POST("/task/info", handler.Create)
	e.DELETE("/task/info", handler.Delete)
}
