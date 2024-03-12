package firewall

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"netops/libs"
	"netops/pkg/policy"
)

type Handler struct {
	libs.Controller
}

var handler *Handler

func init() {
	handler = &Handler{}
}
func (h *Handler) List(ctx *gin.Context) {
	params := struct {
		DeviceId  int    `form:"device_id"`
		Page      int    `form:"page"`
		Size      int    `form:"size"`
		Direction string `form:"direction"`
		Src       string `form:"src"`
		Dst       string `form:"dst"`
		Port      string `form:"port"`
	}{}
	if e := ctx.ShouldBindQuery(&params); e != nil {
		libs.HttpParamsError(ctx, "参数解析失败, err: %w", e)
		return
	}
	result, total, e := policy.FirewallList(params.DeviceId, params.Page, params.Size, params.Direction, params.Src, params.Dst, params.Port)
	if e != nil {
		libs.HttpServerError(ctx, e.Error())
		return
	}
	ctx.JSON(http.StatusOK, libs.ListSuccess(result, total))
}
