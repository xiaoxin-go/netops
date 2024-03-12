package database

import (
	"fmt"
	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"sync"
)

var (
	once   sync.Once
	Casbin *casbinHandler
)

type casbinHandler struct {
	syncedEnforcer *casbin.SyncedEnforcer
}

func (c *casbinHandler) init() {
	once.Do(func() {
		adapter, err := gormadapter.NewAdapterByDB(DB)
		if err != nil {
			panic(err)
		}
		c.syncedEnforcer, err = casbin.NewSyncedEnforcer("conf/rbac_model.conf", adapter)
		if err != nil {
			panic(err)
		}
	})
	c.syncedEnforcer.AddFunction("isAdmin", func(arguments ...interface{}) (interface{}, error) {
		// 获取用户名
		username := arguments[0].(string)
		// 检查用户名的角色是否为超级管理员
		return c.syncedEnforcer.HasRoleForUser(username, "role_1")
	})
	err := c.syncedEnforcer.LoadPolicy()
	if err != nil {
		panic(err)
	}
}

func (c *casbinHandler) Enforce(user, uri, action string) (bool, error) {
	return c.syncedEnforcer.Enforce(user, uri, action)
}

// AddPolicy 添加策略
func (c *casbinHandler) AddPolicy(roleId int, uri, method string) (bool, error) {
	return c.syncedEnforcer.AddPolicy(c.MakeRoleName(roleId), uri, method)
}

func (c *casbinHandler) MakeRoleName(roleId int) string {
	return fmt.Sprintf("role_%d", roleId)
}

// AddPolicies 批量添加策略
func (c *casbinHandler) AddPolicies(rules [][]string) (bool, error) {
	return c.syncedEnforcer.AddPolicies(rules)
}

// DeleteRole 删除角色对应的用户和权限
func (c *casbinHandler) DeleteRole(roleId int) (bool, error) {
	return c.syncedEnforcer.DeleteRole(c.MakeRoleName(roleId))
}

// DeleteRolePolicy 删除角色下的权限
func (c *casbinHandler) DeleteRolePolicy(roleId int) (bool, error) {
	return c.syncedEnforcer.RemoveFilteredNamedPolicy("p", 0, c.MakeRoleName(roleId))
}

// DeleteRoleUser 删除添加用户
func (c *casbinHandler) DeleteRoleUser(roleId int) (bool, error) {
	return c.syncedEnforcer.RemoveFilteredNamedGroupingPolicy("g", 1, c.MakeRoleName(roleId))
}

// AddUserRole 添加角色和用户对应关系
func (c *casbinHandler) AddUserRole(user string, roleId int) (bool, error) {
	return c.syncedEnforcer.AddGroupingPolicy(user, c.MakeRoleName(roleId))
}

// AddUserRoles 批量添加角色和用户对应关联
func (c *casbinHandler) AddUserRoles(usernames []string, roleIds []int) (bool, error) {
	rules := make([][]string, 0)
	for _, u := range usernames {
		for _, r := range roleIds {
			rules = append(rules, []string{u, c.MakeRoleName(r)})
		}
	}
	return c.syncedEnforcer.AddGroupingPolicies(rules)
}

// DeleteUserRole 删除用户的角色信息
func (c *casbinHandler) DeleteUserRole(user string) (bool, error) {
	return c.syncedEnforcer.RemoveFilteredNamedGroupingPolicy("g", 0, user)
}
