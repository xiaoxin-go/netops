package auth

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"netops/database"
	"netops/model"
	"netops/utils"
	"regexp"
	"time"
)

func NewResetPasswordHandler(operateId string) *resetPasswordHandler {
	h := &resetPasswordHandler{}
	h.OperateId = operateId
	return h
}

type resetPasswordHandler struct {
	OperateId string
	Err       error
}

func (r *resetPasswordHandler) decodePwd(pwd string) (string, error) {
	privateKey, e := database.R.HGet(r.OperateId, "private_key").Result()
	if e != nil {
		return "", fmt.Errorf("密码私钥已过期，请重试")
	}
	result, e := utils.RsaDecrypt(pwd, privateKey)
	if e != nil {
		return "", fmt.Errorf("解密失败: %w", e)
	}
	return result, nil
}

func (h *resetPasswordHandler) Reset(user *model.TUser, password string) error {
	l := zap.L().With(zap.String("func", "修改密码"), zap.String("OperateId", h.OperateId))
	l.Info("用户修改密码")
	l.Info("解密密码")
	pwd, e := h.decodePwd(password)
	if e != nil {
		return e
	}

	l.Info("校验密码格式")
	if !h.verifyPwd(pwd) {
		return errors.New("密码格式不正确")
	}
	l.Info("更新用户密码")
	if e := database.DB.Model(user).Updates(map[string]any{
		"password":            utils.HashString(pwd),
		"password_updated_at": time.Now()}).Error; e != nil {
		return fmt.Errorf("密码更新失败: %s", e.Error())
	}
	return nil
}

// 校对密码格式
func (r *resetPasswordHandler) verifyPwd(pwd string) bool {
	// 密码不能小于8位
	if len(pwd) < 8 {
		return false
	}
	if ok, _ := regexp.MatchString("[a-z]", pwd); !ok {
		return false
	}
	if ok, _ := regexp.MatchString("[A-Z]", pwd); !ok {
		return false
	}
	if ok, _ := regexp.MatchString("[!@#$%^&*]", pwd); !ok {
		return false
	}
	return true
}
