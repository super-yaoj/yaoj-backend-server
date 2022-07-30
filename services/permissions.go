package services

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"yao/internal"
	"yao/libs"

	"github.com/gin-gonic/gin"
)

type PermCreateParam struct {
	PermName string `body:"permission_name" validate:"lte=190"`
	UserGrp  int    `session:"user_group" validate:"admin"`
}

func PermCreate(ctx Context, param PermCreateParam) {
	id, err := internal.PMCreate(param.PermName)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(http.StatusOK, "", gin.H{"permission_id": id})
	}
}

type PermRenameParam struct {
	PermID   int    `body:"permission_id" binding:"required" validate:"prmsid"`
	PermName string `body:"permission_name" validate:"lte=190"`
	UserGrp  int    `session:"user_group" validate:"admin"`
}

func PermRename(ctx Context, param PermRenameParam) {
	err := internal.PMChangeName(param.PermID, param.PermName)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type PermDelParam struct {
	PermID  int `query:"permission_id" binding:"required" validate:"prmsid"`
	UserGrp int `session:"user_group" validate:"admin"`
}

func PermDel(ctx Context, param PermDelParam) {
	if param.PermID == libs.DefaultGroup {
		ctx.JSONAPI(http.StatusBadRequest, "you cannot modify the default group", nil)
		return
	}
	_, err := internal.PMDelete(param.PermID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}

type PermGetParam struct {
	Page
	UserGrp int `session:"user_group" validate:"admin"`
}

func PermGet(ctx Context, param PermGetParam) {
	if !param.CanBound() {
		return
	}
	p, isfull, err := internal.PMQuery(param.Bound(), param.PageSize, param.IsLeft())
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(http.StatusOK, "", gin.H{"isfull": isfull, "data": p})
	}
}

type PermGetUserParam struct {
	PermID     *int `query:"permission_id" validate:"prmsid"`
	PermUserID *int `query:"user_id" validate:"userid"`
	Auth
}

func PermGetUser(ctx Context, param PermGetUserParam) {
	if param.PermID != nil {
		if !param.IsAdmin() {
			ctx.JSONAPI(http.StatusForbidden, "", nil)
			return
		}
		users, err := internal.PMQueryUser(*param.PermID)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": users})
		}
	} else {
		if !param.IsAdmin() && *param.PermUserID != param.UserID {
			ctx.JSONAPI(http.StatusForbidden, "", nil)
			return
		}
		permissions, err := internal.USQueryPermission(*param.PermUserID)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": permissions})
		}
	}
}

type PermAddUserParam struct {
	PermID  int    `body:"permission_id" binding:"required" validate:"prmsid"`
	UserIDs string `body:"user_ids"`
	UserGrp int    `session:"user_group" validate:"admin"`
}

func PermAddUser(ctx Context, param PermAddUserParam) {
	user_ids := strings.Split(param.UserIDs, ",")
	if len(user_ids) == 0 {
		ctx.JSONAPI(http.StatusBadRequest, "there's no user", nil)
		return
	}
	if param.PermID == libs.DefaultGroup {
		ctx.JSONAPI(http.StatusBadRequest, "you cannot modify the default group", nil)
		return
	}
	if !internal.PMExists(param.PermID) {
		ctx.JSONAPI(http.StatusBadRequest, "invalid request: permission id is wrong", nil)
		return
	}

	var err error
	num_ids := make([]int, len(user_ids))
	for i, j := range user_ids {
		num_ids[i], err = strconv.Atoi(j)
		if err != nil {
			ctx.JSONAPI(http.StatusBadRequest, "invalid request: meet user id \""+j+"\"", nil)
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
		ctx.JSONAPI(http.StatusBadRequest, "invalid ids exist", gin.H{"invalid_ids": invalid_ids})
		return
	}
	res, err := internal.PMAddUser(real_ids, param.PermID)
	if err != nil {
		ctx.ErrorAPI(err)
	} else {
		ctx.JSONAPI(http.StatusOK, "", gin.H{"affected": res})
	}
}

type PermDelUserParam struct {
	PermID  int `query:"permission_id" binding:"required" validate:"prmsid"`
	UserID  int `query:"user_id" binding:"required" validate:"userid"`
	UserGrp int `session:"user_group" validate:"admin"`
}

func PermDelUser(ctx Context, param PermDelUserParam) {
	if param.PermID == libs.DefaultGroup {
		ctx.JSONAPI(http.StatusBadRequest, "you cannot modify the default group", nil)
		return
	}
	_, err := internal.PMDeleteUser(param.PermID, param.UserID)
	if err != nil {
		ctx.ErrorAPI(err)
	}
}
