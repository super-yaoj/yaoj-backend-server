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
	Auth
	PermName string `body:"permission_name" validate:"lte=190"`
}

func PermCreate(ctx Context, param PermCreateParam) {
	param.NewPermit().AsAdmin().Success(func(a any) {
		id, err := internal.PMCreate(param.PermName)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", gin.H{"permission_id": id})
		}
	}).FailAPIStatusForbidden(ctx)
}

type PermRenameParam struct {
	Auth
	PermID   int    `body:"permission_id" validate:"required,prmsid"`
	PermName string `body:"permission_name" validate:"lte=190"`
}

func PermRename(ctx Context, param PermRenameParam) {
	param.NewPermit().AsAdmin().Success(func(a any) {
		err := internal.PMChangeName(param.PermID, param.PermName)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}

type PermDelParam struct {
	Auth
	PermID  int `query:"permission_id" validate:"required,prmsid"`
}

func PermDel(ctx Context, param PermDelParam) {
	param.NewPermit().AsAdmin().Success(func(a any) {
		if param.PermID == libs.DefaultGroup {
			ctx.JSONAPI(http.StatusBadRequest, "you cannot modify the default group", nil)
			return
		}
		_, err := internal.PMDelete(param.PermID)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}

type PermGetParam struct {
	Auth
	Page `validate:"pagecanbound"`
}

func PermGet(ctx Context, param PermGetParam) {
	param.NewPermit().AsAdmin().Success(func(a any) {
		p, isfull, err := internal.PMQuery(param.Bound(), *param.PageSize, param.IsLeft())
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", gin.H{"isfull": isfull, "data": p})
		}
	}).FailAPIStatusForbidden(ctx)
}

type PermGetUserParam struct {
	Auth
	PermID     int `query:"permission_id" validate:"required,prmsid"`
}

func PermGetUser(ctx Context, param PermGetUserParam) {
	param.NewPermit().AsAdmin().Success(func(a any) {
		users, err := internal.PMQueryUser(param.PermID)
		if err != nil {
			ctx.ErrorAPI(err)
		} else {
			ctx.JSONAPI(http.StatusOK, "", map[string]any{"data": users})
		}
	}).FailAPIStatusForbidden(ctx)
}

type PermAddUserParam struct {
	Auth
	PermID  int    `body:"permission_id" validate:"required,prmsid"`
	UserIDs string `body:"user_ids"`
}

func PermAddUser(ctx Context, param PermAddUserParam) {
	param.NewPermit().AsAdmin().Success(func(a any) {
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
	}).FailAPIStatusForbidden(ctx)
}

type PermDelUserParam struct {
	Auth
	PermID  int `query:"permission_id" validate:"required,prmsid"`
	UserID  int `query:"user_id" validate:"required,userid"`
}

func PermDelUser(ctx Context, param PermDelUserParam) {
	param.NewPermit().AsAdmin().Success(func(a any) {
		if param.PermID == libs.DefaultGroup {
			ctx.JSONAPI(http.StatusBadRequest, "you cannot modify the default group", nil)
			return
		}
		_, err := internal.PMDeleteUser(param.PermID, param.UserID)
		if err != nil {
			ctx.ErrorAPI(err)
		}
	}).FailAPIStatusForbidden(ctx)
}
