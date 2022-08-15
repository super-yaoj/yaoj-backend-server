package internal

import (
	"regexp"
	"time"
	"yao/config"
	"yao/db"

	utils "github.com/super-yaoj/yaoj-utils"
)

type UserBase struct {
	Id        int    `db:"user_id" json:"user_id"`
	Name      string `db:"user_name" json:"user_name"`
	Usergroup int    `db:"user_group" json:"user_group"`
	Rating    int    `db:"rating" json:"rating"`
}

type User struct {
	UserBase
	Password      string    `db:"password" json:"password"`
	Motto         string    `db:"motto" json:"motto"`
	RegisterTime  time.Time `db:"register_time" json:"register_time"`
	RememberToken string    `db:"remember_token" json:"remember_token"`
	Gender        int       `db:"gender" json:"gender"`
	Email         string    `db:"email" json:"email"`
	Organization  string    `db:"organization" json:"organization"`
}

const (
	UserBanned = 0
	UserNormal = 1
	UserAdmin  = 2
	UserRoot   = 3
)

func IsAdmin(user_group int) bool {
	return (user_group == UserAdmin || user_group == UserRoot)
}

func IsBanned(user_group int) bool {
	return user_group == UserBanned
}

func SaltPassword(password string) string {
	return utils.SHA256(config.Global.Sault + password)
}

func ValidUsername(name string) bool {
	matched, err := regexp.MatchString("^[\\w_]{3,18}$", name)
	if err != nil {
		return false
	}
	return matched
}

func ValidPassword(password string) bool {
	matched, err := regexp.MatchString("^[A-Z0-9]{64}$", password)
	if err != nil {
		return false
	}
	return matched
}

func CheckPassword(user_id int, password string) (bool, error) {
	if !ValidPassword(password) {
		return false, nil
	}
	count, err := db.SelectSingleInt("select count(*) from user_info where user_id=? and password=?", user_id, SaltPassword(password))
	return count == 1, err
}

func UserQuery(user_id int) (User, error) {
	var user User
	err := db.SelectSingle(&user, "select * from user_info where user_id=?", user_id)
	return user, err
}

func UserQueryBase(user_id int) (UserBase, error) {
	user := UserBase{Id: user_id}
	err := db.SelectSingle(&user, "select user_name, user_group, rating from user_info where user_id=?", user_id)
	return user, err
}

func Userubmodify(password string, gender int, motto, email, organization string, user_id int) error {
	_, err := db.Exec("update user_info set password=?, gender=?, motto=?, email=?, organization=? where user_id=?", password, gender, motto, email, organization, user_id)
	return err
}

func UserGroupEdit(user_id, user_group int) error {
	_, err := db.ExecGetAffected("update user_info set user_group=? where user_id=?", user_group, user_id)
	return err
}

func UserList(bound_user_id, bound_rating, pagesize int, isleft bool) ([]User, bool, error) {
	pagesize += 1
	var users []User
	var err error
	if isleft {
		err = db.SelectAll(&users, "select user_id, user_name, motto, rating from user_info where rating<? or (rating=? and user_id>=?) order by rating desc,user_id limit ?", bound_rating, bound_rating, bound_user_id, pagesize)
	} else {
		err = db.SelectAll(&users, "select user_id, user_name, motto, rating from user_info where rating>? or (rating=? and user_id<=?) order by rating, user_id desc limit ?", bound_rating, bound_rating, bound_user_id, pagesize)
	}
	if err != nil {
		return nil, false, err
	}
	isfull := pagesize == len(users)
	if isfull {
		users = users[:pagesize-1]
	}
	if !isleft {
		utils.Reverse(users)
	}
	return users, isfull, nil
}

func UserQueryPermission(user_id int) ([]Permission, error) {
	var p []Permission
	err := db.SelectAll(&p, "select permissions.permission_id, permission_name, count from (permissions join user_permissions on permissions.permission_id=user_permissions.permission_id) where user_id=?", user_id)
	return p, err
}

func UserPermissions(user_id int) ([]int, error) {
	if user_id < 0 {
		return []int{config.Global.DefaultGroup}, nil
	}
	p, err := db.SelectInts("select permission_id from user_permissions where user_id=?", user_id)
	if err != nil {
		return nil, nil
	}
	return append(p, -user_id), err
}

func UserExists(user_id int) bool {
	count, _ := db.SelectSingleInt("select count(*) from user_info where user_id=?", user_id)
	return count > 0
}

func UserListByName(user_name string, bound, pagesize int, isleft bool) ([]UserBase, bool, error) {
	var users []UserBase
	pagesize += 1
	var err error
	if isleft {
		err = db.SelectAll(&users, "select user_id, user_name, rating from user_info where user_id>=? and user_name like ? order by user_id limit ?", bound, user_name, pagesize)
	} else {
		err = db.SelectAll(&users, "select user_id, user_name, rating from user_info where user_id<=? and user_name like ? order by user_id desc limit ?", bound, user_name, pagesize)
	}
	isfull := len(users) == pagesize
	if isfull {
		users = users[:pagesize-1]
	}
	if !isleft {
		utils.Reverse(users)
	}
	return users, isfull, err
}
