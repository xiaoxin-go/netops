package libs

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

// HttpResponse 封装接口返回类型
type HttpResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type ListResponse struct {
	HttpResponse
	Page  int   `json:"page"`
	Size  int   `json:"size"`
	Count int64 `json:"count"`
}

func ListSuccess(data interface{}, count int64) *ListResponse {
	r := &ListResponse{}
	r.Code = 0
	r.Data = data
	r.Count = count
	r.Msg = "ok"
	return r
}

// 公共方法
func response(code int, data interface{}, message string) *HttpResponse {
	h := &HttpResponse{}
	h.Code = code
	h.Data = data
	h.Msg = message
	return h
}

// Success 请求成功，返回状态码为200， 消息， 数据
func Success(data interface{}, message string) *HttpResponse {
	return response(0, data, message)
}

// ServerError 服务器异常，返回状态码为500， 消息
func ServerError(message string) *HttpResponse {
	return response(500, nil, message)
}

// ParamsError 参数异常， 返回状态码400， 消息
func ParamsError(message string) *HttpResponse {
	return response(400, nil, message)
}

// AuthorError 无访问权限
func AuthorError(message string) *HttpResponse {
	return response(405, nil, message)
}

// SessionError 认证失败
func SessionError(message string) *HttpResponse {
	return response(401, nil, message)
}

func HttpListSuccess(ctx *gin.Context, data interface{}, count int64) {
	ctx.JSON(http.StatusOK, ListSuccess(data, count))
}
func HttpSuccess(ctx *gin.Context, data interface{}, format string, a ...any) {
	ctx.JSON(http.StatusOK, Success(data, fmt.Sprintf(format, a...)))
}
func HttpParamsError(ctx *gin.Context, format string, a ...any) {
	ctx.JSON(http.StatusOK, ParamsError(fmt.Sprintf(format, a...)))
}
func HttpServerError(ctx *gin.Context, format string, a ...any) {
	ctx.JSON(http.StatusOK, ServerError(fmt.Sprintf(format, a...)))
}
func HttpAuthorError(ctx *gin.Context, format string, a ...any) {
	ctx.JSON(http.StatusOK, AuthorError(fmt.Sprintf(format, a...)))
}
func HttpSessionError(ctx *gin.Context, format string, a ...any) {
	ctx.JSON(http.StatusOK, SessionError(fmt.Sprintf(format, a...)))
}
