package tools

import "netops/model"

type base struct {
	region *model.TRegion
	device *model.TFirewallDevice
	error  error
}

func (b *base) Error() error {
	return b.error
}
