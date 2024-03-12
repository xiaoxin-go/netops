package auth

import "github.com/gin-gonic/gin"

func Routers(e *gin.RouterGroup) {
	e.GET("/auth/public_key", GetPublicKey)       // 获取公钥
	e.POST("/auth/login", Login)                  // 登录
	e.POST("/auth/reset_password", ResetPassword) // 重置密码
	e.GET("/auth/user_info", GetUserInfo)
	e.GET("/auth/menus", GetMenus)
}
