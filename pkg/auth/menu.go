package auth

import (
	"fmt"
	"netops/database"
	"netops/model"
)

// GetApis 获取所有权限列表，并且根据用户角色是否拥有权限，设置select的值
func GetApis(roleIds []int) (apis []*model.TApi, err error) {
	apis = make([]*model.TApi, 0)
	if e := database.DB.Where("enabled = ?", 1).Find(&apis).Error; e != nil {
		err = fmt.Errorf("获取api信息异常: <%s>", e.Error())
		return
	}
	// 取出角色拥有的权限信息
	roleApis := make([]*model.TRoleApi, 0)
	if e := database.DB.Where("role_id in ?", roleIds).Find(&roleApis).Error; e != nil {
		err = fmt.Errorf("获取角色API异常: <%s>", e.Error())
		return
	}
	selectApis := make(map[int]bool)
	for _, v := range roleApis {
		selectApis[v.ApiId] = true
	}
	// 为权限信息设置select状态
	for _, v := range apis {
		v.Select = IsAdmin(roleIds) || selectApis[v.Id]
	}
	return
}

func GetMenu(roleIds []int) (result []*model.TMenu, err error) {
	// 1. 获取所有启用的菜单
	menuList := make([]*model.TMenu, 0)
	if e := database.DB.Where("enabled = ?", 1).Order("sort").Find(&menuList).Error; e != nil {
		err = fmt.Errorf("获取菜单信息异常: <%s>", e.Error())
		return
	}
	// 2. 获取所有权限信息，并根据角色设置是否选中
	apis, err := GetApis(roleIds)
	if err != nil {
		return
	}
	// 获取菜单和api的所有关联关系
	menuApis, err := model.QueryApiMenusByApiIds([]int{})
	if err != nil {
		return
	}
	// 3. 将同菜单下的权限挂到菜单ID下
	menuApiDict := make(map[int][]*model.TApi)
	for _, api := range apis {
		for _, mi := range menuApis[api.Id] { // 根据api id获取到api的菜单列表，然后将api添加到菜单列表里
			if _, ok := menuApiDict[mi]; !ok {
				menuApiDict[mi] = []*model.TApi{api}
			} else {
				menuApiDict[mi] = append(menuApiDict[mi], api)
			}
		}
	}
	// 4. 获取角色拥有的菜单信息
	roleMenuList := make([]model.TRoleMenu, 0)
	if e := database.DB.Where("role_id in ?", roleIds).Find(&roleMenuList).Error; e != nil {
		err = fmt.Errorf("获取角色的菜单信息异常: <%s>", e.Error())
		return
	}
	// 为角色拥有的菜单ID设置select为true
	selectMenu := make(map[int]bool)
	for _, roleMenu := range roleMenuList {
		selectMenu[roleMenu.MenuId] = true
	}

	menuDict := make(map[int]*model.TMenu) // 存储所有菜单的id对应值
	// 5. 为所有菜单增加select是否选中和权限信息
	for _, menu := range menuList {
		menu.Apis = menuApiDict[menu.Id]
		menu.Children = make([]*model.TMenu, 0)
		if IsAdmin(roleIds) { // 如果是超级管理员，则菜单选中全部设置为true
			menu.Select = true
		} else {
			menu.Select = selectMenu[menu.Id]
		}
		// 这里是为了有些菜单的权限是挂在父菜单上，所以要把子菜单显示出来
		//if menu.ParentId > 0 && len(menu.Apis) == 0 {
		//	fmt.Println("menu---------->", menu.Id)
		//	menu.Select = true
		//}
		menuDict[menu.Id] = menu
	}
	// 6. 循环所有菜单信息，将子菜单挂在父菜单头上
	for _, menu := range menuList {
		if menu.ParentId > 0 {
			menuDict[menu.ParentId].Children = append(menuDict[menu.ParentId].Children, menu)
		}
	}
	// 7. 过滤出一级菜单，挂出来即可，没有父菜单ID的则为一级菜单
	result = make([]*model.TMenu, 0)
	for _, menu := range menuList {
		if menu.ParentId == 0 {
			result = append(result, menu)
		}
	}
	return
}

func IsAdmin(roleIds []int) bool {
	for _, v := range roleIds {
		if v == 1 {
			return true
		}
	}
	return false
}

func GetUserMenuApis(username, menuUrl string) ([]string, error) {
	// 获取菜单信息
	menu := model.TMenu{}
	if err := menu.FirstByUrl(menuUrl); err != nil {
		return nil, err
	}
	// 获取用户信息
	user := model.TUser{}
	if err := user.FirstByUsername(username); err != nil {
		return nil, err
	}
	// 获取用户角色信息
	roleIds, err := (&model.TUserRole{}).PluckRoleIdsByUserId(user.Id)
	if err != nil {
		return nil, err
	}
	// 获取菜单的所有APIid
	apiIds, err := (&model.TMenuApi{}).PluckApiIdsByMenuId(menu.Id)
	if err != nil {
		return nil, err
	}
	// 如果是超级管理员，返回菜单的所有api，如果不是admin则需要做一次过滤
	if !IsAdmin(roleIds) {
		// 获取与用户角色关联的api id列表
		if err := database.DB.Model(&model.TRoleApi{}).Where("role_id in ? and api_id in ?", roleIds, apiIds).Pluck("api_id", &apiIds).Error; err != nil {
			return nil, err
		}
	}

	// 根据api id列表获取api的uris
	uris := make([]string, 0)
	if len(apiIds) > 0 {
		if err := database.DB.Model(&model.TApi{}).Where("id in ?", apiIds).Pluck("uri", &uris).Error; err != nil {
			return nil, fmt.Errorf("根据apiIds<%+v>获取uris异常: <%s>", apiIds, err.Error())
		}
	}
	return uris, nil
}
