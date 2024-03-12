package invalid_policy_task

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"netops/libs"
	"netops/model"
	"netops/pkg/tools"
)

type Handler struct {
	libs.Controller
}

var handler *Handler

func init() {
	handler = &Handler{}
	handler.NewInstance = func() libs.Instance {
		return new(model.TInvalidPolicyTask)
	}
	handler.NewResults = func() any {
		return &[]*model.TInvalidPolicyTask{}
	}
}

func (h *Handler) Parse(request *gin.Context) {
	params := struct {
		TaskId int `json:"task_id"`
	}{}
	if err := request.ShouldBindJSON(&params); err != nil {
		libs.HttpParamsError(request, fmt.Sprintf("解析参数异常: <%s>", err.Error()))
		return
	}
	libs.AddLog(request, fmt.Sprintf("解析无效策略<%d>状态", params.TaskId))
	ph := tools.NewInvalidPolicyHandler(params.TaskId)
	if e := ph.Parse(); e != nil {
		libs.HttpServerError(request, e.Error())
		return
	}
	libs.HttpSuccess(request, nil, "操作成功")
}
