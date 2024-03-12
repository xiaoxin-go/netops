package backup

import (
	"github.com/gin-gonic/gin"
	"net/http"
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
		return new(model.TDeviceBackup)
	}
	handler.NewResults = func() any {
		return &[]*model.TDeviceBackup{}
	}
}

func (h *Handler) Download(ctx *gin.Context) {
	id, e := h.GetId(ctx)
	if e != nil {
		libs.HttpParamsError(ctx, e.Error())
		return
	}
	backHandler := device.NewBackupHandler(id)
	result, e := backHandler.Download()
	if e != nil {
		ctx.JSON(500, e.Error())
		return
	}
	ctx.Header("response-type", "blob")
	ctx.Data(http.StatusOK, "application/zip", result)
}
