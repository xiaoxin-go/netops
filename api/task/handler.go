package task

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"net/http"
	"netops/database"
	"netops/libs"
	"netops/model"
	"netops/pkg/task"
	"strings"
)

type Handler struct {
	libs.Controller
}

var handler *Handler

func init() {
	handler = &Handler{}
	handler.NewInstance = func() libs.Instance {
		return new(model.TTask)
	}
	handler.NewResults = func() any {
		return &[]*model.TTask{}
	}
}

func (h *Handler) List(ctx *gin.Context) {
	// 根据IP地址过滤
	ip := ctx.Query("ip")
	taskIds := make([]int, 0)
	if strings.TrimSpace(ip) != "" {
		ip = "%" + ip + "%"
		if err := database.DB.Model(&model.TTaskInfo{}).Where("src like ? or dst like ?", ip, ip).Pluck("task_id", &taskIds).Error; err != nil {
			libs.HttpServerError(ctx, "查询工单信息失败, ip: %s, err: %s", ip, err.Error())
			return
		}
		// 如果根据IP地址搜索，则要取出所有包含此工作项的工单
		h.QueryFilter = func(db *gorm.DB) *gorm.DB {
			return db.Where("id in ?", taskIds)
		}
	}
	h.Controller.List(ctx)
}

// Create 生成工单
func (h *Handler) Create(ctx *gin.Context) {
	// 1. 获取前端传入的工单号
	l := zap.L().With(zap.String("func", "Create"))
	l.Info("创建工单任务--->")
	t := model.TTask{}
	if err := ctx.ShouldBind(&t); err != nil {
		l.Error("参数解析失败", zap.Error(err))
		libs.HttpParamsError(ctx, "参数解析失败, err: %w", err)
		return
	}
	l.Info("任务信息", zap.Any("task", t))
	t.JiraKey = strings.TrimSpace(t.JiraKey)
	th := task.NewTaskHandler()
	if e := th.AddTask(&t); e != nil {
		l.Error("任务创建失败", zap.Error(e))
		libs.HttpServerError(ctx, e.Error())
		return
	}
	libs.AddLog(ctx, fmt.Sprintf("添加工单, 工单号: %s", t.JiraKey))
	libs.HttpSuccess(ctx, t, "添加成功")
}

type OperateParams struct {
	TaskId int `json:"task_id"`
}

// GetJiraAttachment 获取工单附件
func (h *Handler) GetJiraAttachment(ctx *gin.Context) {
	params := new(OperateParams)
	err := ctx.ShouldBindJSON(params)
	if err != nil {
		libs.HttpParamsError(ctx, fmt.Sprintf("参数异常: <%s>", err.Error()))
		return
	}
	user := ctx.GetString("Operator")
	if e := task.GetJiraAttachment(params.TaskId, user); e != nil {
		libs.HttpServerError(ctx, e.Error())
		return
	}
	libs.HttpSuccess(ctx, nil, "获取成功")
}

// GetOperateLog 获取操作日志
func (h *Handler) GetOperateLog(ctx *gin.Context) {
	taskId, e := h.GetId(ctx)
	if e != nil {
		libs.HttpParamsError(ctx, e.Error())
		return
	}
	log := model.TTaskOperateLog{}
	if e := log.LastByTaskId(taskId); e != nil {
		libs.HttpServerError(ctx, e.Error())
	}
	libs.HttpSuccess(ctx, log, "ok")
}

// GeneConfig 生成配置
func (h *Handler) GeneConfig(ctx *gin.Context) {
	params := new(OperateParams)
	err := ctx.ShouldBindJSON(params)
	if err != nil {
		libs.HttpParamsError(ctx, fmt.Sprintf("参数解析异常: <%s>", err.Error()))
		return
	}
	user := ctx.GetString("Operator")
	if e := task.GeneConfig(params.TaskId, user); e != nil {
		libs.HttpServerError(ctx, e.Error())
		return
	}
	ctx.JSON(http.StatusOK, libs.Success(nil, "恭喜亲，配置生成成功！"))
}

// Exec 执行工单
func (h *Handler) Exec(ctx *gin.Context) {
	operator := ctx.GetString("Operator")
	params := new(OperateParams)
	err := ctx.ShouldBindJSON(params)
	if err != nil {
		libs.HttpParamsError(ctx, fmt.Sprintf("参数解析异常: <%s>", err.Error()))
		return
	}
	if e := task.ExecTask(params.TaskId, operator); e != nil {
		libs.HttpServerError(ctx, e.Error())
		return
	}
	libs.HttpSuccess(ctx, nil, "亲，网络策略执行中...")
}

// VerifyPass 更新jira状态，审核通过，把工单状态修改为review
func (h *Handler) VerifyPass(ctx *gin.Context) {
	params := new(OperateParams)
	err := ctx.ShouldBindJSON(params)
	if err != nil {
		libs.HttpParamsError(ctx, fmt.Sprintf("参数解析异常: <%s>", err.Error()))
		return
	}
	user := ctx.GetString("Operator")
	if e := task.VerifyPass(params.TaskId, user); e != nil {
		libs.HttpServerError(ctx, e.Error())
		return
	}
	libs.HttpSuccess(ctx, nil, "操作成功")
}

// ToExecutor 更新jira状态，送执行方审批，把工单状态修改为review
func (h *Handler) ToExecutor(ctx *gin.Context) {
	params := new(OperateParams)
	err := ctx.ShouldBindJSON(params)
	if err != nil {
		libs.HttpParamsError(ctx, fmt.Sprintf("参数解析异常: <%s>", err.Error()))
		return
	}
	user := ctx.GetString("Operator")
	if e := task.ToExecutor(params.TaskId, user); e != nil {
		libs.HttpServerError(ctx, e.Error())
		return
	}
	libs.HttpSuccess(ctx, nil, "操作成功")
}

// Reject 驳回工单 更新jira状态，驳回工单，把工单状态修改为init
func (h *Handler) Reject(ctx *gin.Context) {
	params := struct {
		TaskId  int    `json:"task_id"`
		Content string `json:"content"`
	}{}
	err := ctx.ShouldBindJSON(&params)
	if err != nil {
		libs.HttpParamsError(ctx, fmt.Sprintf("参数解析异常: <%s>", err.Error()))
		return
	}
	user := ctx.GetString("Operator")
	if e := task.Reject(params.TaskId, params.Content, user); e != nil {
		libs.HttpServerError(ctx, e.Error())
		return
	}
	libs.HttpSuccess(ctx, nil, "操作成功")
}

// SyncJira 同步jira状态
func (h *Handler) SyncJira(ctx *gin.Context) {
	params := new(OperateParams)
	err := ctx.ShouldBindJSON(params)
	if err != nil {
		libs.HttpParamsError(ctx, fmt.Sprintf("参数解析异常: <%s>", err.Error()))
		return
	}
	user := ctx.GetString("Operator")
	if e := task.SyncJira(params.TaskId, user); e != nil {
		libs.HttpServerError(ctx, e.Error())
		return
	}
	libs.HttpSuccess(ctx, nil, "同步成功")
}

// ToLeader 送leader审核
func (h *Handler) ToLeader(ctx *gin.Context) {
	params := new(OperateParams)
	err := ctx.ShouldBindJSON(params)
	if err != nil {
		libs.HttpParamsError(ctx, fmt.Sprintf("参数解析异常: <%s>", err.Error()))
		return
	}
	user := ctx.GetString("Operator")
	if e := task.ToLeader(params.TaskId, user); e != nil {
		libs.HttpServerError(ctx, e.Error())
		return
	}
	libs.HttpSuccess(ctx, nil, "流程改变成功")
}
