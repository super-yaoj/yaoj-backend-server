package db

import (
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	_ "github.com/go-sql-driver/mysql"
)

var db *sqlx.DB

func Init(dsn string) error {
	var err error
	db, err = sqlx.Connect("mysql", dsn)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(64)
	db.SetMaxIdleConns(25)
	return nil
}

func Query(query string, args ...any) (*sqlx.Rows, error) {
	return db.Queryx(query, args...)
}

func SelectInts(query string, args ...any) ([]int, error) {
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

func SelectAll(arr any, query string, args ...any) error {
	return db.Select(arr, query, args...)
}

func SelectSingleInt(query string, args ...any) (int, error) {
	var a int
	rows, err := db.Queryx(query, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&a)
	} else {
		err = errors.New("no rows read in SelectSingleInt()")
	}
	return a, err
}

func SelectSingle(arr any, query string, args ...any) error {
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
	return errors.New("no rows read by SelectSingle()")
}

func SelectSingleColumn(arr any, query string, args ...any) error {
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
	return errors.New("no rows read by SelectSingleColumn()")
}

func Update(query string, args ...any) (sql.Result, error) {
	return db.Exec(query, args...)
}

func UpdateGetAffected(query string, args ...any) (int64, error) {
	res, err := db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func InsertGetId(query string, args ...any) (int64, error) {
	res, err := db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func Close() error {
	return db.Close()
}
