package menu

import (
	"github.com/gin-gonic/gin"
	"netops/database"
	"netops/libs"
	"netops/model"
)

type Handler struct {
	libs.Controller
}

var handler *Handler

func init() {
	handler = &Handler{}
	handler.NewInstance = func() libs.Instance {
		return new(model.TMenu)
	}
	handler.NewResults = func() any {
		return &[]*model.TMenu{}
	}
}

func (h *Handler) GetRelationApiIds(ctx *gin.Context) {
	params := struct {
		MenuId int `form:"menu_id" binding:"required"`
	}{}
	if e := ctx.ShouldBindQuery(&params); e != nil {
		libs.HttpParamsError(ctx, "解析参数失败, err: %w", e)
		return
	}
	//menuId, _ := strconv.Atoi(params.MenuId)
	apiIds, e := new(model.TMenuApi).PluckApiIdsByMenuId(params.MenuId)
	if e != nil {
		libs.HttpServerError(ctx, e.Error())
		return
	}
	libs.HttpSuccess(ctx, apiIds, "获取成功")
}

func (h *Handler) RelationApi(ctx *gin.Context) {
	params := struct {
		MenuId int   `json:"menu_id" binding:"required"`
		ApiIds []int `json:"api_ids" binding:"required"`
	}{}
	if e := ctx.ShouldBindJSON(&params); e != nil {
		libs.HttpParamsError(ctx, "解析参数失败, err: %w", e)
		return
	}
	menu := model.TMenu{}
	if e := menu.FirstById(params.MenuId); e != nil {
		libs.HttpServerError(ctx, e.Error())
		return
	}
	tx := database.DB.Begin()
	if e := new(model.TMenuApi).DeleteByMenuId(menu.Id, tx); e != nil {
		tx.Rollback()
		libs.HttpServerError(ctx, e.Error())
		return
	}
	menuAuths := make([]*model.TMenuApi, 0)
	for _, v := range params.ApiIds {
		menuAuths = append(menuAuths, &model.TMenuApi{ApiId: v, MenuId: menu.Id})
	}
	if e := new(model.TMenuApi).BulkCreate(menuAuths, tx); e != nil {
		tx.Rollback()
		libs.HttpServerError(ctx, e.Error())
		return
	}
	if e := tx.Commit().Error; e != nil {
		libs.HttpServerError(ctx, "关联失败, err: %w", e)
		return
	}
	libs.HttpSuccess(ctx, nil, "关联成功")
	return
}
