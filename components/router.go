package components

import (
	"time"
	"yao/libs"

	"github.com/gin-gonic/gin"
)

type Request struct {
	Method   string
	Function func(*gin.Context)
}

var Router map[string][]Request = map[string][]Request {
	"/GetTime": {{"POST", GetTime}},
	"/Init": {{"POST", USInit}},
	"/UserLogin": {{"POST", USLogin}},
	"/UserLogout": {{"POST", USLogout}},
	
	"/user": {{"GET", USQuery}, {"POST", USSignup}, {"PUT", USModify}, {"PATCH", USGroupEdit}},
	"/captcha": {{"GET", CaptchaImage}, {"POST", CaptchaId}, {"PATCH", ReloadCaptchaImage}},
	"/permissions": {{"GET", PMQuery}, {"POST", PMCreate}, {"PATCH", PMChangeName}, {"DELETE", PMDelete}},
	"/user_permissions": {{"GET", PMQueryUser}, {"POST", PMAddUser}, {"DELETE", PMDeleteUser}},
	"/users": {{"GET", USList}},
	
	"/blog": {{"GET", BLQuery}, {"POST", BLCreate}, {"PUT", BLEdit}, {"DELETE", BLDelete}},
	"/blogs": {{"GET", BLList}},
	"/likes": {{"POST", ClickLike}},
	"/blog_comments": {{"GET", BLGetComments}, {"POST", BLCreateComment}, {"DELETE", BLDeleteComment}},
	"/announcements": {{"GET", ANQuery}, {"POST", ANCreate}, {"DELETE", ANDelete}},
	
	"/problems": {{"GET", PRList}},
	"/problem": {{"GET", PRQuery}, {"POST", PRCreate}, {"PATCH", PRModify}},
	"/problem_permissions": {{"GET", PRGetPermissions}, {"POST", PRAddPermission}, {"DELETE", PRDeletePermission}},
	"/problem_managers": {{"GET", PRGetManagers}, {"POST", PRAddManager}, {"DELETE", PRDeleteManager}},
	"/problem_data": {{"GET", PRDownloadData}, {"PUT", PRPutData}},
	
	"/contests": {{"GET", CTList}},
	"/contest": {{"GET", CTQuery}, {"POST", CTCreate}, {"PATCH", CTModify}},
	"/contest_permissions": {{"GET", CTGetPermissions}, {"POST", CTAddPermission}, {"DELETE", CTDeletePermission}},
	"/contest_managers": {{"GET", CTGetManagers}, {"POST", CTAddManager}, {"DELETE", CTDeleteManager}},
	"/contest_participants": {{"GET", CTGetParticipants}, {"POST", CTSignup}, {"DELETE", CTSignout}},
	"/contest_problems": {{"GET", CTGetProblems}, {"POST", CTAddProblem}, {"DELETE", CTDeleteProblem}},
	
	"/submission": {{"POST", SMSubmit}},
	"/submissions": {{"GET", SMList}},
}

func GetTime(ctx *gin.Context) {
	libs.RPCWriteBack(ctx, 200, 0, "", map[string]any{ "server_time": time.Now() })
}