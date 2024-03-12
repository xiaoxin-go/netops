package device_nat_pool

import (
	"netops/libs"
	"netops/model"
)

type Handler struct {
	libs.Controller
}

var handler *Handler

func init() {
	handler = &Handler{}
	handler.NewInstance = func() libs.Instance {
		return new(model.TDeviceNatPool)
	}
	handler.NewResults = func() any {
		return &[]*model.TDeviceNatPool{}
	}
}
