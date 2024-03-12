package firewall_nat

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/policy/firewall_nat", handler.List)
}
