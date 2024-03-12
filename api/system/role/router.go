package role

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/system/roles", handler.List)
	e.GET("/system/role", handler.Get)
	e.POST("/system/role", handler.Create)
	e.PUT("/system/role", handler.Update)
	e.DELETE("/system/role", handler.Delete)
	e.GET("/system/role/permission", handler.GetPermission)
	e.PUT("/system/role/permission", handler.UpdatePermission)
}
