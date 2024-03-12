package database

import (
	"fmt"
	"github.com/go-redis/redis"
	"netops/conf"
)

var R *redis.Client

func InitRedis() {
	redisAddr := fmt.Sprintf("%s:%s", conf.Config.Redis.Host, conf.Config.Redis.Port)
	R = redis.NewClient(&redis.Options{
		Addr:     redisAddr,                  // redis地址
		Password: conf.Config.Redis.Password, // redis密码，没有则留空
		DB:       conf.Config.Redis.DB,       // 默认数据库，默认是0
	})

	//通过 *redis.Client.Ping() 来检查是否成功连接到了redis服务器
	_, err := R.Ping().Result()
	if err != nil {
		panic(err)
	}
}
