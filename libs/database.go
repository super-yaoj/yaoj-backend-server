package libs

import (
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	_ "github.com/go-sql-driver/mysql"
)

var db *sqlx.DB

func DBInit() error {
	dsn := "yaoj@tcp(127.0.0.1:3306)/yaoj?charset=utf8mb4&parseTime=True&multiStatements=true"
	var err error
	db, err = sqlx.Connect("mysql", dsn)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(64)
	db.SetMaxIdleConns(25)
	return nil
}

func DBQuery(query string, args ...any) (*sqlx.Rows, error) {
	return db.Queryx(query, args...)
}

func DBSelectInts(query string, args ...any) ([]int, error) {
	var ret []int
	row, err := db.Queryx(query, args...)
	if err != nil {
		return nil, err
	}
	defer row.Close()
	for row.Next() {
		var cur int
		row.Scan(&cur)
		ret = append(ret, cur)
	}
	return ret, nil
}

func DBSelectAll(arr any, query string, args ...any) error {
	return db.Select(arr, query, args...)
}

func DBSelectSingleInt(query string, args ...any) (int, error) {
	var a int
	rows, err := db.Queryx(query, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&a)
	} else {
		err = errors.New("no rows read in DBSelectSingleInt()")
	}
	return a, err
}

func DBSelectSingle(arr any, query string, args ...any) error {
	rows, err := db.Queryx(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.StructScan(arr)
		rows.Close()
		return err
	}
	return errors.New("no rows read by DBSelectSingle()")
}

func DBSelectSingleColumn(arr any, query string, args ...any) error {
	rows, err := db.Queryx(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(arr)
		rows.Close()
		return err
	}
	return errors.New("no rows read by DBSelectSingleColumn()")
}

func DBUpdate(query string, args ...any) (sql.Result, error) {
	return db.Exec(query, args...)
}

func DBUpdateGetAffected(query string, args ...any) (int64, error) {
	res, err := db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func DBInsertGetId(query string, args ...any) (int64, error) {
	res, err := db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func DBClose() error {
	return db.Close()
}
