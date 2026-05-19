package main

import (
	"database/sql"
	"fmt"
	"gorb/config"
	"gorb/db"
	"log"

	"github.com/joho/godotenv"
)

var DB *sql.DB

func main() {
	fmt.Println("Hello, World!")
	
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	myconf, err := config.Load()
	if err != nil {
		log.Fatal("Env conf error", err)
	}
	DB, _ = db.InitDB(myconf)

}
