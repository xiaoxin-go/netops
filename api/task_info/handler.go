package task_info

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"netops/libs"
	"netops/model"
	task "netops/pkg/task"
)

type Handler struct {
	libs.Controller
}

var handler *Handler

func init() {
	handler = &Handler{}
	handler.NewInstance = func() libs.Instance {
		return new(model.TTaskInfo)
	}
	handler.NewResults = func() any {
		return &[]*model.TTaskInfo{}
	}
}

// Create 生成工单
func (h *Handler) Create(ctx *gin.Context) {
	data := &model.TTaskInfo{}
	err := ctx.ShouldBindJSON(data)
	if err != nil {
		libs.HttpParamsError(ctx, fmt.Sprintf("参数解析异常: <%s>", err.Error()))
		return
	}
	operator := ctx.GetString("Operator")
	if e := task.AddTaskInfo(data, operator); e != nil {
		ctx.JSON(http.StatusOK, libs.ServerError(e.Error()))
		return
	}
	ctx.JSON(http.StatusOK, libs.Success(data, "添加成功"))
}

type InfoParams struct {
	TaskId int `json:"task_id"`
}

// Delete 删除TaskInfo
func (h *Handler) Delete(ctx *gin.Context) {
	id, e := h.GetId(ctx)
	if e != nil {
		libs.HttpParamsError(ctx, e.Error())
	}
	info := model.TTaskInfo{}
	if e := info.FirstById(id); e != nil {
		libs.HttpServerError(ctx, e.Error())
		return
	}
	user := ctx.GetString("Operator")
	if e := task.DelTaskInfo(&info, user); e != nil {
		ctx.JSON(http.StatusOK, libs.ServerError(e.Error()))
		return
	}
	ctx.JSON(http.StatusOK, libs.Success(info, "删除成功"))
}
