package libs

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"netops/conf"
	"netops/database"
	"netops/model"
	"regexp"
	"strings"
)

func getUser(ctx *gin.Context) (user *model.TUser, err error) {
	user, err = GetUser(ctx)
	if err != nil {
		return
	}
	if user.Enabled == 0 {
		err = fmt.Errorf("用户已禁用")
	}
	return
}

func isExcludeLoginAuth(method, uri string) bool {
	for _, v := range conf.LoginExcludeAuth {
		if method == v.Method {
			if ok, _ := regexp.MatchString(v.Uri, uri); ok {
				return true
			}
		}
	}
	return false
}

// 地址是否排除
func isExcludeAuth(method, uri string) bool {
	for _, v := range conf.ExcludeAuth {
		if method == v.Method {
			if ok, _ := regexp.MatchString(v.Uri, uri); ok {
				return true
			}
		}
	}
	return false
}

// CasbinAuthor 使用casbin进行访问控制权限
func CasbinAuthor() gin.HandlerFunc {
	return func(request *gin.Context) {
		// 获取请求接口和方法
		obj := strings.TrimRight(request.Request.URL.Path, "/")
		objs := strings.Split(obj, "/api/v1")
		obj = objs[len(objs)-1]
		act := request.Request.Method
		// 排除不需要校验的权限
		if isExcludeAuth(act, obj) {
			request.Next()
			return
		}
		// 获取用户信息
		user, err := getUser(request)
		if err != nil {
			request.JSON(http.StatusOK, SessionError(err.Error()))
			request.Abort()
			return
		}
		if isExcludeLoginAuth(act, obj) {
			request.Next()
			return
		}
		// 循环用户角色ID，如果有一个角色拥有权限则设置为true
		success, err := database.Casbin.Enforce(user.Username, obj, act)
		if err != nil {
			request.JSON(http.StatusOK, ServerError(err.Error()))
			request.Abort()
			return
		}
		if !success {
			HttpAuthorError(request, "用户没有操作权限")
			request.Abort()
			return
		}
	}
}

// Cors 支持跨域
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		//接收客户端发送的origin （重要！）
		c.Header("Access-Control-Allow-Origin", "*")
		//服务器支持的所有跨域请求的方法
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		//允许跨域设置可以返回其他子段，可以自定义字段
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Length, X-CSRF-Token, Token,session,Content-Type")
		// 允许浏览器（客户端）可以解析的头部 （重要）
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers,Content-Type")
		//设置缓存时间
		c.Header("Access-Control-Max-Age", "172800")
		//允许客户端传递校验信息比如 cookie (重要)
		c.Header("Access-Control-Allow-Credentials", "true")

		//允许类型校验
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}

		//defer func() {
		//	if err := recover(); err != nil {
		//		log.Printf("Panic info is: %v", err)
		//	}
		//}()

		c.Next()
	}
}
