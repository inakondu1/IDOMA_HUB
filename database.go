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
		ALTER TABLE posts
		ADD COLUMN IF NOT EXISTS category TEXT DEFAULT 'general'
	`)
	if err != nil {
		log.Fatal("Unable to add post category column:", err)
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

	_, err = db.Exec(`
                CREATE TABLE IF NOT EXISTS groups (
                        id SERIAL PRIMARY KEY,
                        name TEXT NOT NULL UNIQUE,
                        description TEXT NOT NULL,
                        creator_id INTEGER NOT NULL,
                        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                        FOREIGN KEY (creator_id) REFERENCES users(id)
                )
        `)
	if err != nil {
		log.Fatal("Unable to create groups table:", err)
	}

	_, err = db.Exec(`
                CREATE TABLE IF NOT EXISTS group_members (
                        id SERIAL PRIMARY KEY,
                        group_id INTEGER NOT NULL,
                        user_id INTEGER NOT NULL,
                        joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                        UNIQUE(group_id, user_id),
                        FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
                        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
                )
        `)
	if err != nil {
		log.Fatal("Unable to create group_members table:", err)
	}

	_, err = db.Exec(`
                CREATE TABLE IF NOT EXISTS group_posts (
                        id SERIAL PRIMARY KEY,
                        group_id INTEGER NOT NULL,
                        user_id INTEGER NOT NULL,
                        content TEXT NOT NULL,
                        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                        FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
                        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
                )
        `)
	if err != nil {
		log.Fatal("Unable to create group_posts table:", err)
	}

	log.Println("PostgreSQL database connected successfully.")

	// Anonymous room tables.
	// IMPORTANT: these tables intentionally contain NO user_id.
	// Anonymous activity must never be mapped back to a real account.

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS anonymous_sessions (
				id SERIAL PRIMARY KEY,
				session_hash TEXT NOT NULL UNIQUE,
				anonymous_name TEXT NOT NULL,
				expires_at TIMESTAMP NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatal("Unable to create anonymous_sessions table:", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS anonymous_messages (
				id SERIAL PRIMARY KEY,
				anonymous_name TEXT NOT NULL,
				content TEXT NOT NULL DEFAULT '',
				media_url TEXT NOT NULL DEFAULT '',
				media_public_id TEXT NOT NULL DEFAULT '',
				media_type TEXT NOT NULL DEFAULT '',
				media_duration INTEGER NOT NULL DEFAULT 0,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				expires_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		log.Fatal("Unable to create anonymous_messages table:", err)
	}

	// Add parent_id so Anonymous messages can have direct replies.
	_, err = db.Exec(`
            ALTER TABLE anonymous_messages
            ADD COLUMN IF NOT EXISTS parent_id INTEGER
    `)
	if err != nil {
		log.Fatal("Unable to add Anonymous reply column:", err)
	}

	_, err = db.Exec(`
            CREATE INDEX IF NOT EXISTS idx_anonymous_messages_parent_id
            ON anonymous_messages(parent_id)
    `)
	if err != nil {
		log.Fatal("Unable to create Anonymous reply index:", err)
	}

	_, err = db.Exec(`
            CREATE TABLE IF NOT EXISTS anonymous_views (
                            id SERIAL PRIMARY KEY,
                            session_hash TEXT NOT NULL,
                            message_id INTEGER NOT NULL,
                            viewed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                            UNIQUE(session_hash, message_id),
                            FOREIGN KEY (message_id)
                                            REFERENCES anonymous_messages(id)
                                            ON DELETE CASCADE
            )
    `)
	if err != nil {
		log.Fatal("Unable to create anonymous_views table:", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS anonymous_presence (
				session_hash TEXT NOT NULL UNIQUE,
				expires_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		log.Fatal("Unable to create anonymous_presence table:", err)
	}

	// Remove expired Anonymous data.
	_, err = db.Exec(`
		DELETE FROM anonymous_views
		WHERE message_id IN (
				SELECT id
				FROM anonymous_messages
				WHERE expires_at <= CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatal("Unable to clean anonymous views:", err)
	}

	_, err = db.Exec(`
		DELETE FROM anonymous_messages
		WHERE expires_at <= CURRENT_TIMESTAMP
	`)
	if err != nil {
		log.Fatal("Unable to clean anonymous messages:", err)
	}

	_, err = db.Exec(`
		DELETE FROM anonymous_sessions
		WHERE expires_at <= CURRENT_TIMESTAMP
	`)
	if err != nil {
		log.Fatal("Unable to clean anonymous sessions:", err)
	}

	_, err = db.Exec(`
		DELETE FROM anonymous_presence
		WHERE expires_at <= CURRENT_TIMESTAMP
	`)
	if err != nil {
		log.Fatal("Unable to clean anonymous presence:", err)
	}

	// IDOMA HUB Earn Together: activity tracking and monthly reward records.
	// Cash payouts must remain disabled until funding and payment arrangements are ready.

	_, err = db.Exec(`
                CREATE TABLE IF NOT EXISTS reward_activity (
                        id BIGSERIAL PRIMARY KEY,
                        user_id INTEGER NOT NULL REFERENCES users(id),
                        creator_id INTEGER REFERENCES users(id),
                        post_id INTEGER,
                        activity_type TEXT NOT NULL
                                CHECK (activity_type IN ('like', 'comment', 'video_watch')),
                        activity_key TEXT NOT NULL,
                        quantity NUMERIC(12, 2) NOT NULL DEFAULT 1 CHECK (quantity >= 0),
                        reward_month DATE NOT NULL,
                        status TEXT NOT NULL DEFAULT 'pending'
                                CHECK (status IN ('pending', 'eligible', 'rejected')),
                        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                        UNIQUE (user_id, activity_type, activity_key)
                )
        `)
	if err != nil {
		log.Fatal("Unable to create reward_activity table:", err)
	}

	// Add creator tracking to existing databases.
	_, err = db.Exec(`
                ALTER TABLE reward_activity
                ADD COLUMN IF NOT EXISTS creator_id INTEGER REFERENCES users(id)
        `)
	if err != nil {
		log.Fatal("Unable to add reward creator ID:", err)
	}

	// Backfill creator IDs for existing activities.
	_, err = db.Exec(`
                UPDATE reward_activity ra
                SET creator_id = p.user_id
                FROM posts p
                WHERE ra.post_id = p.id
                  AND ra.creator_id IS NULL
        `)
	if err != nil {
		log.Fatal("Unable to backfill reward creator IDs:", err)
	}

	// Create server-side video watch sessions.
	_, err = db.Exec(`
                CREATE TABLE IF NOT EXISTS video_watch_sessions (
                        token TEXT PRIMARY KEY,
                        viewer_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                        post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
                        creator_id INTEGER NOT NULL REFERENCES users(id),
                        started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                        expires_at TIMESTAMP NOT NULL,
                        completed_at TIMESTAMP
                )
        `)
	if err != nil {
		log.Fatal("Unable to create video watch sessions:", err)
	}

	_, err = db.Exec(`
                CREATE INDEX IF NOT EXISTS idx_video_watch_sessions_viewer
                ON video_watch_sessions(viewer_id, expires_at)
        `)
	if err != nil {
		log.Fatal("Unable to create video watch session index:", err)
	}

	_, err = db.Exec(`
                CREATE INDEX IF NOT EXISTS idx_reward_activity_month_status
                ON reward_activity(reward_month, status)
        `)
	if err != nil {
		log.Fatal("Unable to create reward activity month index:", err)
	}

	_, err = db.Exec(`
                CREATE TABLE IF NOT EXISTS monthly_earnings (
                        id BIGSERIAL PRIMARY KEY,
                        user_id INTEGER NOT NULL REFERENCES users(id),
                        reward_month DATE NOT NULL,
                        watch_seconds BIGINT NOT NULL DEFAULT 0 CHECK (watch_seconds >= 0),
                        eligible_comments INTEGER NOT NULL DEFAULT 0 CHECK (eligible_comments >= 0),
                        eligible_likes INTEGER NOT NULL DEFAULT 0 CHECK (eligible_likes >= 0),
                        amount_usd NUMERIC(12, 4) NOT NULL DEFAULT 0 CHECK (amount_usd >= 0),
                        status TEXT NOT NULL DEFAULT 'pending'
                                CHECK (status IN ('pending', 'under_review', 'approved', 'paid', 'rejected')),
                        calculated_at TIMESTAMP,
                        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                        UNIQUE (user_id, reward_month)
                )
        `)
	if err != nil {
		log.Fatal("Unable to create monthly_earnings table:", err)
	}

	_, err = db.Exec(`
                CREATE TABLE IF NOT EXISTS reward_payouts (
                        id BIGSERIAL PRIMARY KEY,
                        user_id INTEGER NOT NULL REFERENCES users(id),
                        reward_month DATE NOT NULL,
                        amount_usd NUMERIC(12, 4) NOT NULL CHECK (amount_usd >= 0),
                        payout_method TEXT NOT NULL DEFAULT '',
                        payout_currency TEXT NOT NULL DEFAULT 'USD',
                        exchange_rate NUMERIC(18, 6),
                        amount_local NUMERIC(18, 2),
                        status TEXT NOT NULL DEFAULT 'pending'
                                CHECK (status IN ('pending', 'processing', 'paid', 'failed', 'cancelled')),
                        provider_reference TEXT NOT NULL DEFAULT '',
                        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                        UNIQUE (user_id, reward_month)
                )
        `)
	if err != nil {
		log.Fatal("Unable to create reward_payouts table:", err)
	}

	_, err = db.Exec(`
                CREATE TABLE IF NOT EXISTS admin_users (
                        user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
                        role TEXT NOT NULL CHECK (role IN ('owner', 'admin')),
                        granted_by INTEGER REFERENCES users(id),
                        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
                )
        `)
	if err != nil {
		log.Fatal("Unable to create admin_users table:", err)
	}

	_, err = db.Exec(`
                CREATE TABLE IF NOT EXISTS creator_monetization_applications (
                        id BIGSERIAL PRIMARY KEY,
                        user_id INTEGER NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
                        status TEXT NOT NULL DEFAULT 'pending'
                                CHECK (status IN ('pending', 'approved', 'rejected', 'suspended')),
                        application_note TEXT NOT NULL DEFAULT '',
                        reviewed_by INTEGER REFERENCES users(id),
                        reviewed_at TIMESTAMP,
                        review_note TEXT NOT NULL DEFAULT '',
                        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
                )
        `)
	if err != nil {
		log.Fatal("Unable to create creator monetization applications table:", err)
	}

	_, err = db.Exec(`
                CREATE TABLE IF NOT EXISTS admin_audit_log (
                        id BIGSERIAL PRIMARY KEY,
                        actor_id INTEGER REFERENCES users(id),
                        target_user_id INTEGER REFERENCES users(id),
                        action TEXT NOT NULL,
                        details TEXT NOT NULL DEFAULT '',
                        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
                )
        `)
	if err != nil {
		log.Fatal("Unable to create admin audit log table:", err)
	}

	_, err = db.Exec(`
                CREATE INDEX IF NOT EXISTS idx_creator_monetization_status
                ON creator_monetization_applications(status, created_at)
        `)
	if err != nil {
		log.Fatal("Unable to create monetization status index:", err)
	}

	return db
}
