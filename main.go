package main

import (
	"log"
	"time"
	"yao/components"
	"yao/controllers"
	"yao/libs"

	"github.com/dchest/captcha"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
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
	err := libs.DBInit()
	if err != nil {
		log.Fatal(err)
	}
	app := gin.Default()
	
	go controllers.JudgersInit()
	app.POST("/FinishJudging", controllers.FinishJudging)
	
	app.Use(sessions.Sessions("sessionId", cookie.NewStore([]byte("3.1y4a1o5j9"))))
	captcha.SetCustomStore(captcha.NewMemoryStore(1024, 10 * time.Minute))
	for url, value := range components.Router {
		app.OPTIONS(url, process(func(ctx *gin.Context) {}))
		for _, req := range value {
			switch req.Method {
			case "GET": 	app.GET(url, process(req.Function))
			case "POST": 	app.POST(url, process(req.Function))
			case "PATCH": 	app.PATCH(url, process(req.Function))
			case "PUT": 	app.PUT(url, process(req.Function))
			case "DELETE": 	app.DELETE(url, process(req.Function))
			}
		}
	}
	app.Run("0.0.0.0:8081")
	defer libs.DBClose()
}
