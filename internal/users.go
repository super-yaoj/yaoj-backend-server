package internal

import (
	"regexp"
	"time"
	"yao/libs"
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

func SaltPassword(password string) string {
	return libs.MySHA256(libs.Sault + password)
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
	count, err := libs.DBSelectSingleInt("select count(*) from user_info where user_id=? and password=?", user_id, SaltPassword(password))
	return count == 1, err
}

func USQuery(user_id int) (User, error) {
	var user User
	err := libs.DBSelectSingle(&user, "select * from user_info where user_id=?", user_id)
	return user, err
}

func UserQueryBase(user_id int) (UserBase, error) {
	user := UserBase{Id: user_id}
	err := libs.DBSelectSingle(&user, "select user_name, user_group, rating from user_info where user_id=?", user_id)
	return user, err
}

func USModify(password string, gender int, motto, email, organization string, user_id int) error {
	_, err := libs.DBUpdate("update user_info set password=?, gender=?, motto=?, email=?, organization=? where user_id=?", password, gender, motto, email, organization, user_id)
	return err
}

func USGroupEdit(user_id, user_group int) error {
	_, err := libs.DBUpdateGetAffected("update user_info set user_group=? where user_id=?", user_group, user_id)
	return err
}

func USList(bound_user_id, bound_rating, pagesize int, isleft bool) ([]User, bool, error) {
	pagesize += 1
	var users []User
	var err error
	if isleft {
		err = libs.DBSelectAll(&users, "select user_id, user_name, motto, rating from user_info where rating<? or (rating=? and user_id>=?) order by rating desc,user_id limit ?", bound_rating, bound_rating, bound_user_id, pagesize)
	} else {
		err = libs.DBSelectAll(&users, "select user_id, user_name, motto, rating from user_info where rating>? or (rating=? and user_id<=?) order by rating, user_id desc limit ?", bound_rating, bound_rating, bound_user_id, pagesize)
	}
	if err != nil {
		return nil, false, err
	}
	isfull := pagesize == len(users)
	if isfull {
		users = users[:pagesize-1]
	}
	if !isleft {
		libs.Reverse(users)
	}
	return users, isfull, nil
}

func USQueryPermission(user_id int) ([]Permission, error) {
	var p []Permission
	err := libs.DBSelectAll(&p, "select permissions.permission_id, permission_name, count from (permissions join user_permissions on permissions.permission_id=user_permissions.permission_id) where user_id=?", user_id)
	return p, err
}

func USPermissions(user_id int) ([]int, error) {
	if user_id < 0 {
		return []int{libs.DefaultGroup}, nil
	}
	p, err := libs.DBSelectInts("select permission_id from user_permissions where user_id=?", user_id)
	if err != nil {
		return nil, nil
	}
	return append(p, -user_id), err
}

func USExists(user_id int) bool {
	count, _ := libs.DBSelectSingleInt("select count(*) from user_info where user_id=?", user_id)
	return count > 0
}

func USListByName(user_name string, bound, pagesize int, isleft bool) ([]UserBase, bool, error) {
	var users []UserBase
	pagesize += 1
	var err error
	if isleft {
		err = libs.DBSelectAll(&users, "select user_id, user_name, rating from user_info where user_id>=? and user_name like ? order by user_id limit ?", bound, user_name, pagesize)
	} else {
		err = libs.DBSelectAll(&users, "select user_id, user_name, rating from user_info where user_id<=? and user_name like ? order by user_id desc limit ?", bound, user_name, pagesize)
	}
	isfull := len(users) == pagesize
	if isfull {
		users = users[:pagesize-1]
	}
	if !isleft {
		libs.Reverse(users)
	}
	return users, isfull, err
}
