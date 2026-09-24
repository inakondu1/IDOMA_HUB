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

	_, err = db.Exec(`
                CREATE TABLE IF NOT EXISTS friend_requests (
                        id INTEGER PRIMARY KEY AUTOINCREMENT,
                        sender_id INTEGER NOT NULL,
                        receiver_id INTEGER NOT NULL,
                        status TEXT NOT NULL DEFAULT 'pending',
                        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
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
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)

	if err != nil {
		log.Fatal("Unable to create posts table:", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS post_likes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			post_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
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
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			post_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (post_id) REFERENCES posts(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)

	if err != nil {
		log.Fatal("Unable to create comments table:", err)
	}

	_, err = db.Exec(`
                CREATE TABLE IF NOT EXISTS notifications (
                        id INTEGER PRIMARY KEY AUTOINCREMENT,
                        recipient_id INTEGER NOT NULL,
                        sender_id INTEGER NOT NULL,
                        post_id INTEGER,
                        type TEXT NOT NULL,
                        is_read INTEGER NOT NULL DEFAULT 0,
                        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                        FOREIGN KEY (recipient_id) REFERENCES users(id),
                        FOREIGN KEY (sender_id) REFERENCES users(id),
                        FOREIGN KEY (post_id) REFERENCES posts(id)
                )
        `)

	if err != nil {
		log.Fatal("Unable to create notifications table:", err)
	}

	addPostMediaColumns(db)
	addProfilePictureColumn(db)

	log.Println("Database connected successfully.")

	return db
}

func addProfilePictureColumn(db *sql.DB) {
	rows, err := db.Query("PRAGMA table_info(users)")
	if err != nil {
		log.Fatal("Unable to inspect users table:", err)
	}
	defer rows.Close()

	columns := make(map[string]bool)

	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue interface{}
		var primaryKey int

		err := rows.Scan(
			&cid,
			&name,
			&columnType,
			&notNull,
			&defaultValue,
			&primaryKey,
		)
		if err != nil {
			log.Fatal("Unable to read users table information:", err)
		}

		columns[name] = true
	}

	if !columns["profile_picture"] {
		_, err = db.Exec("ALTER TABLE users ADD COLUMN profile_picture TEXT")
		if err != nil {
			log.Fatal("Unable to add profile_picture column:", err)
		}
	}
}

func addPostMediaColumns(db *sql.DB) {
	rows, err := db.Query("PRAGMA table_info(posts)")
	if err != nil {
		log.Fatal("Unable to inspect posts table:", err)
	}
	defer rows.Close()

	columns := make(map[string]bool)

	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue interface{}
		var primaryKey int

		err := rows.Scan(
			&cid,
			&name,
			&columnType,
			&notNull,
			&defaultValue,
			&primaryKey,
		)
		if err != nil {
			log.Fatal("Unable to read posts table information:", err)
		}

		columns[name] = true
	}

	if !columns["media_url"] {
		_, err = db.Exec("ALTER TABLE posts ADD COLUMN media_url TEXT")
		if err != nil {
			log.Fatal("Unable to add media_url column:", err)
		}
	}

	if !columns["media_type"] {
		_, err = db.Exec("ALTER TABLE posts ADD COLUMN media_type TEXT")
		if err != nil {
			log.Fatal("Unable to add media_type column:", err)
		}
	}

	if !columns["original_post_id"] {
		_, err = db.Exec("ALTER TABLE posts ADD COLUMN original_post_id INTEGER")
		if err != nil {
			log.Fatal("Unable to add original_post_id column:", err)
		}
	}

}
