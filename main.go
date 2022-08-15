package main

import (
	"flag"
	"log"
	"os"
	"time"
	config "yao/config"
	"yao/db"
	"yao/internal"
	"yao/services"

	"github.com/dchest/captcha"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

func process(f func(*gin.Context)) func(*gin.Context) {
	return func(ctx *gin.Context) {
		ctx.Header("Access-Control-Allow-Origin", config.Global.FrontDomain)
		ctx.Header("Access-Control-Allow-Credentials", "true")
		if ctx.Request.Method == "OPTIONS" {
			ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
			ctx.Status(204)
			return
		}
		sessions.Default(ctx).Options(sessions.Options{MaxAge: 0})
		f(ctx)
	}
}

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
	err := db.DBInit()
	if err != nil {
		log.Fatal(err)
	}
	defer db.DBClose()
	go internal.JudgersInit()

	// server init
	app := gin.Default()
	app.Use(sessions.Sessions("sessionId", cookie.NewStore([]byte("3.1y4a1o5j9"))))
	//FinishJuding rpc dooesn't need process function
	app.POST("/FinishJudging", services.FinishJudging)
	captcha.SetCustomStore(captcha.NewMemoryStore(1024, 10*time.Minute))
	for url, value := range services.Router {
		app.OPTIONS(url, process(func(ctx *gin.Context) {}))
		for _, req := range value {
			switch req.Method {
			case "GET":
				app.GET(url, process(req.Function))
			case "POST":
				app.POST(url, process(req.Function))
			case "PATCH":
				app.PATCH(url, process(req.Function))
			case "PUT":
				app.PUT(url, process(req.Function))
			case "DELETE":
				app.DELETE(url, process(req.Function))
			}
		}
	}
	app.Run(config.Global.Listen)
}
