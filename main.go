package main

import (
	"flag"
	"log"
	"os"
	"time"
	config "yao/config"
	"yao/db"
	"yao/internal"
	"yao/service"
	"yao/services"

	"github.com/dchest/captcha"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// func process(f func(*gin.Context)) func(*gin.Context) {
// 	return func(ctx *gin.Context) {
// 		ctx.Header("Access-Control-Allow-Origin", config.Global.FrontDomain)
// 		ctx.Header("Access-Control-Allow-Credentials", "true")
// 		if ctx.Request.Method == "OPTIONS" {
// 			ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
// 			ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
// 			ctx.Status(204)
// 			return
// 		}
// 		sessions.Default(ctx).Options(sessions.Options{MaxAge: 0})
// 		f(ctx)
// 	}
// }

func main() {
	// parse config
	flag.Parse()
	if config.GenConfig() {
		data, _ := yaml.Marshal(config.Global)
		os.WriteFile("config.yaml", data, os.ModePerm)
		return
	}
	if config.ConfigFile() != "" {
		data, _ := os.ReadFile(config.ConfigFile())
		yaml.Unmarshal(data, &config.Global)
	}

	// init
	os.MkdirAll(config.Global.DataDir, os.ModePerm)
	dsn := config.Global.DataSource
	err := db.Init(dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	go internal.JudgersInit()
	captcha.SetCustomStore(captcha.NewMemoryStore(1024, 10*time.Minute))

	// server init
	db, err := db.NewDBR("mysql", config.Global.DataSource)
	if err != nil {
		log.Fatal(err)
	}
	app := service.NewServer(db)
	app.Use(sessions.Sessions("sessionId", cookie.NewStore([]byte("3.1y4a1o5j9"))))
	// FinishJuding rpc dooesn't need process function
	app.POST("/FinishJudging", services.FinishJudging)

	app.Use(func(ctx *gin.Context) {
		ctx.Header("Access-Control-Allow-Origin", config.Global.FrontDomain)
		ctx.Header("Access-Control-Allow-Credentials", "true")

		if ctx.Request.Method != "OPTIONS" {
			sessions.Default(ctx).Options(sessions.Options{MaxAge: 0})
		}

		ctx.Next()
	})

	for url, value := range services.Router {
		app.RestApi(url, value)
	}
	app.Run(config.Global.Listen)
}
