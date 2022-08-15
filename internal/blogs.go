package internal

import (
	"time"
	"yao/db"

	utils "github.com/super-yaoj/yaoj-utils"
)

type Blog struct {
	Id         int       `db:"blog_id" json:"blog_id"`
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

func BlogCreate(user_id, private int, title, content string) (int64, error) {
	return db.DBInsertGetId("insert into blogs values (null, ?, ?, ?, ?, ?, 0, 0)", user_id, title, content, private, time.Now())
}

func BlogEdit(id, private int, title, content string) error {
	_, err := db.DBUpdate("update blogs set title=?, content=?, private=? where blog_id=?", title, content, private, id)
	return err
}

func BlogDelete(id int) error {
	_, err := db.DBUpdate("delete from blogs where blog_id=?", id)
	if err != nil {
		return err
	} else {
		db.DBUpdate("delete from click_like where target=? and id=?", BLOG, id)
		db.DBUpdate("delete from click_like where target=? and id in (select comment_id from blog_comments where blog_id=?)", COMMENT, id)
		db.DBUpdate("delete from blog_comments where blog_id=?", id)
		return nil
	}
}

func BlogQuery(id, user_id int) (Blog, error) {
	var blog Blog
	err := db.DBSelectSingle(&blog, "select * from blogs where blog_id=?", id)
	if err != nil {
		return blog, err
	} else {
		db.DBSelectSingle(&blog, "select user_name from user_info where user_id=?", blog.Author)
		blog.Liked = GetLike(BLOG, user_id, id)
		return blog, nil
	}
}

func BlogListUser(id, user_id int) ([]Blog, error) {
	var blogs []Blog
	err := db.DBSelectAll(&blogs, "select blog_id, title, private, create_time, comments, `like` from blogs where author=?", id)
	BlogGetLikes(blogs, user_id)
	return blogs, err
}

func BlogListAll(bound, pagesize, user_id int, isleft, isadmin bool) ([]Blog, bool, error) {
	pagesize += 1
	var err error
	var blogs []Blog
	if isleft {
		if isadmin {
			err = db.DBSelectAll(&blogs, "select blog_id, title, author, create_time, private, user_name, comments, `like` from (user_info join blogs on user_info.user_id=blogs.author) where blog_id<? order by blog_id desc limit ?", bound, pagesize)
		} else {
			err = db.DBSelectAll(&blogs, "select blog_id, title, author, create_time, private, user_name, comments, `like` from (user_info join blogs on user_info.user_id=blogs.author) where blog_id<? and (author=? or private=0) order by blog_id desc limit ?", bound, user_id, pagesize)
		}
	} else {
		if isadmin {
			err = db.DBSelectAll(&blogs, "select blog_id, title, author, create_time, private, user_name, comments, `like` from (user_info join blogs on user_info.user_id=blogs.author) where blog_id>? order by blog_id limit ?", bound, pagesize)
		} else {
			err = db.DBSelectAll(&blogs, "select blog_id, title, author, create_time, private, user_name, comments, `like` from (user_info join blogs on user_info.user_id=blogs.author) where blog_id>? and (author=? or private=0) order by blog_id desc limit ?", bound, user_id, pagesize)
		}
	}
	if err != nil {
		return nil, false, err
	}
	var isfull = len(blogs) == pagesize
	if isfull {
		blogs = blogs[:pagesize-1]
	}
	if !isleft {
		utils.Reverse(blogs)
	}
	BlogGetLikes(blogs, user_id)
	return blogs, isfull, nil
}

func BlogGetLikes(blogs []Blog, user_id int) {
	if user_id < 0 {
		return
	}
	ids := []int{}
	for _, i := range blogs {
		ids = append(ids, i.Id)
	}
	ret := GetLikes(BLOG, user_id, ids)
	for i := range blogs {
		blogs[i].Liked = utils.HasInt(ret, blogs[i].Id)
	}
}

func BlogExists(blog_id int) bool {
	count, err := db.DBSelectSingleInt("select count(*) from blogs where blog_id=?", blog_id)
	return err == nil && count > 0
}
