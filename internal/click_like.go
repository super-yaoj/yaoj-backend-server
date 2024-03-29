package internal

import (
	"fmt"
	"sort"
	"yao/db"

	utils "github.com/super-yaoj/yaoj-utils"
)

type ClickType int

const (
	BLOG    ClickType = 1
	CONTEST ClickType = 2
	PROBLEM ClickType = 3
	COMMENT ClickType = 4
)

func ClickLike(target ClickType, id, user_id int) error {
	has := GetLike(target, user_id, id)
	val := 1
	if has {
		val = -1
	}
	var err error
	switch target {
	case BLOG:
		_, err = db.Exec("update blogs set `like` = `like` + ? where blog_id=?", val, id)
	case COMMENT:
		_, err = db.Exec("update blog_comments set `like` = `like` + ? where comment_id=?", val, id)
	case PROBLEM:
		_, err = db.Exec("update problems set `like` = `like` + ? where problem_id=?", val, id)
	case CONTEST:
		_, err = db.Exec("update contests set `like` = `like` + ? where contest_id=?", val, id)
	}
	if err != nil {
		return err
	}
	if has {
		_, err = db.Exec("delete from click_like where target=? and id=? and user_id=?", target, id, user_id)
	} else {
		_, err = db.Exec("insert into click_like values (?,?,?)", target, id, user_id)
	}
	return err
}

func GetLikes(target ClickType, user_id int, ids []int) []int {
	if len(ids) == 0 {
		return []int{}
	}
	ret, _ := db.SelectInts(fmt.Sprintf("select id from click_like where target=%d and user_id=%d and id in (%s)", target, user_id, utils.JoinArray(ids)))
	sort.Ints(ret)
	return ret
}

func GetLike(target ClickType, user_id, id int) bool {
	val, _ := db.SelectSingleInt("select count(*) from click_like where target=? and id=? and user_id=?", target, id, user_id)
	return val > 0
}
