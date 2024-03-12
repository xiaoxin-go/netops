package menu

import (
	"github.com/gin-gonic/gin"
)

func Routers(e *gin.RouterGroup) {
	e.GET("/admin/menus", handler.List)
	e.GET("/admin/menu", handler.Get)
	e.POST("/admin/menu", handler.Create)
	e.PUT("/admin/menu", handler.Update)
	e.DELETE("/admin/menu", handler.Delete)
	e.POST("/admin/menu/relation_api", handler.RelationApi)
	e.GET("/admin/menu/relation_api_ids", handler.GetRelationApiIds)
}
