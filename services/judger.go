package services

import (
	"io/ioutil"
	"net/http"
	"yao/internal"
)

type FinishJudingParam struct {
	JudgerId string `query:"jid" validate:"required"`
}

func FinishJudging(ctx Context, param FinishJudingParam) {
	result, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.ErrorRPC(err)
	}
	err = internal.FinishJudging(param.JudgerId, result)
}

type JudgerLogParam struct {
	Id int `query:"id"`
}

func JudgerLog(ctx Context, param JudgerLogParam) {
	log := internal.JudgerLog(param.Id)
	ctx.JSONAPI(http.StatusOK, "", map[string]any{"log": log})
}