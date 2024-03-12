package public_whitelist

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/tools/public_whitelists", handler.List)
	e.GET("/tools/public_whitelist", handler.Get)
	e.POST("/tools/public_whitelist", handler.Create)
	e.PUT("/tools/public_whitelist", handler.Update)
	e.DELETE("/tools/public_whitelist", handler.Delete)
	e.POST("/tools/public_whitelist/parse", handler.Parse)
	e.POST("/tools/public_whitelist/parse_all", handler.ParseAll)
}
