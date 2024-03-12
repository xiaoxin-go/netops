package role

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"netops/database"
	"netops/libs"
	"netops/model"
	"netops/pkg/auth"
)

type Handler struct {
	libs.Controller
}

var handler *Handler

func init() {
	handler = &Handler{}
	handler.NewInstance = func() libs.Instance {
		return new(model.TRole)
	}
	handler.NewResults = func() any {
		return &[]*model.TRole{}
	}
}

// GetAuthor 获取角色权限信息
// 1. 查询角色信息是否存在
// 2. 获取所有菜单信息, 获取角色拥有菜单信息，将角色拥有的菜单信息的select设置为true
// 3. 获取所有权限信息，获取角色拥有权限信息，将角色拥有的权限信息的select设置为true，并挂在菜单ID下
// 返回的数据格式是这样的
// {"code": 200, "message": "ok", "data": {
//	"data_list": [
//		{"id": 1, "name": "用户管理", "select": true, "child": [以下嵌套的是子菜单]}
//	]
//}}

func (h *Handler) GetPermission(ctx *gin.Context) {
	// 获取角色权限信息
	roleId, err := h.GetId(ctx)
	if err != nil {
		libs.HttpParamsError(ctx, err.Error())
		return
	}
	results, err := auth.GetMenu([]int{roleId})
	if err != nil {
		msg := fmt.Sprintf("获取菜单信息异常: <%s>", err.Error())
		zap.L().Error(msg)
		ctx.JSON(http.StatusOK, libs.ServerError(msg))
		return
	}
	ctx.JSON(http.StatusOK, libs.Success(results, "ok"))
}

// UpdatePermission 修改角色权限信息
// 接收数据： {"menu_list": [], "author_list": []}	// 里面存的是菜单ID和权限ID
// 执行逻辑： 删除角色原有的菜单和权限信息，根据选择的重新添加即可
func (h *Handler) UpdatePermission(ctx *gin.Context) {
	l := zap.L().With(zap.String("func", "updatePermission"))
	params := struct {
		RoleId  int   `json:"role_id" binding:"required"`
		MenuIds []int `json:"menu_ids" binding:"required"`
		ApiIds  []int `json:"api_ids" binding:"required"`
	}{}
	l.Info("更新角色权限", zap.Any("params", params))
	if err := ctx.ShouldBindJSON(&params); err != nil {
		libs.HttpParamsError(ctx, fmt.Sprintf("参数解析失败, err: %s", err.Error()))
		return
	}
	role := model.TRole{}
	if err := role.FirstById(params.RoleId); err != nil {
		libs.HttpServerError(ctx, err.Error())
		return
	}
	l.Info("1. 删除角色权限信息..............")
	// 删除角色权限信息
	tx := database.DB.Begin()
	if err := tx.Where("role_id = ?", params.RoleId).Delete(&model.TRoleApi{}).Error; err != nil {
		tx.Rollback()
		l.Error("删除角色权限失败", zap.Error(err))
		libs.HttpServerError(ctx, "删除角色权限信息失败, err: %s ", err.Error())
		return
	}
	l.Info("2. 删除角色菜单信息...................")
	// 删除角色菜单信息
	if err := tx.Where("role_id = ?", params.RoleId).Delete(&model.TRoleMenu{}).Error; err != nil {
		tx.Rollback()
		l.Error("删除角色关联菜单失败", zap.Error(err))
		libs.HttpServerError(ctx, "删除角色菜单信息失败, err: %s", err.Error())
		return
	}
	l.Info("3. 添加角色菜单信息....................")
	// 添加角色菜单信息
	roleMenuList := make([]model.TRoleMenu, 0)
	for _, item := range params.MenuIds {
		roleMenuList = append(roleMenuList, model.TRoleMenu{RoleId: params.RoleId, MenuId: item})
	}
	if err := tx.Create(&roleMenuList).Error; err != nil {
		tx.Rollback()
		l.Error("关联角色与菜单失败", zap.Error(err))
		libs.HttpServerError(ctx, "关联角色菜单失败, err: %s", err.Error())
		return
	}
	l.Info("4. 添加角色权限信息....................")
	// 添加角色权限信息
	roleAuthorList := make([]*model.TRoleApi, 0)
	rules := make([][]string, 0)
	for v, _ := range apisToMap(params.ApiIds) {
		api := model.TApi{}
		_ = api.QueryById(v)
		roleAuthorList = append(roleAuthorList, &model.TRoleApi{RoleId: params.RoleId, ApiId: v})
		rules = append(rules, []string{database.Casbin.MakeRoleName(role.Id), api.Uri, api.Method})
	}
	if err := tx.Create(&roleAuthorList).Error; err != nil {
		tx.Rollback()
		l.Error("关联角色与权限失败", zap.Error(err))
		libs.HttpServerError(ctx, "关联角色和权限失败, err: %s ", err.Error())
		return
	}
	// 删除casbin权限
	if _, err := database.Casbin.DeleteRolePolicy(role.Id); err != nil {
		tx.Rollback()
		l.Error("清除casbin角色权限失败", zap.Error(err))
		libs.HttpServerError(ctx, "清除casbin角色权限失败, err: %s", err.Error())
		return
	}
	// 添加新的casbin权限
	if ok, err := database.Casbin.AddPolicies(rules); !ok || err != nil {
		tx.Rollback()
		l.Error("添加casbin角色权限失败", zap.Error(err))
		libs.HttpServerError(ctx, "添加casbin角色权限失败, err: %s", err.Error())
		return
	}
	tx.Commit()
	libs.HttpSuccess(ctx, nil, "更新成功")
}

// 对apis进行去重
func apisToMap(apis []int) map[int]struct{} {
	m := make(map[int]struct{})
	for _, v := range apis {
		m[v] = struct{}{}
	}
	return m
}
