package backup

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/device/backups", handler.List)
	e.GET("/device/backup/download", handler.Download)
}
