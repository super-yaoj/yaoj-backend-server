package libs

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func SetCookie(ctx *gin.Context, key, value string, security bool) {
	ctx.SetCookie(key, value, Year, "/", FrontDomain, security, false)
}

func DeleteCookie(ctx *gin.Context, key string) {
	ctx.SetCookie(key, "", -1, "/", FrontDomain, false, false)
}

func RPCWriteBack(ctx *gin.Context, statusCode int, errorCode int, errorMessage string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	data["_error"] = map[string]any{"code": errorCode, "message": errorMessage}
	ctx.JSON(statusCode, data)
}

func APIWriteBack(ctx *gin.Context, statusCode int, errorMessage string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	data["_error"] = errorMessage
	ctx.JSON(statusCode, data)
}

func APIInternalError(ctx *gin.Context, err error) {
	ctx.JSON(500, map[string]any{"_error": err.Error()})
}

func RPCInternalError(ctx *gin.Context, err error) {
	ctx.JSON(500, map[string]any{"_error": map[string]any{"code": -32603, "message": err.Error()}})
}

func GetInt(ctx *gin.Context, name string) (int, bool) {
	ret, err := strconv.Atoi(ctx.Query(name))
	if err != nil {
		APIWriteBack(ctx, 400, "invalid request: parameter "+name+" should be int", nil)
		return 0, false
	}
	return ret, true
}

func GetIntRange(ctx *gin.Context, name string, l, r int) (int, bool) {
	ret, ok := GetInt(ctx, name)
	if !ok {
		return 0, false
	}
	if ret > r || ret < l {
		APIWriteBack(ctx, 400, fmt.Sprintf("invalid request: parameter %s should be in [%d, %d]", name, l, r), nil)
		return 0, false
	}
	return ret, true
}

func GetIntDefault(ctx *gin.Context, name string, d int) int {
	ret, err := strconv.Atoi(ctx.Query(name))
	if err != nil {
		return d
	}
	return ret
}

func PostInt(ctx *gin.Context, name string) (int, bool) {
	ret, err := strconv.Atoi(ctx.PostForm(name))
	if err != nil {
		APIWriteBack(ctx, 400, "invalid request: parameter "+name+" should be int", nil)
		return 0, false
	}
	return ret, true
}

func PostIntRange(ctx *gin.Context, name string, l, r int) (int, bool) {
	ret, ok := PostInt(ctx, name)
	if !ok {
		return 0, false
	}
	if ret > r || ret < l {
		APIWriteBack(ctx, 400, fmt.Sprintf("invalid request: parameter %s should be in [%d, %d]", name, l, r), nil)
		return 0, false
	}
	return ret, true
}

func PostIntDefault(ctx *gin.Context, name string, d int) int {
	ret, err := strconv.Atoi(ctx.PostForm(name))
	if err != nil {
		return d
	}
	return ret
}

func GetQuerys(query map[string]string) string {
	first := true
	ret := strings.Builder{}
	for key, val := range query {
		if !first {
			ret.WriteString("&")
		} else {
			first = false
		}
		ret.WriteString(key + "=" + url.QueryEscape(val))
	}
	return ret.String()
}