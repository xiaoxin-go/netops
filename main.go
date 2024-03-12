package main

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"netops/conf"
	"netops/database"
	"netops/libs"
	"netops/routers"
	"os"
)

func main() {
	flag.Parse()
	r := gin.New()
	gin.Default()
	log.Println("初始化配置--->")
	conf.InitConfig()

	log.Println("初始化日志--->")
	if e := libs.InitLogger(&conf.Config.Log); e != nil {
		log.Fatalf("初始化日志异常: %s", e.Error())
	}

	// 模板渲染
	if f, err := os.Stat("dist/"); err == nil && f.IsDir() {
		// 模板渲染
		r.LoadHTMLGlob("dist/index.html")
		// 静态目录
		r.StaticFS("/css", http.Dir("dist/css"))
		r.StaticFS("/js", http.Dir("dist/js"))
		r.StaticFS("/img", http.Dir("dist/img"))
	}

	log.Println("注册中间件--->")
	r.Use(libs.GinLogger())
	r.Use(libs.GinRecovery(true))
	r.Use(libs.Cors())
	r.Use(libs.CasbinAuthor())

	r.GET("/", func(request *gin.Context) {
		request.HTML(http.StatusOK, "index.html", gin.H{"title": "netops", "ce": "123456"})
	})

	log.Println("注册路由--->")
	routers.Init(r.Group("/api/v1"))

	log.Println("初始化数据库--->")
	database.InitDB()
	database.InitRedis()

	log.Println("监听端口--->")
	addr := fmt.Sprintf(":%s", conf.Config.Port)
	log.Println(addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf(err.Error())
	}
}
