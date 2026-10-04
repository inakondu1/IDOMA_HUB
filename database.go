package main

import (
	"database/sql"
	"encoding/csv"
	"io"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func importIdomaCSV(db *sql.DB, filename string, dataType string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)

	_, err = reader.Read()
	if err != nil {
		return err
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if len(record) < 5 {
			continue
		}

		_, err = db.Exec(`
                        INSERT INTO idoma_language_data
                        (english, idoma, type, status, source, notes)
                        SELECT $1, $2, $3, $4, $5, $6
                        WHERE NOT EXISTS (
                                SELECT 1 FROM idoma_language_data
                                WHERE english = $1 AND idoma = $2 AND type = $3
                        )
                `, record[0], record[1], dataType, record[2], record[3], record[4])

		if err != nil {
			return err
		}
	}

	return nil
}

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
                CREATE TABLE IF NOT EXISTS statuses (
                        id SERIAL PRIMARY KEY,
                        user_id INTEGER NOT NULL,
                        content TEXT,
                        media_url TEXT,
                        media_type TEXT,
                        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                        expires_at TIMESTAMP NOT NULL,
                        FOREIGN KEY (user_id) REFERENCES users(id)
                )
        `)
	if err != nil {
		log.Fatal("Unable to create statuses table:", err)
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

	_, err = db.Exec(`
                CREATE TABLE IF NOT EXISTS messages (
                        id SERIAL PRIMARY KEY,
                        sender_id INTEGER NOT NULL,
                        receiver_id INTEGER NOT NULL,
                        content TEXT NOT NULL,
                        is_read INTEGER NOT NULL DEFAULT 0,
                        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                        FOREIGN KEY (sender_id) REFERENCES users(id),
                        FOREIGN KEY (receiver_id) REFERENCES users(id)
                )
        `)
	if err != nil {
		log.Fatal("Unable to create messages table:", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS farm_produce (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL,
			image_url TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatal("Unable to create farm_produce table:", err)
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS idoma_language_data (
            id SERIAL PRIMARY KEY,
            english TEXT NOT NULL,
            idoma TEXT NOT NULL,
            type TEXT NOT NULL,
            status TEXT NOT NULL,
            source TEXT NOT NULL,
            notes TEXT NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `)
	if err != nil {
		log.Fatal("Unable to create idoma_language_data table:", err)
	}

	var languageCount int
	err = db.QueryRow("SELECT COUNT(*) FROM idoma_language_data").Scan(&languageCount)
	if err != nil {
		log.Fatal("Unable to check Idoma language data:", err)
	}
	if languageCount == 0 {
		if err = importIdomaCSV(db, "verified_idoma.csv", "word"); err != nil {
			log.Fatal("Unable to import verified Idoma words:", err)
		}
		if err = importIdomaCSV(db, "everyday_idoma_phrases.csv", "phrase"); err != nil {
			log.Fatal("Unable to import Idoma phrases:", err)
		}
		log.Println("Idoma language data imported successfully.")
	}
	var farmProduceCount int
	err = db.QueryRow("SELECT COUNT(*) FROM farm_produce").Scan(&farmProduceCount)
	if err != nil {
		log.Fatal("Unable to check farm produce:", err)
	}

	if farmProduceCount == 0 {
		_, err = db.Exec(`
			INSERT INTO farm_produce (name, description, image_url) VALUES
			('Plantain', 'Plantain is grown in many farming communities and is an important source of food.', '/static/images/farm-produce/plantain.jpeg'),
			('Tomatoes', 'Tomatoes are an important vegetable crop used in many meals and grown by farmers in different communities.', '/static/images/farm-produce/tomatoes.jpeg'),
			('Rice', 'Rice is an important food crop grown and consumed across many communities.', '/static/images/farm-produce/rice.jpeg'),
			('Rice Farm', 'Rice farming provides food and supports the livelihood of farmers and farming communities.', '/static/images/farm-produce/rice-farm.jpeg'),
			('Pawpaw', 'Pawpaw is a nutritious fruit that can be grown in farming communities and enjoyed as part of the local food supply.', '/static/images/farm-produce/pawpaw.jpeg'),
			('Yam', 'Yam is an important food crop in Idoma communities and is closely connected with farming, food and cultural life.', '' ),
			('Cassava', 'Cassava is another important crop grown by farmers and used in different forms of food.', '' ),
			('Fruits', 'Fruits such as mango and orange are also part of the agricultural produce found in Idoma communities.', '' )
		`)
		if err != nil {
			log.Fatal("Unable to insert farm produce:", err)
		}
	}

	log.Println("PostgreSQL database connected successfully.")

	return db
}
