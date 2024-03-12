package tools

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"netops/database"
	"netops/model"
	"netops/pkg/device"
)

func NewInvalidPolicyHandler(id int) *invalidPolicyHandler {
	r := &invalidPolicyHandler{id: id}
	r.init()
	return r
}

type invalidPolicyHandler struct {
	id     int
	data   *model.TInvalidPolicyTask
	region *model.TRegion
	device *model.TFirewallDevice
	error  error
}

func (h *invalidPolicyHandler) Error() error {
	return h.error
}
func (h *invalidPolicyHandler) init() {
	h.initData()
	h.initDevice()
	h.initRegion()
}

func (h *invalidPolicyHandler) initData() {
	data := model.TInvalidPolicyTask{}
	if e := data.FirstById(h.id); e != nil {
		h.error = e
		return
	}
	h.data = &data
}

func (h *invalidPolicyHandler) initDevice() {
	if h.error != nil {
		return
	}
	d := model.TFirewallDevice{}
	if e := d.FirstById(h.data.DeviceId); e != nil {
		h.error = e
		return
	}
	h.device = &d
}
func (h *invalidPolicyHandler) initRegion() {
	if h.error != nil {
		return
	}
	r := model.TRegion{}
	if e := r.FirstById(h.data.RegionId); e != nil {
		h.error = e
		return
	}
	h.region = &r
}
func (h *invalidPolicyHandler) Parse() error {
	log := zap.L().With(zap.String("func", "InvalidPolicyParse"), zap.Int("id", h.data.Id))
	if h.error != nil {
		return h.error
	}
	log.Debug("1. 校验任务状态是否为running----->")
	if h.data.Status == "running" {
		return errors.New("任务正在执行中")
	}
	log.Debug("2. 修改任务状态为running------->")
	if e := h.updateStatus("running"); e != nil {
		return e
	}
	log.Debug("3. 开始解析----------->")
	if e := h.parse(); e != nil {
		return e
	}
	if h.error != nil {
		log.Debug(fmt.Sprintf("解析失败: <%s>", h.error.Error()))
		if e := h.updateStatus("failed"); e != nil {
			return e
		}
	} else {
		log.Debug("4. 解析成功！")
		if e := h.updateStatus("success"); e != nil {
			return e
		}
	}
	return nil
}
func (h *invalidPolicyHandler) parse() error {
	p, e := device.NewDeviceHandler(h.data.DeviceId)
	if e != nil {
		return e
	}
	p.ParseInvalidPolicy()
	if p.Error() != nil {
		return p.Error()
	}
	return nil
}
func (h *invalidPolicyHandler) updateStatus(status string) error {
	if e := database.DB.Model(h.data).Update("status", status).Error; e != nil {
		return fmt.Errorf("更新任务状态异常: <%s>", e.Error())
	}
	return nil
}
