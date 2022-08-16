// 存放数据库用到的数据结构模型
package db

import (
	"time"

	"github.com/gocraft/dbr/v2"
)

type Blog struct {
	// 博客 ID，正整数
	BlogID     uint      `db:"blog_id" json:"blog_id"`
	Author     int       `db:"author" json:"author"`
	AuthorName string    `db:"user_name" json:"author_name"`
	Title      string    `db:"title" json:"title"`
	Content    string    `db:"content" json:"content"`
	Private    bool      `db:"private" json:"private"`
	CreateTime time.Time `db:"create_time" json:"create_time"`
	Comments   int       `db:"comments" json:"comments"`
	Like       int       `db:"like" json:"like"`
	Liked      bool      `json:"liked"`
}

func NewBlogMgr(sess *dbr.Session) *blogMgr {
	return &blogMgr{sess: sess}
}

type blogMgr struct {
	sess *dbr.Session
}

func (r *blogMgr) Exist(blogID int) bool {
	var cnt int
	err := r.sess.Select("count(*)").From("blogs").Where("blog_id=?", blogID).LoadOne(&cnt)
	if err != nil {
		panic(err)
	}
	return cnt > 0
}
