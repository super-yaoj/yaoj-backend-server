// use dbr for faster and more convenient database functionality
package db

import "github.com/gocraft/dbr/v2"

func NewDBR(driver string, dsn string) (*dbr.Connection, error) {
	conn, err := dbr.Open(driver, dsn, nil)
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(64)
	conn.SetMaxIdleConns(25)
	return conn, nil
}
