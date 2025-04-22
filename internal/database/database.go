package database

import (
	"database/sql"
	"fmt"
	"simple-invest/internal/servicelog"

	_ "github.com/lib/pq"
)

var (
	db *sql.DB
)

func init() {
	var err error
	db, err = connectDB()
	if err != nil {
		servicelog.ErrorLog().Panicf("ошибка подключения к базе данных: %s", err.Error())
	}
}

func connectDB() (*sql.DB, error) {

	connstr := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=%s",
		"postgres",
		"postgres",
		"invest_db",
		"disable")
	db, err := sql.Open("postgres", connstr)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func DB() *sql.DB {
	return db
}
