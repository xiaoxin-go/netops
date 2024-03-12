package firewall_nat

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
		return new(model.TDeviceNat)
	}
	handler.NewResults = func() any {
		return &[]*model.TDeviceNat{}
	}
}
