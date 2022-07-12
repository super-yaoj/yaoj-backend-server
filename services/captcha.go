package services

import (
	"bytes"

	"github.com/dchest/captcha"
)

type CaptchaPostParam struct {
	Length int `body:"length"`
}

func CaptchaPost(ctx Context, param CaptchaPostParam) {
	// default value
	if param.Length == 0 {
		param.Length = 4
	}
	// len, err := strconv.Atoi(ctx.DefaultPostForm("length", "4"))
	if param.Length < 1 || param.Length > 10 {
		ctx.JSONAPI(400, "invalid request", nil)
		return
	}
	id := captcha.NewLen(param.Length)
	ctx.JSONAPI(200, "", map[string]any{"id": id})
}

type CaptchaGetParam struct {
	ID     string `query:"id"`
	Width  int    `query:"width"`
	Height int    `query:"height"`
}

func CaptchaGet(ctx Context, param CaptchaGetParam) {
	if param.Width == 0 {
		param.Width = 95
	}
	if param.Height == 0 {
		param.Height = 45
	}
	ctx.Header("Cache-Control", "no-cache, no-store, must-revalidate")

	var content bytes.Buffer
	err := captcha.WriteImage(&content, param.ID, param.Width, param.Height)
	if err != nil {
		ctx.JSONAPI(400, err.Error(), nil)
		return
	}
	ctx.Data(200, "image/png", content.Bytes())
}

type CaptchaReloadParam struct {
	ID string `body:"id"`
}

func CaptchaReload(ctx Context, param CaptchaReloadParam) {
	if captcha.Reload(param.ID) {
		ctx.JSONAPI(200, "", nil)
	} else {
		ctx.JSONAPI(400, "id doesn't exist", nil)
	}
}

func VerifyCaptcha(id, num string) bool {
	return captcha.VerifyString(id, num)
}
