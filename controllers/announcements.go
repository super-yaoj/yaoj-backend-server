package controllers

import (
	"time"
	"yao/libs"
)

type Announcement struct {
	Id 			int 		`db:"id" json:"id"`
	BlogId 		int 		`db:"blog_id" json:"blog_id"`
	Title 		string 		`db:"title" json:"title"`
	Priority 	int 		`db:"priority" json:"priority"`
	ReleaseTime	time.Time 	`db:"release_time" json:"release_time"`
	Comments 	int 		`db:"comments" json:"comments"`
	Like 		int 		`db:"like" json:"like"`
	Liked 		bool 		`json:"liked"`
}

func ANCreate(blog_id, priority int) error {
	_, err := libs.DBUpdate("insert into announcements values (?, ?, ?, null)", blog_id, time.Now(), priority)
	return err
}

func ANQuery(user_id int) []Announcement {
	var ans []Announcement
	libs.DBSelectAll(&ans, "select id, blogs.blog_id, title, priority, comments, `like`, release_time from (announcements join blogs on announcements.blog_id = blogs.blog_id)")
	ANGetLikes(ans, user_id)
	return ans
}

func ANGetLikes(blogs []Announcement, user_id int) {
	if user_id < 0 { return }
	ids := []int{}
	for _, i := range blogs {
		ids = append(ids, i.BlogId)
	}
	ret := GetLikes(BLOG, user_id, ids)
	for i := range blogs {
		blogs[i].Liked = libs.HasInt(ret, blogs[i].BlogId)
	}
}

func ANDelete(id int) {
	libs.DBUpdate("delete from announcements where id=?", id)
}