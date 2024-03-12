package model

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"netops/database"
	"netops/utils"
	"time"
)

type TLog struct {
	BaseModel
	Operator string `gorm:"column:operator" json:"operator"`
	Content  string `gorm:"column:content" json:"content"`
}

func (TLog) TableName() string {
	return "t_log"
}
func AddLog(operator, format string, a ...any) {
	content := fmt.Sprintf(format, a...)
	l := &TLog{Operator: operator, Content: content}
	if err := database.DB.Save(l).Error; err != nil {
		zap.L().Error("添加日志异常", zap.Error(err), zap.String("operator", operator), zap.String("content", content))
	}
}

type TApi struct {
	BaseModel
	Name    string   `gorm:"column:name" json:"name" binding:"required"`
	Uri     string   `gorm:"column:uri" json:"uri" binding:"required"`
	Method  string   `gorm:"column:method" json:"method"`
	MenuIds []int    `gorm:"-" json:"menu_ids" binding:"required"`
	Menus   []string `gorm:"-" json:"menus"`
	Enabled int      `gorm:"column:enabled" json:"enabled"`
	Select  bool     `gorm:"-" json:"select"`
}

func (TApi) TableName() string {
	return "t_api"
}
func (t *TApi) FirstById(id int) error {
	return firstById(t, id)
}
func (t *TApi) redisKey() string {
	return fmt.Sprintf("netops_api_info_%d", t.Id)
}
func (t *TApi) QueryById(id int) error {
	t.Id = id
	fields := []string{
		"name",
		"uri",
		"method",
		"select",
		"enabled",
	}
	return queryById(t, t.redisKey(), t.Id, fields)
}

// AfterCreate 添加接口后，创建接口与菜单的关系
func (t *TApi) AfterCreate(tx *gorm.DB) (err error) {
	// 添加好权限后，添加菜单权限
	err = t.bulkCreateMenuApi()
	return
}

// BeforeUpdate 更新接口前，重新关联与菜单的关系
func (t *TApi) BeforeUpdate(tx *gorm.DB) (err error) {
	// 先清除权限历史菜单
	if err = t.deleteMenuApi(); err != nil {
		return
	}
	// 重新构建与菜单的关系
	err = t.bulkCreateMenuApi()
	return
}

// BeforeDelete 删除接口前，清除与菜单的关系
func (t *TApi) BeforeDelete(tx *gorm.DB) (err error) {
	err = t.deleteMenuApi()
	return
}

// 批量构建接口与菜单的关系
func (t *TApi) bulkCreateMenuApi() (err error) {
	menuAuths := make([]*TMenuApi, 0)
	for _, v := range t.MenuIds {
		menuAuths = append(menuAuths, &TMenuApi{MenuId: v, ApiId: t.Id})
	}
	if e := database.DB.Create(&menuAuths).Error; e != nil {
		err = fmt.Errorf("关联权限菜单异常: <%s>", e.Error())
	}
	return err
}

// 清除接口与菜单的关系
func (t *TApi) deleteMenuApi() (err error) {
	if e := database.DB.Where("api_id = ?", t.Id).Delete(&TMenuApi{}).Error; e != nil {
		err = fmt.Errorf("删除权限菜单异常: <%s>", e.Error())
	}
	return
}

type TMenu struct {
	BaseModel
	Name       string   `gorm:"column:name" json:"name" binding:"required"`
	NameEn     string   `gorm:"column:name_en" json:"name_en" binding:"required"`
	Path       string   `gorm:"column:path" json:"path" binding:"required"`
	Icon       string   `gorm:"column:icon" json:"icon"`
	Sort       int      `gorm:"column:sort" json:"sort"`
	ParentId   int      `gorm:"column:parent_id" json:"parent_id"`
	ParentName string   `gorm:"-" json:"parent_name"`
	Children   []*TMenu `gorm:"-" json:"children" binding:"-"`
	Select     bool     `gorm:"-" json:"select" binding:"-"`
	Apis       []*TApi  `gorm:"-" json:"apis" binding:"-"`
	ApiIds     []int    `gorm:"-" json:"api_ids"`
	Enabled    int      `gorm:"column:enabled" json:"enabled"`
}

func (TMenu) TableName() string {
	return "t_menu"
}
func (t *TMenu) FirstById(id int) error {
	return firstById(t, id)
}
func (t *TMenu) redisKeyById() string {
	return fmt.Sprintf("netops_menu_info_by_id_%d", t.Id)
}
func (t *TMenu) FirstByUrl(url string) error {
	return firstByField(t, "url", url)
}

// QueryById 带有缓存的menu信息查询
func (t *TMenu) QueryById(id int) error {
	t.Id = id
	k := t.redisKeyById()
	fields := []string{
		"name",
		"name_en",
		"path",
		"icon",
		"parent_id",
		"sort",
		"enabled",
	}
	return queryById(t, k, id, fields)
}
func (t *TMenu) AfterFind(tx *gorm.DB) (err error) {
	if t.ParentId > 0 {
		menu := TMenu{}
		if e := menu.QueryById(t.ParentId); e == nil {
			t.ParentName = menu.Name
		}
	}
	return
}

func (t *TMenu) BeforeUpdate(tx *gorm.DB) (err error) {
	database.R.Del(t.redisKeyById())
	return nil
}
func (t *TMenu) AfterDelete(tx *gorm.DB) (err error) {
	if e := new(TMenuApi).DeleteByMenuId(t.Id, tx); e != nil {
		return e
	}
	return
}

type TMenuApi struct {
	EmptyModel
	MenuId int `gorm:"menu_id" json:"menu_id"`
	ApiId  int `gorm:"api_id" json:"api_id"`
}

func (t *TMenuApi) TableName() string {
	return "t_menu_api"
}
func (t *TMenuApi) PluckApiIdsByMenuId(menuId int) ([]int, error) {
	apiIds := make([]int, 0)
	if err := database.DB.Model(t).Where("menu_id = ?", menuId).Pluck("api_id", &apiIds).Error; err != nil {
		return nil, fmt.Errorf("获取菜单的api_id列表失败, menu_id: %d, err: %w", menuId, err)
	}
	return apiIds, nil
}
func (t *TMenuApi) DeleteByMenuId(menuId int, tx *gorm.DB) error {
	if tx == nil {
		tx = database.DB
	}
	if e := tx.Delete(t, "menu_id = ?", menuId).Error; e != nil {
		return fmt.Errorf("删除菜单关联api信息失败, menu_id: %d, err: %w", menuId, e)
	}
	return nil
}
func (t *TMenuApi) BulkCreate(data []*TMenuApi, tx *gorm.DB) error {
	if tx == nil {
		tx = database.DB
	}
	if len(data) == 0 {
		return nil
	}
	if e := tx.Create(&data).Error; e != nil {
		return fmt.Errorf("关联菜单与api失败, err: %w", e)
	}
	return nil
}

type TRole struct {
	BaseModel
	Name        string `gorm:"column:name;type:varchar(50);unique_index" json:"name" binding:"required"`
	Description string `gorm:"column:description" json:"description"`
}

func (TRole) TableName() string {
	return "t_role"
}

func (t *TRole) FirstById(id int) error {
	return firstById(t, id)
}
func (t *TRole) redisKeyById() string {
	return fmt.Sprintf("netops_role_info_by_id_%d", t.Id)
}

// QueryById 带有缓存的role信息查询
func (t *TRole) QueryById(id int) error {
	t.Id = id
	k := t.redisKeyById()
	fields := []string{
		"name",
		"description",
	}
	return queryById(t, k, id, fields)
}
func (t *TRole) PluckNamesByIds(ids []int) ([]string, error) {
	results := make([]string, 0)
	if e := database.DB.Model(t).Where("id in ?", ids).Pluck("name", &results).Error; e != nil {
		return nil, fmt.Errorf("获取角色名称列表失败, ids: %+v, err: %w", ids, e)
	}
	return results, nil
}

// BeforeDelete 添加角色前，清除角色与用户的关联
func (t *TRole) BeforeDelete(tx *gorm.DB) (err error) {
	// 清除casbin用户与角色关联
	if _, e := database.Casbin.DeleteRole(t.Id); e != nil {
		err = fmt.Errorf("清除casbin角色权限异常: <%s>", e.Error())
	}
	// 清除数据库中用户与角色的关联
	return t.deleteRoleUser()
}

// 删除用户与角色的关联
func (t *TRole) deleteRoleUser() (err error) {
	if e := database.DB.Where("role_id = ?", t.Id).Delete(&TUserRole{}).Error; e != nil {
		err = fmt.Errorf("删除用户角色关联异常: <%s>", e.Error())
	}
	return
}

type TRoleMenu struct {
	Id     int `gorm:"primary_key" json:"id"`
	RoleId int `gorm:"role_id" json:"role_id"`
	MenuId int `gorm:"menu_id" json:"menu_id"`
}

func (TRoleMenu) TableName() string {
	return "t_role_menu"
}

type TRoleApi struct {
	Id     int    `gorm:"primary_key" json:"id"`
	RoleId int    `gorm:"role_id" json:"role_id"`
	ApiId  int    `gorm:"author_id" json:"author_id"`
	Uri    string `gorm:"-" json:"uri"`
}

func (TRoleApi) TableName() string {
	return "t_role_api"
}

func (d *TRoleApi) AfterFind(tx *gorm.DB) (err error) {
	api := TApi{}
	if e := api.QueryById(d.ApiId); e == nil {
		d.Uri = api.Uri
	}
	return
}

type TUser struct {
	BaseModel
	Username          string    `gorm:"column:username" json:"username" binding:"required"`
	Password          string    `gorm:"column:password" json:"password" binding:"required"`
	OtpSecret         string    `gorm:"column:otp_secret" json:"otp_secret"`
	NameCn            string    `gorm:"column:name_cn" json:"name_cn" binding:"required"`
	Enabled           int       `gorm:"column:enabled" json:"enabled"`
	PasswordUpdatedAt time.Time `gorm:"column:password_updated_at" json:"password_updated_at"`
	TenantIds         []int     `gorm:"-" json:"tenant_ids"`
	Tenants           []string  `gorm:"-" json:"tenants"`
	RoleIds           []int     `gorm:"-" json:"role_ids"`
	Roles             []string  `gorm:"-" json:"roles"`
}

func (TUser) TableName() string {
	return "t_user"
}

func (t *TUser) FirstById(id int) error {
	return firstById(t, id)
}
func (t *TUser) redisKey() string {
	return fmt.Sprintf("netops_user_info_by_id_%d", t.Id)
}

// QueryById 带有缓存的用户信息查询
func (t *TUser) QueryById(id int) error {
	t.Id = id
	k := t.redisKey()
	fields := []string{"username", "name_cn", "email", "enabled", "created_at"}
	return queryById(t, k, id, fields)
}

// FirstByUsername 根据用户名获取用户信息
func (t *TUser) FirstByUsername(username string) error {
	return firstByField(t, "username", username)
}

// FirstByEmail 根据用户邮箱获取用户信息
func (t *TUser) FirstByEmail(email string) error {
	return firstByField(t, "email", email)
}

// FirstByNameOrEmail 根据用户名或邮箱查找用户
func (t *TUser) FirstByNameOrEmail(name string) error {
	if e := database.DB.Where("username = ?", name).First(t).Error; e != nil && errors.Is(e, gorm.ErrRecordNotFound) {
		return fmt.Errorf("用户信息不存在")
	} else if e != nil {
		return fmt.Errorf("获取用户信息失败, name: %s, err: %w", name, e)
	}
	return nil
}

func (t *TUser) FindByIds(ids []int) ([]*TUser, error) {
	results := make([]*TUser, 0)
	if e := database.DB.Where("id in ?", ids).Find(&results).Error; e != nil {
		return nil, fmt.Errorf("获取用户信息失败, ids: %v, err: %w", ids, e)
	}
	return results, nil
}
func (t *TUser) PluckNamesByIds(ids []int) ([]string, error) {
	results := make([]string, 0)
	if e := database.DB.Model(t).Where("id in ?", ids).Pluck("username", &results).Error; e != nil {
		return nil, fmt.Errorf("获取用户名列表失败, ids: %v, err: %w", ids, e)
	}
	return results, nil
}

// AfterFind 获取用户后，获取用户的角色信息
func (t *TUser) AfterFind(tx *gorm.DB) (err error) {
	roleIds, err := new(TUserRole).PluckRoleIdsByUserId(t.Id)
	if err != nil {
		return
	}
	roleNames, err := new(TRole).PluckNamesByIds(roleIds)
	if err != nil {
		return
	}
	t.Roles = roleNames
	t.RoleIds = roleIds
	return
}

func (t *TUser) BeforeCreate(tx *gorm.DB) (err error) {
	t.Password = utils.HashString(t.Password)
	return
}

// AfterCreate 添加用户后，创建用户与角色的关联
func (t *TUser) AfterCreate(tx *gorm.DB) (err error) {
	// 添加用户后，要添加用户角色并且添加到casbin
	err = t.addCasbinRole()
	if err != nil {
		return
	}
	err = t.bulkCreateUserRole()
	return
}

func (t *TUser) Create() (err error) {
	if e := database.DB.Create(t).Error; e != nil {
		zap.L().Error("创建用户失败", zap.Any("user", t), zap.Error(e))
		return fmt.Errorf("创建用户失败, user: %+v, err: %w", t, e)
	}
	return nil
}

// 添加casbin
func (t *TUser) addCasbinRole() error {
	if len(t.RoleIds) == 0 {
		return nil
	}
	if _, e := database.Casbin.AddUserRoles([]string{t.Username}, t.RoleIds); e != nil {
		return fmt.Errorf("关联用户和角色到casbin异常: <%s>", e.Error())
	}
	return nil
}

// 批量创建用户与角色的关联
func (t *TUser) bulkCreateUserRole() (err error) {
	if len(t.RoleIds) == 0 {
		return nil
	}
	bulks := make([]*TUserRole, 0)
	for _, v := range t.RoleIds {
		bulks = append(bulks, &TUserRole{RoleId: v, UserId: t.Id})
	}
	if e := database.DB.Create(&bulks).Error; e != nil {
		err = fmt.Errorf("关联用户角色异常: <%s>", e.Error())
	}
	return err
}

// 删除用户与角色的关联
func (t *TUser) deleteUserRole() (err error) {
	if e := database.DB.Where("user_id = ?", t.Id).Delete(&TUserRole{}).Error; e != nil {
		err = fmt.Errorf("删除用户角色关联异常: <%s>", e.Error())
	}
	return
}

// BeforeUpdate 更新用户信息前，先清除用户角色关联，然后再重新添加
func (t *TUser) BeforeUpdate(tx *gorm.DB) (err error) {
	// 清除casbin用户和角色关联
	if _, e := database.Casbin.DeleteUserRole(t.Username); e != nil {
		err = fmt.Errorf("删除用户<%s>的casbin角色关联异常: <%s>", t.Username, e.Error())
		return
	}
	// 添加用户和角色关联
	if err = t.deleteUserRole(); err != nil {
		return
	}
	// 重新构建casbin用户和角色
	return t.AfterCreate(tx)
}
func (t *TUser) AfterUpdate(tx *gorm.DB) (err error) {
	database.R.Del(t.redisKey())
	return
}

// BeforeDelete 删除用户前清除用户与角色的关联信息
func (t *TUser) BeforeDelete(tx *gorm.DB) (err error) {
	if err = t.deleteUserRole(); err != nil {
		return
	}
	return
}
func (t *TUser) AfterDelete(tx *gorm.DB) (err error) {
	database.R.Del(t.redisKey())
	return
}

type TUserRole struct {
	Id     int `gorm:"primary_key" json:"id"`
	UserId int `gorm:"column:user_id" json:"user_id"`
	RoleId int `gorm:"column:role_id" json:"role_id"`
}

func (TUserRole) TableName() string {
	return "t_user_role"
}

func (t *TUserRole) PluckRoleIdsByUserId(userId int) ([]int, error) {
	roleIds := make([]int, 0)
	if err := database.DB.Model(t).Where("user_id = ?", userId).Pluck("role_id", &roleIds).Error; err != nil {
		return nil, fmt.Errorf("获取用户的角色ID列表失败, user_id: %d, err: %w ", userId, err)
	}
	return roleIds, nil
}

// QueryApiMenusByApiIds 根据api id 获取所有菜单信息，并组装成 map[apiId][]menuId
func QueryApiMenusByApiIds(apiIds []int) (map[int][]int, error) {
	db := database.DB
	menuApis := make([]*TMenuApi, 0)
	if len(apiIds) > 0 {
		db = db.Where("api_id in ?", apiIds)
	}
	if e := db.Find(&menuApis).Error; e != nil {
		return nil, fmt.Errorf("获取菜单和api关联信息失败, err: %s", e)
	}
	result := make(map[int][]int, 0)
	for _, v := range menuApis {
		if _, ok := result[v.ApiId]; ok {
			result[v.ApiId] = append(result[v.ApiId], v.MenuId)
		} else {
			result[v.ApiId] = []int{v.MenuId}
		}
	}
	return result, nil
}

// QueryMenuApisByMenuIds 根据菜单 id 获取所有api信息，并组装成 map[menuId][]apiId
func QueryMenuApisByMenuIds(menuIds []int) (map[int][]int, error) {
	db := database.DB
	menuApis := make([]*TMenuApi, 0)
	if len(menuIds) > 0 {
		db = db.Where("menu_id in ?", menuIds)
	}
	if e := db.Find(&menuApis).Error; e != nil {
		return nil, fmt.Errorf("获取菜单和api关联信息异常, err: %s", e)
	}
	result := make(map[int][]int, 0)
	for _, v := range menuApis {
		if _, ok := result[v.MenuId]; ok {
			result[v.MenuId] = append(result[v.MenuId], v.ApiId)
		} else {
			result[v.MenuId] = []int{v.ApiId}
		}
	}
	return result, nil
}
