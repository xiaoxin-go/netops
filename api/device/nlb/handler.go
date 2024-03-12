package nlb

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"netops/libs"
	"netops/model"
	"netops/pkg/device"
)

type Handler struct {
	libs.Controller
}

var handler *Handler

func init() {
	handler = &Handler{}
	handler.NewInstance = func() libs.Instance {
		return new(model.TNLBDevice)
	}
	handler.NewResults = func() any {
		return &[]*model.TNLBDevice{}
	}
}

func (h *Handler) Policy(ctx *gin.Context) {
	params := &struct {
		DeviceId int `json:"device_id"`
	}{}
	zap.L().Info("<---------------更新设备策略--------------->")
	err := ctx.ShouldBindJSON(params)
	if err != nil {
		zap.L().Error(fmt.Sprintf("接收参数异常: <%s>", err.Error()))
		libs.HttpParamsError(ctx, fmt.Sprintf("接收参数异常: <%s>", err.Error()))
		return
	}
	zap.L().Info(fmt.Sprintf("deviceId: <%d>", params.DeviceId))
	parser := device.NewF5Policy(params.DeviceId)
	if e := parser.ParseConfig(); e != nil {
		libs.HttpServerError(ctx, e.Error())
	}
	libs.HttpSuccess(ctx, nil, "策略解析中...")
	return
}

func (h *Handler) PolicyLog(ctx *gin.Context) {
	id, e := h.GetId(ctx)
	if e != nil {
		libs.HttpParamsError(ctx, e.Error())
		return
	}
	result := model.TPolicyLog{}
	if e := result.LastByDeviceIdAndType(id, "nlb"); e != nil {
		libs.HttpServerError(ctx, e.Error())
		return
	}
	libs.HttpSuccess(ctx, result, "获取成功")
}
