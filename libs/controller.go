package libs

import (
	"errors"
	"fmt"
	"net/http"
	"netops/conf"
	"netops/database"
	"netops/model"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Restfuller interface {
	Get(c *gin.Context)
	List(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
}

type Instance interface {
	GetId() int
	SetUpdatedBy(string)
	SetCreatedBy(string)
}

type Controller struct {
	NewResults  func() any
	NewInstance func() Instance
	OrderFilter func(db *gorm.DB) *gorm.DB
	QueryFilter func(db *gorm.DB) *gorm.DB
}

func AdvancedQuery(ctx *gin.Context) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		for k, v := range ctx.Request.URL.Query() {
			// 排除page size
			if k == "page" || k == "size" {
				continue
			}
			if !strings.Contains(k, "__") {
				db = db.Where(fmt.Sprintf("%s = ?", k), v[0])
				continue
			}
			// name__contains  取出name和contains
			ks := strings.Split(k, "__")
			k1 := ks[0]
			kType := ks[1]

			// ?name=lisi&age__gt=18&id__in=1,2,3
			switch kType {
			case "eq":
				db = db.Where(fmt.Sprintf("%s = ?", k1), v[0])
			case "neq":
				db = db.Where(fmt.Sprintf("%s != ?", k1), v[0])
			case "contains":
				db = db.Where(fmt.Sprintf("%s like ?", k1), "%"+v[0]+"%")
			case "in":
				db = db.Where(fmt.Sprintf("%s in ?", k1), strings.Split(v[0], ","))
			case "not_in":
				db = db.Where(fmt.Sprintf("%s not in ?", k1), strings.Split(v[0], ","))
			case "gte":
				db = db.Where(fmt.Sprintf("%s >= ?", k1), v[0])
			case "lte":
				db = db.Where(fmt.Sprintf("%s <= ?", k1), v[0])
			case "gt":
				db = db.Where(fmt.Sprintf("%s > ?", k1), v[0])
			case "lt":
				db = db.Where(fmt.Sprintf("%s < ?", k1), v[0])
			}
		}
		return db
	}
}

func Pagination(page, size int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}
		if size <= 0 {
			size = 20
		}
		db.Offset((page - 1) * size).Limit(size)
		return db
	}
}

func (c *Controller) orderBy(db *gorm.DB) *gorm.DB {
	if c.OrderFilter != nil {
		return c.OrderFilter(db)
	}
	return db.Order("-id")
}

func (c *Controller) QuerySet(ctx *gin.Context) (db *gorm.DB) {
	m := c.NewInstance()
	db = database.DB.Model(m).Scopes(AdvancedQuery(ctx))
	if c.QueryFilter != nil {
		db.Scopes(c.QueryFilter)
	}
	return
}
func (c *Controller) Count(db *gorm.DB, count *int64) *gorm.DB {
	return db.Count(count)
}
func (c *Controller) OrderBy(db *gorm.DB) *gorm.DB {
	return c.orderBy(db)
}
func (c *Controller) Pagination(ctx *gin.Context) func(db *gorm.DB) *gorm.DB {
	page, size := c.GetPagination(ctx)
	return Pagination(page, size)
}
func (c *Controller) Response(ctx *gin.Context, results interface{}, total int64, err error) {
	if err != nil {
		HttpServerError(ctx, err.Error())
		return
	}
	data := map[string]interface{}{
		"data":  results,
		"total": total,
	}
	ctx.JSON(http.StatusOK, Success(data, "ok"))
}
func (c *Controller) QueryListData(ctx *gin.Context) (results any, total int64, err error) {
	db := c.QuerySet(ctx).Count(&total).Scopes(c.orderBy, c.Pagination(ctx))
	results = c.NewResults()
	if e := db.Find(results).Error; e != nil {
		err = fmt.Errorf("查询数据异常, err: %w", e)
		zap.L().Error("查询数据异常", zap.Error(e))
		return
	}
	return
}
func (c *Controller) List(ctx *gin.Context) {
	results, total, err := c.QueryListData(ctx)
	if err != nil {
		HttpServerError(ctx, err.Error())
		return
	}
	HttpListSuccess(ctx, results, total)
}
func (c *Controller) Get(ctx *gin.Context) {
	id, e := c.GetId(ctx)
	if e != nil {
		HttpParamsError(ctx, e.Error())
		return
	}
	m := c.NewInstance()
	if err := database.DB.Where("id = ?", id).First(m).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		HttpParamsError(ctx, fmt.Sprintf("根据ID<%d>未获取到数据", id))
		return
	} else if err != nil {
		HttpServerError(ctx, fmt.Sprintf("获取数据异常, id: %d, err: %s", id, err.Error()))
		return
	}
	HttpSuccess(ctx, m, "ok")
}
func (c *Controller) Create(ctx *gin.Context) {
	traceId, _ := ctx.Get("TraceId")
	m := c.NewInstance()
	l := zap.L().With(zap.String("TraceId", traceId.(string)), zap.String("model", fmt.Sprintf("%T", m)))

	err := ctx.ShouldBindJSON(m)
	if err != nil {
		HttpParamsError(ctx, fmt.Sprintf("参数解析异常: <%s>", err.Error()))
		return
	}

	c.setOperator(ctx, m, "created_by")

	if err := database.DB.Save(m).Error; err != nil && strings.Contains(err.Error(), "Duplicate entry") {
		HttpParamsError(ctx, "添加失败：数据已存在")
		return
	} else if err != nil {
		l.Error("添加数据失败", zap.Error(err))
		HttpServerError(ctx, fmt.Sprintf("添加数据异常: <%s>", err.Error()))
		return
	}
	l.Info("插入数据", zap.Any("data", m))
	AddLog(ctx, "插入数据, data: %+v", m)
	HttpSuccess(ctx, m, "ok")
}
func (c *Controller) Update(ctx *gin.Context) {
	traceId, _ := ctx.Get("TraceId")
	params := c.NewInstance()
	l := zap.L().With(zap.String("TraceId", traceId.(string)), zap.String("model", fmt.Sprintf("%T", params)))
	if e := ctx.ShouldBindJSON(params); e != nil {
		HttpParamsError(ctx, "参数解析失败, err: %s", e.Error())
		return
	}
	id := params.GetId()
	oldM := c.NewInstance()
	if e := database.DB.First(oldM, id).Error; e != nil {
		HttpServerError(ctx, "获取数据失败, err: %s", e.Error())
		return
	}

	// 设置操作人
	user, e := GetUser(ctx)
	if e != nil {
		HttpAuthorError(ctx, e.Error())
		return
	}
	params.SetUpdatedBy(user.Username)
	if e := database.DB.Model(params).Where("id = ?", id).Updates(params).Error; e != nil {
		l.Error("更新数据异常", zap.Error(e))
		HttpServerError(ctx, "更新数据异常, err: %s", e.Error())
		return
	}

	l.Info("修改数据", zap.Any("before", oldM), zap.Any("after", params))
	fmt.Printf("更新数据\n, before: %+v\n, after: %+v\n, params: %+v\n", oldM, params, params)
	AddLog(ctx, "修改数据, before: %+v, after: %+v", oldM, params)
	HttpSuccess(ctx, params, "ok")
}
func (c *Controller) Delete(ctx *gin.Context) {
	id, e := c.GetId(ctx)
	if e != nil {
		HttpParamsError(ctx, e.Error())
		return
	}
	traceId, _ := ctx.Get("TraceId")
	m := c.NewInstance()
	l := zap.L().With(zap.String("TraceId", traceId.(string)), zap.String("model", fmt.Sprintf("%T", m)))
	if err := database.DB.First(m, id).Error; err != nil {
		l.Error("获取数据异常", zap.Error(err))
		HttpServerError(ctx, "获取数据异常, err: %s", err.Error())
		return
	}
	if c.softDelete(m) {
		if err := database.DB.Model(m).Where("id = ?", id).Update("is_deleted", 1).Error; err != nil {
			l.Error("软删除数据异常", zap.Error(err))
			HttpServerError(ctx, "软删除数据异常, err: %s", err.Error())
			return
		}
	} else {
		if err := database.DB.Delete(m).Error; err != nil {
			l.Error("删除数据异常", zap.Error(err))
			HttpServerError(ctx, "删除数据异常, err: %s", err.Error())
			return
		}
	}
	l.Info("删除数据", zap.Any("data", m))
	AddLog(ctx, "删除数据, data: %+v", m)
	HttpSuccess(ctx, m, "ok")
}
func (c *Controller) GetId(ctx *gin.Context) (int, error) {
	idStr := ctx.Query("id")
	fmt.Println("id----->", idStr)
	id, e := strconv.Atoi(idStr)
	if e != nil {
		return 0, errors.New("id不能是字符串")
	}
	if id <= 0 {
		return 0, errors.New("ID必须大于0")
	}
	return id, nil
}
func (c *Controller) BatchDelete(ctx *gin.Context) {
	params := struct {
		DataIds []int `json:"data_ids"`
	}{}
	if err := ctx.ShouldBindJSON(&params); err != nil {
		HttpParamsError(ctx, fmt.Sprintf("读取请求参数异常: <%s>", err.Error()))
		return
	}
	m := c.NewInstance()
	l := zap.L().With(zap.String("func", "BatchDelete"))
	l.Debug("批量删除数据------------->")
	if c.softDelete(m) {
		if err := database.DB.Model(m).Where("id in ?", params.DataIds).Update("is_deleted", 1).Error; err != nil {
			HttpServerError(ctx, fmt.Sprintf("批量删除数据异常: <%s>", err.Error()))
			return
		}
	} else {
		if err := database.DB.Where("id in ?", params.DataIds).Delete(m).Error; err != nil {
			HttpServerError(ctx, fmt.Sprintf("批量删除数据异常: <%s>", err.Error()))
			return
		}
	}
	AddLog(ctx, fmt.Sprintf("ids: <%+v>, model: <%T>", params.DataIds, m))
	HttpSuccess(ctx, nil, "删除成功")
}
func AddLog(ctx *gin.Context, format string, a ...any) {
	operator, ok := ctx.Get("Operator")
	if ok {
		model.AddLog(operator.(string), format, a...)
	} else {
		model.AddLog("unknown", format, a...)
		zap.L().Warn("添加操作日志未获取到用户信息")
	}
}

// 设置操作人
func (c *Controller) setOperator(ctx *gin.Context, m any, fieldName string) {
	te := reflect.ValueOf(m)
	te = te.Elem()
	fe := te.FieldByName(fieldName)
	if fe.IsValid() {
		operator, _ := ctx.Get("Operator")
		fe.SetString(operator.(string))
	}
}

// 如果数据有is_delete则较删除
func (c *Controller) softDelete(m any) bool {
	te := reflect.ValueOf(m)
	te = te.Elem()
	fe := te.FieldByName("IsDeleted")
	if fe.IsValid() {
		return true
	}
	return false
}

// GetPagination 获取分页内容
func (c *Controller) GetPagination(ctx *gin.Context) (page, pageSize int) {
	pageStr := ctx.DefaultQuery("page", "1")
	pageSizeStr := ctx.DefaultQuery("size", "20")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = 1
	}
	pageSize, err = strconv.Atoi(pageSizeStr)
	if err != nil {
		pageSize = 20
	}
	return
}
func GetUser(ctx *gin.Context) (*model.TUser, error) {
	user := &model.TUser{}
	if conf.Config.Env == "test" {
		user.Id = 1
	} else {
		sessionId, err := ctx.Cookie(conf.SessionKey)
		fmt.Println("sessionId------------->:", sessionId)
		if err != nil {
			return nil, errors.New("认证过期")
		}

		userId, e := database.R.HGet(sessionId, "user_id").Int()
		if e != nil {
			return nil, errors.New("无效的session")
		}
		if userId == 0 {
			return nil, errors.New("用户session过期")
		}
		user.Id = userId
	}
	if e := user.FirstById(user.Id); e != nil {
		return nil, e
	}
	return user, nil
}
func (c *Controller) StopRun() {
	panic("stop run")
}
