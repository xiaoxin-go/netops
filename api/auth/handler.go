package auth

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"netops/conf"
	"netops/database"
	"netops/libs"
	"netops/model"
	"netops/pkg/auth"
	"netops/utils"
	"time"
)

type loginRequest struct {
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
	OperateId string `json:"operate_id" binding:"required"`
}

// Login 登录
func Login(ctx *gin.Context) {
	params := loginRequest{}
	if e := ctx.ShouldBindJSON(&params); e != nil {
		libs.HttpParamsError(ctx, e.Error())
		return
	}
	h := auth.NewLoginHandler(params.OperateId)
	user, sessionId, err := h.Login(params.Username, params.Password)
	if err != nil {
		libs.HttpServerError(ctx, err.Error())
		return
	}
	ctx.SetCookie(conf.SessionKey, sessionId, 3600*24, "/", "", false, true)
	time.Sleep(time.Second)
	libs.HttpSuccess(ctx, user, "登录成功")
}

type retrievePasswordRequest struct {
	OperateId string `json:"operate_id" binding:"required"` // 操作ID
	Password  string `json:"password" binding:"required"`   // 加密后的密码
}

// ResetPassword 修改密码
func ResetPassword(ctx *gin.Context) {
	params := retrievePasswordRequest{}
	if e := ctx.ShouldBindJSON(&params); e != nil {
		libs.HttpParamsError(ctx, e.Error())
		return
	}
	user, err := libs.GetUser(ctx)
	if err != nil {
		libs.HttpAuthorError(ctx, err.Error())
		return
	}
	h := auth.NewResetPasswordHandler(params.OperateId)
	if e := h.Reset(user, params.Password); e != nil {
		libs.HttpServerError(ctx, err.Error())
		return
	}
	libs.HttpSuccess(ctx, nil, "修改成功")
}

// GetPublicKey 获取公钥
func GetPublicKey(ctx *gin.Context) {
	operateId := ctx.Query("operate_id")
	if operateId == "" {
		libs.HttpParamsError(ctx, "operate_id不能为空")
		return
	}
	publicKey, privateKey, err := utils.GenerateKey()
	if err != nil {
		libs.HttpServerError(ctx, "公钥生成失败: %s", err.Error())
		return
	}
	// 保存私钥
	if e := database.R.HSet(operateId, "private_key", privateKey).Err(); e != nil {
		libs.HttpServerError(ctx, "保存私钥失败: %s", e.Error())
		return
	}
	database.R.Expire(operateId, 5*time.Minute)
	libs.HttpSuccess(ctx, publicKey, "ok")
}

// GetUserInfo 获取用户信息
func GetUserInfo(ctx *gin.Context) {
	user, e := libs.GetUser(ctx)
	if e != nil {
		libs.AuthorError(e.Error())
		return
	}
	libs.HttpSuccess(ctx, user, "ok")
}

func GetMenus(ctx *gin.Context) {
	user, e := libs.GetUser(ctx)
	if e != nil {
		libs.HttpAuthorError(ctx, e.Error())
		return
	}
	zap.L().Debug(fmt.Sprintf("用户获取菜单------><%+v>", user))
	roleIds, err := new(model.TUserRole).PluckRoleIdsByUserId(user.Id)
	fmt.Println("role_ids-------->", roleIds)
	if err != nil {
		libs.HttpServerError(ctx, err.Error())
		return
	}
	result, err := auth.GetMenu(roleIds)
	if err != nil {
		msg := fmt.Sprintf("获取菜单信息异常: <%s>", err.Error())
		zap.L().Error(msg)
		libs.HttpServerError(ctx, msg)
		return
	}
	libs.HttpSuccess(ctx, result, "ok")
}
