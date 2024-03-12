package task

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/tasks", handler.List)
	e.GET("/task", handler.Get)
	e.POST("/task", handler.Create)
	e.PUT("/task", handler.Update)
	e.DELETE("/task", handler.Delete)
	e.POST("/task/get_jira_attachment", handler.GetJiraAttachment)
	e.GET("/task/operate_log", handler.GetOperateLog)
	e.POST("/task/gene_config", handler.GeneConfig)
	e.POST("/task/exec", handler.Exec)
	e.POST("/task/verify_pass", handler.VerifyPass)
	e.POST("/task/to_executor", handler.ToExecutor)
	e.POST("/task/reject", handler.Reject)
	e.POST("/task/sync_jira", handler.SyncJira)
	e.POST("/task/to_leader", handler.ToLeader)
}
