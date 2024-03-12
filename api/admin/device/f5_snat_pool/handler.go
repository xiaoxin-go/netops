package f5_snat_pool

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
		return new(model.TF5SnatPool)
	}
	handler.NewResults = func() any {
		return &[]*model.TF5SnatPool{}
	}
}
