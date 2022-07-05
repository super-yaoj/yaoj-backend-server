package components

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"yao/controllers"
	"yao/libs"

	"github.com/gin-gonic/gin"
)

func PMCreate(ctx *gin.Context) {
	if !ISAdmin(ctx) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	name := ctx.PostForm("permission_name")
	if len(name) > 190 {
		libs.APIWriteBack(ctx, 400, "permission name is too long", nil)
		return
	}
	id, err := controllers.PMCreate(name)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{ "permission_id": id })
	}
}

func PMChangeName(ctx *gin.Context) {
	if !ISAdmin(ctx) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	id, ok := libs.PostInt(ctx, "permission_id")
	if !ok { return }
	name := ctx.PostForm("permission_name")
	if len(name) > 190 {
		libs.APIWriteBack(ctx, 400, "permission name is too long", nil)
		return
	}
	if !controllers.PMExists(id) {
		libs.APIWriteBack(ctx, 400, "no such permission id", nil)
		return
	}
	err := controllers.PMChangeName(id, name)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

func PMDelete(ctx *gin.Context) {
	if !ISAdmin(ctx) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	id, ok := libs.GetInt(ctx, "permission_id")
	if !ok { return }
	if id == libs.DefaultGroup {
		libs.APIWriteBack(ctx, 400, "you cannot modify the default group", nil)
		return
	}
	num, err := controllers.PMDelete(id)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else if num != 1 {
		libs.APIWriteBack(ctx, 400, "no such permission id", nil)
	} else {
		libs.APIWriteBack(ctx, 200, "", nil)
	}
}

func PMQuery(ctx *gin.Context) {
	if !ISAdmin(ctx) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	pagesize, ok := libs.GetIntRange(ctx, "pagesize", 1, 100)
	if !ok { return }
	_, isleft := ctx.GetQuery("left")
	bound, ok := libs.GetInt(ctx, libs.If(isleft, "left", "right"))
	if !ok { return }
	p, isfull, err := controllers.PMQuery(bound, pagesize, isleft)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{ "isfull": isfull, "data": p })
	}
}

func PMQueryUser(ctx *gin.Context) {
	_, ok := ctx.GetQuery("permission_id")
	if ok {
		if !ISAdmin(ctx) {
			libs.APIWriteBack(ctx, 403, "", nil)
			return
		}
		id, ok := libs.GetInt(ctx, "permission_id")
		if !ok { return }
		users, err := controllers.PMQueryUser(id)
		if err != nil {
			libs.APIInternalError(ctx, err)
		} else {
			libs.APIWriteBack(ctx, 200, "", map[string]any{ "data": users })
		}
	} else {
		id, ok := libs.GetInt(ctx, "user_id")
		if !ok { return }
		if !ISAdmin(ctx) && id != GetUserId(ctx) {
			libs.APIWriteBack(ctx, 403, "", nil)
			return
		}
		permissions, err := controllers.USQueryPermission(id)
		if err != nil {
			libs.APIInternalError(ctx, err)
		} else {
			libs.APIWriteBack(ctx, 200, "", map[string]any{ "data": permissions })
		}
	}
}

func PMAddUser(ctx *gin.Context) {
	if !ISAdmin(ctx) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	id, ok := libs.PostInt(ctx, "permission_id")
	if !ok { return }
	str := ctx.PostForm("user_ids")
	user_ids := strings.Split(str, ",")
	if len(user_ids) == 0 {
		libs.APIWriteBack(ctx, 400, "there's no user", nil)
		return
	}
	if id == libs.DefaultGroup {
		libs.APIWriteBack(ctx, 400, "you cannot modify the default group", nil)
		return
	}
	if !controllers.PMExists(id) {
		libs.APIWriteBack(ctx, 400, "invalid request: permission id is wrong", nil)
		return
	}
	
	var err error
	num_ids := make([]int, len(user_ids))
	for i, j := range user_ids {
		num_ids[i], err = strconv.Atoi(j)
		if err != nil {
			libs.APIWriteBack(ctx, 400, "invalid request: meet user id \"" + j + "\"", nil)
			return
		}
	}
	real_ids, err := libs.DBSelectInts(fmt.Sprintf("select user_id from user_info where user_id in (%s) order by user_id", str))
	if err != nil {
		libs.APIInternalError(ctx, err)
		return
	}
	var invalid_ids []int
	for _, i := range num_ids {
		j := sort.SearchInts(real_ids, i)
		if j == len(real_ids) || real_ids[j] != i {
			invalid_ids = append(invalid_ids, i)
		}
	}
	if len(invalid_ids) != 0 {
		libs.APIWriteBack(ctx, 400, "invalid ids exist", map[string]any{ "invalid_ids": invalid_ids })
		return
	}
	res, err := controllers.PMAddUser(real_ids, id)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", map[string]any{ "affected": res })
	}
}

func PMDeleteUser(ctx *gin.Context) {
	if !ISAdmin(ctx) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	pid, ok := libs.GetInt(ctx, "permission_id")
	uid, ok1 := libs.GetInt(ctx, "user_id")
	if !ok || !ok1 { return }
	if pid == libs.DefaultGroup {
		libs.APIWriteBack(ctx, 400, "you cannot modify the default group", nil)
		return
	}
	res, err := controllers.PMDeleteUser(pid, uid)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else if res != 1 {
		libs.APIWriteBack(ctx, 400, "doesn't exist", nil)
	}
}