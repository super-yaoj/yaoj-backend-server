package controllers

import (
	"time"
	"yao/libs"
)

type Comment struct {
	Id         int       `db:"comment_id" json:"comment_id"`
	BlogId     int       `db:"blog_id" json:"blog_id"`
	CreateTime time.Time `db:"create_time" json:"create_time"`
	Author     int       `db:"author" json:"author"`
	AuthorName string    `db:"user_name" json:"author_name"`
	Content    string    `db:"content" json:"content"`
	Like       int       `db:"like" json:"like"`
	Liked      bool      `json:"liked"`
}

func BLCreateComment(blog_id, user_id int, content string) (int64, error) {
	id, err := libs.DBInsertGetId("insert into blog_comments values (?, now(), ?, ?, 0, null)", blog_id, user_id, content)
	if err != nil {
		return 0, err
	} else {
		libs.DBUpdate("update blogs set comments=comments+1 where blog_id=?", blog_id)
		return id, err
	}
}

func BLGetComments(blog_id, user_id int) ([]Comment, error) {
	var comments []Comment
	err := libs.DBSelectAll(&comments, "select comment_id, author, content, `like`, create_time, user_name from ((select * from blog_comments where blog_id=?) as a join user_info on author=user_id)", blog_id)
	if err != nil { return nil, err }
	BLCommentsGetLike(comments, user_id)
	return comments, nil
}

func BLCommentsGetLike(comments []Comment, user_id int) {
	if user_id > 0 {
		var ids []int
		for _, i := range comments {
			ids = append(ids, i.Id)
		}
		ret := GetLikes(COMMENT, user_id, ids)
		for i := range comments {
			comments[i].Liked = libs.HasInt(ret, comments[i].Id)
		}
	}
}

func BLDeleteComment(id, blog_id int) error {
	_, err := libs.DBUpdate("delete from blog_comments where comment_id=?", id)
	if err != nil { return err }
	_, err = libs.DBUpdate("delete from click_like where target=? and id=?", COMMENT, id)
	if err != nil { return err }
	_, err = libs.DBUpdate("update blogs set comments=comments-1 where blog_id=?", blog_id)
	return err
}

func BLCommentExists(comment_id int) bool {
	count, err := libs.DBSelectSingleInt("select count(*) from blog_comments where comment_id=?", comment_id)
	return err == nil && count > 0
}