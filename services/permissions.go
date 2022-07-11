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
}

func PermCreate(ctx *gin.Context, param PermCreateParam) {
	if !ISAdmin(ctx) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	if len(param.PermName) > 190 {
		libs.APIWriteBack(ctx, 400, "permission name is too long", nil)
		return
	}
	id, err := internal.PMCreate(param.PermName)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", gin.H{"permission_id": id})
	}
}

type PermRenameParam struct {
	PermID   int    `body:"permission_id" binding:"required"`
	PermName string `body:"permission_name"`
}

func PermRename(ctx *gin.Context, param PermRenameParam) {
	if !ISAdmin(ctx) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	if len(param.PermName) > 190 {
		libs.APIWriteBack(ctx, 400, "permission name is too long", nil)
		return
	}
	if !internal.PMExists(param.PermID) {
		libs.APIWriteBack(ctx, 400, "no such permission id", nil)
		return
	}
	err := internal.PMChangeName(param.PermID, param.PermName)
	if err != nil {
		libs.APIInternalError(ctx, err)
	}
}

type PermDelParam struct {
	PermID int `query:"permission_id" binding:"required"`
}

func PermDel(ctx *gin.Context, param PermDelParam) {
	if !ISAdmin(ctx) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	if param.PermID == libs.DefaultGroup {
		libs.APIWriteBack(ctx, 400, "you cannot modify the default group", nil)
		return
	}
	num, err := internal.PMDelete(param.PermID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else if num != 1 {
		libs.APIWriteBack(ctx, 400, "no such permission id", nil)
	} else {
		libs.APIWriteBack(ctx, 200, "", nil)
	}
}

type PermGetParam struct {
	PageSize int  `query:"pagesize" binding:"required"`
	Left     *int `query:"left"`
	Right    *int `query:"right"`
}

func PermGet(ctx *gin.Context, param PermGetParam) {
	if !ISAdmin(ctx) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	if param.PageSize < 1 || param.PageSize > 100 {
		return
	}
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
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", gin.H{"isfull": isfull, "data": p})
	}
}

type PermGetUserParam struct {
	PermID    *int `query:"permission_id"`
	UserID    int  `query:"user_id"`
	CurUserID int  `session:"user_id"`
}

func PermGetUser(ctx *gin.Context, param PermGetUserParam) {
	if param.PermID != nil {
		if !ISAdmin(ctx) {
			libs.APIWriteBack(ctx, 403, "", nil)
			return
		}
		users, err := internal.PMQueryUser(*param.PermID)
		if err != nil {
			libs.APIInternalError(ctx, err)
		} else {
			libs.APIWriteBack(ctx, 200, "", map[string]any{"data": users})
		}
	} else {
		if !ISAdmin(ctx) && param.UserID != param.CurUserID {
			libs.APIWriteBack(ctx, 403, "", nil)
			return
		}
		permissions, err := internal.USQueryPermission(param.UserID)
		if err != nil {
			libs.APIInternalError(ctx, err)
		} else {
			libs.APIWriteBack(ctx, 200, "", map[string]any{"data": permissions})
		}
	}
}

type PermAddUserParam struct {
	PermID  int    `body:"permission_id" binding:"required"`
	UserIDs string `body:"user_ids"`
}

func PermAddUser(ctx *gin.Context, param PermAddUserParam) {
	if !ISAdmin(ctx) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	user_ids := strings.Split(param.UserIDs, ",")
	if len(user_ids) == 0 {
		libs.APIWriteBack(ctx, 400, "there's no user", nil)
		return
	}
	if param.PermID == libs.DefaultGroup {
		libs.APIWriteBack(ctx, 400, "you cannot modify the default group", nil)
		return
	}
	if !internal.PMExists(param.PermID) {
		libs.APIWriteBack(ctx, 400, "invalid request: permission id is wrong", nil)
		return
	}

	var err error
	num_ids := make([]int, len(user_ids))
	for i, j := range user_ids {
		num_ids[i], err = strconv.Atoi(j)
		if err != nil {
			libs.APIWriteBack(ctx, 400, "invalid request: meet user id \""+j+"\"", nil)
			return
		}
	}
	real_ids, err := libs.DBSelectInts(fmt.Sprintf("select user_id from user_info where user_id in (%s) order by user_id", param.UserIDs))
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
		libs.APIWriteBack(ctx, 400, "invalid ids exist", gin.H{"invalid_ids": invalid_ids})
		return
	}
	res, err := internal.PMAddUser(real_ids, param.PermID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else {
		libs.APIWriteBack(ctx, 200, "", gin.H{"affected": res})
	}
}

type PermDelUserParam struct {
	PermID int `query:"permission_id" binding:"required"`
	UserID int `query:"user_id" binding:"required"`
}

func PermDelUser(ctx *gin.Context, param PermDelUserParam) {
	if !ISAdmin(ctx) {
		libs.APIWriteBack(ctx, 403, "", nil)
		return
	}
	if param.PermID == libs.DefaultGroup {
		libs.APIWriteBack(ctx, 400, "you cannot modify the default group", nil)
		return
	}
	res, err := internal.PMDeleteUser(param.PermID, param.UserID)
	if err != nil {
		libs.APIInternalError(ctx, err)
	} else if res != 1 {
		libs.APIWriteBack(ctx, 400, "doesn't exist", nil)
	}
}
