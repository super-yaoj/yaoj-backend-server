package controllers

import (
	"fmt"
	"sort"
	"yao/libs"
)

const (
	BLOG = 1
	CONTEST = 2
	PROBLEM = 3
	COMMENT = 4
)

func ClickLike(target, id, user_id int) error {
	has := GetLike(target, user_id, id)
	val := 1; if has { val = -1 }
	var err error
	switch target {
	case BLOG:
		_, err = libs.DBUpdate("update blogs set `like` = `like` + ? where blog_id=?", val, id)
	case COMMENT:
		_, err = libs.DBUpdate("update blog_comments set `like` = `like` + ? where comment_id=?", val, id)
	case PROBLEM:
		_, err = libs.DBUpdate("update problems set `like` = `like` + ? where problem_id=?", val, id)
	case CONTEST:
		_, err = libs.DBUpdate("update contests set `like` = `like` + ? where contest_id=?", val, id)
	}
	if err != nil { return err }
	if has {
		_, err = libs.DBUpdate("delete from click_like where target=? and id=? and user_id=?", target, id, user_id)
	} else {
		_, err = libs.DBUpdate("insert into click_like values (?,?,?)", target, id, user_id)
	}
	return err
}

func GetLikes(target, user_id int, ids []int) []int {
	if len(ids) == 0 { return []int{} }
	ret, _ := libs.DBSelectInts(fmt.Sprintf("select id from click_like where target=%d and user_id=%d and id in (%s)", target, user_id, libs.JoinArray(ids)))
	sort.Ints(ret)
	return ret
}

func GetLike(target, user_id, id int) bool {
	val, _ := libs.DBSelectSingleInt("select count(*) from click_like where target=? and id=? and user_id=?", target, id, user_id)
	return val > 0
}