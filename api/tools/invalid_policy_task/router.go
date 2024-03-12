package invalid_policy_task

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/tools/invalid_policy_tasks", handler.List)
	e.GET("tools/invalid_policy_task", handler.Get)
	e.POST("tools/invalid_policy_task", handler.Create)
	e.PUT("tools/invalid_policy_task", handler.Update)
	e.DELETE("tools/invalid_policy_task", handler.Delete)
	e.POST("/tools/invalid_policy_task/parse", handler.Parse)
	e.GET("/tools/invalid_policy_task/policy_hit_counts", policyHitCountHandler.List)
}
