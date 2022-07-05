package components

import (
	"bytes"
	"strconv"
	"yao/libs"

	"github.com/dchest/captcha"
	"github.com/gin-gonic/gin"
)

func CaptchaId(ctx *gin.Context) {
	len, err := strconv.Atoi(ctx.DefaultPostForm("length", "4"))
	if err != nil || len < 1 || len > 10 {
		libs.APIWriteBack(ctx, 400, "invalid request", nil)
		return
	}
	id := captcha.NewLen(len)
	libs.APIWriteBack(ctx, 200, "", map[string]any{"id": id})
}

func CaptchaImage(ctx *gin.Context) {
	id := ctx.Query("id")
	width, err := strconv.Atoi(ctx.DefaultQuery("width", "95"))
	height, err1 := strconv.Atoi(ctx.DefaultQuery("height", "45"))
	if err != nil || err1 != nil {
		libs.APIWriteBack(ctx, 400, "invalid request", nil)
		return
	}
	ctx.Header("Cache-Control", "no-cache, no-store, must-revalidate")

	var content bytes.Buffer
	err = captcha.WriteImage(&content, id, width, height)
	if err != nil {
		libs.APIWriteBack(ctx, 400, err.Error(), nil)
		return
	}
	ctx.Data(200, "image/png", content.Bytes())
}

func ReloadCaptchaImage(ctx *gin.Context) {
	id := ctx.PostForm("id")
	if captcha.Reload(id) {
		libs.APIWriteBack(ctx, 200, "", nil)
	} else {
		libs.APIWriteBack(ctx, 400, "id doesn't exist", nil)
	}
}

func VerifyCaptcha(id, num string) bool {
	return captcha.VerifyString(id, num)
}
