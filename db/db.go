package db

import (
	"database/sql"
	"gorb/config"
	"log"

	_ "github.com/go-mysql-org/go-mysql/driver"
)

var DB *sql.DB

func InitDB(conf *config.Config) (*sql.DB, error) {
	dns := conf.DatabaseURL()
	log.Print(dns)
	db, err := sql.Open("mysql", dns)
	if err != nil {
		log.Fatal("Could not connect to db", err)
		return nil, err
	}
	defer db.Close()
	log.Print("connected successfully")
	return db, nil
}
