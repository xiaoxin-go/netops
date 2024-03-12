package model

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"netops/database"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type RedisQuery interface {
	FirstById(id int) error
}

type BaseModel struct {
	Id        int       `gorm:"primary_key" json:"id"`
	CreatedBy string    `gorm:"column:created_by" json:"created_by"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedBy string    `gorm:"column:updated_by" json:"updated_by"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (b *BaseModel) GetId() int {
	return b.Id
}

func (b *BaseModel) SetUpdatedBy(operator string) {
	b.UpdatedBy = operator
}

func (b *BaseModel) SetCreatedBy(operator string) {
	b.CreatedBy = operator
}

type EmptyModel struct {
	Id int `gorm:"primary_key" json:"id"`
}

func (b *EmptyModel) GetId() int {
	return b.Id
}
func (b *EmptyModel) SetUpdatedBy(operator string) {
}

func (b *EmptyModel) SetCreatedBy(operator string) {
}

func toUpper(name string) string {
	nameLetters := strings.Split(name, "_")
	results := make([]string, 0)
	for _, str := range nameLetters {
		results = append(results, strings.ToUpper(str[:1])+str[1:])
	}
	return strings.Join(results, "")
}

// QueryDataByParams 根据设备类型ID获取设备类型信息
// redisKey 存储在redis中的key
// params 数据库的查询语法 key value组合
// fields 查询和获取的字段
// result 结果, model
// expire 存储时间
func QueryDataByParams(redisKey string, params map[string]any, fields []string, result any, expire time.Duration) (err error) {
	data, err := database.R.HMGet(redisKey, fields...).Result()
	if err != nil || data[0] == nil {
		db := database.DB
		for key, value := range params {
			db = db.Where(fmt.Sprintf("%s = ?", key), value)
		}
		err = db.First(result).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err != nil {
			zap.L().Error("获取数据异常",
				zap.Error(err),
				zap.String("model", fmt.Sprintf("%T", result)),
				zap.String("redisKey", redisKey),
				zap.Strings("fields", fields),
				zap.String("params", fmt.Sprintf("%v", params)),
			)
			return err
		}
		fieldMap := toMap(result, fields)
		go func() {
			database.R.HMSet(redisKey, fieldMap)
			database.R.Expire(redisKey, expire)
		}()
		return
	}
	setData(result, fields, data)
	return
}

// 根据ID查询单条记录
func firstById(result any, id int) error {
	return firstByField(result, "id", id)
}

// 根据某个字段查询单条记录
func firstByField(result any, field string, val any) error {
	if e := database.DB.First(result, fmt.Sprintf("%s = ?", field), val).Error; errors.Is(e, gorm.ErrRecordNotFound) {
		return fmt.Errorf("record not found, model: %T, %s: %v", result, field, val)
	} else if e != nil {
		return fmt.Errorf("failed to get record, model: %T, %s: %v, err: %w", result, field, val, e)
	}
	return nil
}

// 根据ID查询单条数据, 并存入redis, 若redis中存在则从redis中获取
func queryById(data RedisQuery, redisKey string, id int, fields []string) error {
	results, err := database.R.HMGet(redisKey, fields...).Result()
	if err != nil || results[0] == nil {
		if e1 := data.FirstById(id); e1 != nil {
			return e1
		}
		database.R.HMSet(redisKey, toMap(data, fields))
		database.R.Expire(redisKey, 2*time.Hour)
	} else {
		setData(data, fields, results)
	}
	return nil
}

// 把对象根据字段转换为map
func toMap(data any, fields []string) map[string]any {
	result := make(map[string]any)
	for _, v := range fields {
		te := reflect.ValueOf(data)
		te = te.Elem()
		fe := te.FieldByName(toUpper(v))
		if fe.IsValid() && fe.CanSet() {
			switch fe.Type().Name() {
			case "int", "int32", "int64":
				result[v] = fe.Int()
			case "uint", "uint32", "uint64":
				result[v] = fe.Uint()
			case "string":
				result[v] = fe.String()
			case "Time":
				result[v] = fe.Interface().(time.Time).Format(time.DateTime)
			}
		}
	}
	return result
}

// 从redis hmget取到的数据，根据字段设置到数据上
func setData(data any, fields []string, results []any) {
	for i, v := range fields {
		if results[i] == nil {
			continue
		}
		te := reflect.ValueOf(data)
		te = te.Elem()
		fe := te.FieldByName(toUpper(v))
		if fe.IsValid() && fe.CanSet() {
			switch fe.Type().Name() {
			case "int", "int32", "int64":
				r, _ := strconv.Atoi(results[i].(string))
				fe.SetInt(int64(r))
			case "uint", "uint32", "uint64":
				r, _ := strconv.Atoi(results[i].(string))
				fe.SetUint(uint64(r))
			case "string":
				fe.SetString(results[i].(string))
			case "Time":
				t, _ := time.Parse(time.DateTime, results[i].(string))
				fe.Set(reflect.ValueOf(t))
			}
		}
	}
}
