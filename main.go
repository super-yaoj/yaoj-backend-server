package main

import (
	"flag"
	"log"
	"os"
	"time"
	"yao/internal"
	"yao/libs"
	"yao/services"

	"github.com/dchest/captcha"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

func process(f func(*gin.Context)) func(*gin.Context) {
	return func(ctx *gin.Context) {
		ctx.Header("Access-Control-Allow-Origin", libs.FrontDomain)
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
	flag.Parse()
	if genConfig {
		data, _ := yaml.Marshal(config)
		os.WriteFile("config.yaml", data, os.ModePerm)
		return
	}
	if configFile != "" {
		data, _ := os.ReadFile(configFile)
		yaml.Unmarshal(data, &config)
	}
	libs.FrontDomain = config.FrontDomain
	libs.BackDomain = config.BackDomain
	libs.DataDir = config.DataDir
	libs.TmpDir = config.TmpDir

	libs.DirInit()
	err := libs.DBInit()
	if err != nil {
		log.Fatal(err)
	}
	app := gin.Default()

	go internal.JudgersInit()
	app.POST("/FinishJudging", internal.FinishJudging)

	app.Use(sessions.Sessions("sessionId", cookie.NewStore([]byte("3.1y4a1o5j9"))))
	app.GET("/judgerlog", process(internal.JudgerLog))
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
	app.Run(config.Listen)
	defer libs.DBClose()
}
