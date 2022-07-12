package services

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"yao/internal"
	"yao/libs"

	"github.com/gin-gonic/gin"
)

type PermCreateParam struct {
	PermName string `body:"permission_name"`
	UserGrp  int    `session:"user_group" validate:"admin"`
}

func PermCreate(ctx Context, param PermCreateParam) {
	if len(param.PermName) > 190 {
		ctx.JSONAPI(400, "permission name is too long", nil)
		return
	}
	id, err := internal.PMCreate(param.PermName)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(200, "", gin.H{"permission_id": id})
	}
}

type PermRenameParam struct {
	PermID   int    `body:"permission_id" binding:"required"`
	PermName string `body:"permission_name"`
	UserGrp  int    `session:"user_group" validate:"admin"`
}

func PermRename(ctx Context, param PermRenameParam) {
	if len(param.PermName) > 190 {
		ctx.JSONAPI(400, "permission name is too long", nil)
		return
	}
	if !internal.PMExists(param.PermID) {
		ctx.JSONAPI(400, "no such permission id", nil)
		return
	}
	err := internal.PMChangeName(param.PermID, param.PermName)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type PermDelParam struct {
	PermID  int `query:"permission_id" binding:"required"`
	UserGrp int `session:"user_group" validate:"admin"`
}

func PermDel(ctx Context, param PermDelParam) {
	if param.PermID == libs.DefaultGroup {
		ctx.JSONAPI(400, "you cannot modify the default group", nil)
		return
	}
	num, err := internal.PMDelete(param.PermID)
	if err != nil {
		ctx.ErrorAPI(err)
	} else if num != 1 {
		ctx.JSONAPI(400, "no such permission id", nil)
	} else {
		ctx.JSONAPI(200, "", nil)
	}
}

type PermGetParam struct {
	PageSize int  `query:"pagesize" binding:"required" validate:"gte=1,lte=100"`
	Left     *int `query:"left"`
	Right    *int `query:"right"`
	UserGrp  int  `session:"user_group" validate:"admin"`
}

func PermGet(ctx Context, param PermGetParam) {
	var bound int
	if param.Left != nil {
		bound = *param.Left
	} else if param.Right != nil {
		bound = *param.Right
	} else {
		return
	}
	p, isfull, err := internal.PMQuery(bound, param.PageSize, param.Left != nil)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(200, "", gin.H{"isfull": isfull, "data": p})
	}
}

type PermGetUserParam struct {
	PermID    *int `query:"permission_id"`
	UserID    int  `query:"user_id"`
	CurUserID int  `session:"user_id"`
	UserGrp   int  `session:"user_group"`
}

func PermGetUser(ctx Context, param PermGetUserParam) {
	if param.PermID != nil {
		if !libs.IsAdmin(param.UserGrp) {
			ctx.JSONAPI(403, "", nil)
			return
		}
		users, err := internal.PMQueryUser(*param.PermID)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(200, "", map[string]any{"data": users})
		}
	} else {
		if !libs.IsAdmin(param.UserGrp) && param.UserID != param.CurUserID {
			ctx.JSONAPI(403, "", nil)
			return
		}
		permissions, err := internal.USQueryPermission(param.UserID)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(200, "", map[string]any{"data": permissions})
		}
	}
}

type PermAddUserParam struct {
	PermID  int    `body:"permission_id" binding:"required"`
	UserIDs string `body:"user_ids"`
	UserGrp int    `session:"user_group" validate:"admin"`
}

func PermAddUser(ctx Context, param PermAddUserParam) {
	user_ids := strings.Split(param.UserIDs, ",")
	if len(user_ids) == 0 {
		ctx.JSONAPI(400, "there's no user", nil)
		return
	}
	if param.PermID == libs.DefaultGroup {
		ctx.JSONAPI(400, "you cannot modify the default group", nil)
		return
	}
	if !internal.PMExists(param.PermID) {
		ctx.JSONAPI(400, "invalid request: permission id is wrong", nil)
		return
	}

	var err error
	num_ids := make([]int, len(user_ids))
	for i, j := range user_ids {
		num_ids[i], err = strconv.Atoi(j)
		if err != nil {
			ctx.JSONAPI(400, "invalid request: meet user id \""+j+"\"", nil)
			return
		}
	}
	real_ids, err := libs.DBSelectInts(fmt.Sprintf("select user_id from user_info where user_id in (%s) order by user_id", param.UserIDs))
	if err != nil {
		ctx.ErrorAPI(err)
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
		ctx.JSONAPI(400, "invalid ids exist", gin.H{"invalid_ids": invalid_ids})
		return
	}
	res, err := internal.PMAddUser(real_ids, param.PermID)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(200, "", gin.H{"affected": res})
	}
}

type PermDelUserParam struct {
	PermID  int `query:"permission_id" binding:"required"`
	UserID  int `query:"user_id" binding:"required"`
	UserGrp int `session:"user_group" validate:"admin"`
}

func PermDelUser(ctx Context, param PermDelUserParam) {
	if param.PermID == libs.DefaultGroup {
		ctx.JSONAPI(400, "you cannot modify the default group", nil)
		return
	}
	res, err := internal.PMDeleteUser(param.PermID, param.UserID)
	if err != nil {
		ctx.ErrorAPI(err)
	} else if res != 1 {
		ctx.JSONAPI(400, "doesn't exist", nil)
	}
}
