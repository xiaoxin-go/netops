package firewall

import (
	"fmt"
	"github.com/gin-gonic/gin"
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
		return new(model.TFirewallDevice)
	}
	handler.NewResults = func() any {
		return &[]*model.TFirewallDevice{}
	}
}

type operateParams struct {
	DeviceId int `json:"device_id"`
}

func (h *Handler) Policy(ctx *gin.Context) {
	params := new(operateParams)
	err := ctx.ShouldBindJSON(params)
	if err != nil {
		libs.HttpParamsError(ctx, fmt.Sprintf("接收参数异常: <%s>", err.Error()))
		return
	}
	parser, err := device.NewDeviceHandler(params.DeviceId)
	if err != nil {
		libs.HttpServerError(ctx, err.Error())
		return
	}
	go parser.ParseConfig()
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
	if e := result.LastByDeviceIdAndType(id, "firewall"); e != nil {
		libs.HttpServerError(ctx, e.Error())
		return
	}
	libs.HttpSuccess(ctx, result, "获取成功")
}
func (h *Handler) Backup(ctx *gin.Context) {
	params := new(operateParams)
	err := ctx.ShouldBindJSON(params)
	if err != nil {
		libs.HttpParamsError(ctx, fmt.Sprintf("接收参数失败, err: %w", err))
		return
	}
	dh, e := device.NewDeviceHandler(params.DeviceId)
	if e != nil {
		libs.HttpServerError(ctx, e.Error())
		return
	}
	if e := dh.Backup(); e != nil {
		libs.HttpServerError(ctx, e.Error())
		return
	}
	libs.HttpSuccess(ctx, nil, "备份成功")
}
