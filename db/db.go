package db

import (
	"database/sql"
	"gorb/config"
	"log"

	_ "github.com/go-mysql-org/go-mysql/driver"
)

func InitDB(conf *config.Config) (*sql.DB, error) {
	dns := conf.DatabaseURL()
	db, err := sql.Open("mysql", dns)
	if err != nil {
		return nil, err
	}
	log.Print("connected successfully")
	return db, nil
}
