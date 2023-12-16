package initializers

import (
	"database/sql"
	"log"
	"os"
)

func init() {
	LoadEnvVariables()
}

var DB *sql.DB

func ConnectToDB() {
	var err error
	connStr := os.Getenv("DB_URL")
	DB, err = sql.Open("postgres", connStr)

	if err != nil {
		log.Fatal("Failed to connect to database")
	}
}
