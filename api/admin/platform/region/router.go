package region

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/admin/regions", handler.List)
	e.GET("/admin/region", handler.Get)
	e.POST("/admin/region", handler.Create)
	e.PUT("/admin/region", handler.Update)
	e.DELETE("/admin/region", handler.Delete)
}
