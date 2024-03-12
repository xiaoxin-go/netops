package auth

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"math/rand"
	"netops/database"
	"netops/model"
	"netops/utils"
	"strings"
	"time"
)

func NewLoginHandler(operateId string) *loginHandler {
	return &loginHandler{OperateId: operateId}
}

type loginHandler struct {
	OperateId string
}

func (h *loginHandler) Login(username, password string) (*model.TUser, string, error) {
	l := zap.L().With(zap.String("func", "Login"), zap.String("OperateId", h.OperateId))
	l.Debug("用户登录")
	l.Debug("解密密码")
	pwd, e := h.decodePassword(password)
	if e != nil {
		l.Error("解密密码错误", zap.Error(e))
		return nil, "", e
	}
	l.Debug("获取用户信息")
	user := model.TUser{}
	if e := user.FirstByNameOrEmail(username); e != nil {
		return nil, "", e
	}

	l.Debug("验证密码")
	if user.Password != utils.HashString(pwd) {
		return nil, "", fmt.Errorf("密码错误")
	}
	l.Debug("验证otp")
	l.Debug("生成session_id")
	sessionId := h.geneSessionId()
	if e := h.hSetSessionId(sessionId, &user); e != nil {
		return nil, "", fmt.Errorf("保存session失败: %s", e.Error())
	}
	return &user, sessionId, nil
}
func (h *loginHandler) hSetSessionId(sessionId string, user *model.TUser) error {
	if e := database.R.HSet(sessionId, "username", user.Username).Err(); e != nil {
		return e
	}
	if e := database.R.HSet(sessionId, "user_id", user.Id).Err(); e != nil {
		return e
	}
	database.R.Expire(sessionId, 24*time.Hour)
	return nil
}

func (h *loginHandler) decodePassword(password string) (string, error) {
	privateKey, e := database.R.HGet(h.OperateId, "private_key").Result()
	if e != nil {
		return "", fmt.Errorf("获取私钥异常: %w", e)
	}
	if privateKey == "" {
		return "", errors.New("密钥过期，请刷新页面重试")
	}
	result, e := utils.RsaDecrypt(password, strings.TrimSpace(privateKey))
	if e != nil {
		return "", fmt.Errorf("密码解密失败: %w", e)
	}
	return result, nil
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func (h *loginHandler) geneSessionId() string {
	b := make([]rune, 32)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
