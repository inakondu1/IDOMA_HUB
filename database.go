package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func initDatabase() *sql.DB {
	db, err := sql.Open("sqlite3", "idoma_hub.db")
	if err != nil {
		log.Fatal("Unable to open database:", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Unable to connect to database:", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)

	if err != nil {
		log.Fatal("Unable to create users table:", err)
	}

	log.Println("Database connected successfully.")

	return db
}
