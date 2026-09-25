package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func initDatabase() *sql.DB {
	databaseURL := os.Getenv("DATABASE_URL")

	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal("Unable to open database:", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Unable to connect to database:", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatal("Unable to create users table:", err)
	}

	_, err = db.Exec(`
		ALTER TABLE users
		ADD COLUMN IF NOT EXISTS last_active TIMESTAMP
	`)
	if err != nil {
		log.Fatal("Unable to add last_active column:", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS friend_requests (
			id SERIAL PRIMARY KEY,
			sender_id INTEGER NOT NULL,
			receiver_id INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(sender_id, receiver_id),
			FOREIGN KEY (sender_id) REFERENCES users(id),
			FOREIGN KEY (receiver_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		log.Fatal("Unable to create friend requests table:", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS posts (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		log.Fatal("Unable to create posts table:", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS post_likes (
			id SERIAL PRIMARY KEY,
			post_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(post_id, user_id),
			FOREIGN KEY (post_id) REFERENCES posts(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		log.Fatal("Unable to create post likes table:", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS comments (
			id SERIAL PRIMARY KEY,
			post_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (post_id) REFERENCES posts(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		log.Fatal("Unable to create comments table:", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS notifications (
			id SERIAL PRIMARY KEY,
			recipient_id INTEGER NOT NULL,
			sender_id INTEGER NOT NULL,
			post_id INTEGER,
			type TEXT NOT NULL,
			is_read INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (recipient_id) REFERENCES users(id),
			FOREIGN KEY (sender_id) REFERENCES users(id),
			FOREIGN KEY (post_id) REFERENCES posts(id)
		)
	`)
	if err != nil {
		log.Fatal("Unable to create notifications table:", err)
	}

	_, err = db.Exec(`
		ALTER TABLE users
		ADD COLUMN IF NOT EXISTS profile_picture TEXT
	`)
	if err != nil {
		log.Fatal("Unable to add profile_picture column:", err)
	}

	_, err = db.Exec(`
		ALTER TABLE posts
		ADD COLUMN IF NOT EXISTS media_url TEXT
	`)
	if err != nil {
		log.Fatal("Unable to add media_url column:", err)
	}

	_, err = db.Exec(`
		ALTER TABLE posts
		ADD COLUMN IF NOT EXISTS media_type TEXT
	`)
	if err != nil {
		log.Fatal("Unable to add media_type column:", err)
	}

	_, err = db.Exec(`
		ALTER TABLE posts
		ADD COLUMN IF NOT EXISTS original_post_id INTEGER
	`)
	if err != nil {
		log.Fatal("Unable to add original_post_id column:", err)
	}

	log.Println("PostgreSQL database connected successfully.")

	return db
}
