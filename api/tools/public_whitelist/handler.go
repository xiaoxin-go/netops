package public_whitelist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"netops/database"
	"netops/libs"
	"netops/model"
	"netops/pkg/tools"
	"netops/utils"
)

type Handler struct {
	libs.Controller
}

var handler *Handler

func init() {
	handler = &Handler{}
	handler.NewInstance = func() libs.Instance {
		return new(model.TPublicWhitelist)
	}
	handler.NewResults = func() any {
		return &[]*model.TPublicWhitelist{}
	}
}

func (h *Handler) Parse(request *gin.Context) {
	params := struct {
		Id     int  `json:"id"`
		ToJson bool `json:"to_json"`
	}{}
	if err := request.ShouldBindJSON(&params); err != nil {
		libs.HttpParamsError(request, fmt.Sprintf("解析参数异常: <%s>", err.Error()))
		return
	}
	ph := tools.NewPublicWhitelistHandler(params.Id)
	result, e := ph.Parse()
	if e != nil {
		libs.HttpServerError(request, e.Error())
		return
	}
	if params.ToJson {
		libs.HttpSuccess(request, result, "ok")
		return
	}
	buffers, err := toXlsxBuffers(result)
	if err != nil {
		libs.HttpServerError(request, err.Error())
		return
	}
	request.Header("response-type", "blob")
	request.Data(http.StatusOK, "application/vnd.ms-excel", buffers.Bytes())
}

func (h *Handler) ParseAll(request *gin.Context) {
	params := struct {
		ToJson bool `json:"to_json"`
	}{}
	if err := request.ShouldBindJSON(&params); err != nil {
		libs.HttpParamsError(request, fmt.Sprintf("解析参数异常: <%s>", err.Error()))
		return
	}
	results := make([]*tools.PublicWhitelistResult, 0)
	publicWhitelist := make([]*model.TPublicWhitelist, 0)
	if err := database.DB.Find(&publicWhitelist).Error; err != nil {
		libs.HttpServerError(request, fmt.Sprintf("获取公网白名单信息异常: <%s>", err.Error()))
		return
	}
	for _, v := range publicWhitelist {
		ph := tools.NewPublicWhitelistHandler(v.Id)
		result, e := ph.Parse()
		if e != nil {
			libs.HttpServerError(request, e.Error())
			return
		}
		results = append(results, result...)
	}
	if params.ToJson {
		libs.HttpSuccess(request, results, "ok")
		return
	}
	buffers, err := toXlsxBuffers(results)
	if err != nil {
		libs.HttpServerError(request, err.Error())
		return
	}
	request.Header("response-type", "blob")
	request.Data(http.StatusOK, "application/vnd.ms-excel", buffers.Bytes())
}

func toXlsxBuffers(data []*tools.PublicWhitelistResult) (result *bytes.Buffer, err error) {
	maps := make([]map[string]interface{}, 0)
	bs, er := json.Marshal(&data)
	if er != nil {
		err = fmt.Errorf("转换结果异常: <%s>", err.Error())
		return
	}
	if er := json.Unmarshal(bs, &maps); er != nil {
		err = fmt.Errorf("转换结果异常: <%s>", err.Error())
		return
	}
	titles := []map[string]string{
		{"title": "属地", "key": "region"},
		{"title": "地址", "key": "address"},
		{"title": "端口", "key": "port"},
		{"title": "Vs", "key": "vs"},
		{"title": "Pool", "key": "pool"},
		{"title": "Members", "key": "host"},
	}
	xlsx := utils.Xlsx{}
	result = xlsx.NewFileToBuffer(titles, maps)
	return
}
