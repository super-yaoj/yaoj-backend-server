package services

import (
	"io/ioutil"
	"net/http"
	"yao/internal"

	"github.com/gin-gonic/gin"
	"github.com/super-yaoj/yaoj-utils/promise"
)

func FinishJudging(ctx *gin.Context) {
	var result []byte
	var err error
	promise.NewErrorPromise(func () error {
		result, err = ioutil.ReadAll(ctx.Request.Body)
		return err
	}).Then(func() error {
		return internal.FinishJudging(ctx.Query("jid"), result)
	}).Catch(func(err error) {
		Context{Context: ctx}.ErrorRPC(err)
	})
}

type JudgerLogParam struct {
	Id int `query:"id"`
}

func JudgerLog(ctx Context, param JudgerLogParam) {
	log := internal.JudgerLog(param.Id)
	ctx.JSONAPI(http.StatusOK, "", map[string]any{"log": log})
}