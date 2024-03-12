package user

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/system/users", handler.List)
	e.GET("/system/user", handler.Get)
	e.POST("/system/user", handler.Create)
	e.PUT("/system/user", handler.Update)
	e.DELETE("/system/user", handler.Delete)
}
