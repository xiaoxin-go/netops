package invalid_policy_task

import (
	"netops/libs"
	"netops/model"
)

var policyHitCountHandler *Handler

func init() {
	policyHitCountHandler = &Handler{}
	policyHitCountHandler.NewInstance = func() libs.Instance {
		return new(model.TDevicePolicyHitCount)
	}
	policyHitCountHandler.NewResults = func() any {
		return &[]*model.TDevicePolicyHitCount{}
	}
}
