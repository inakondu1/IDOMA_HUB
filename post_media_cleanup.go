package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func postMediaLockKey(mediaURL, mediaPublicID string) string {
	if mediaURL != "" {
		return "post-media-url:" + mediaURL
	}
	if mediaPublicID != "" {
		return "post-media-id:" + mediaPublicID
	}
	return ""
}

func lockPostMedia(tx *sql.Tx, mediaURL, mediaPublicID string) error {
	keys := make([]string, 0, 2)

	if mediaURL != "" {
		keys = append(keys, "post-media-url:"+mediaURL)
	}
	if mediaPublicID != "" {
		keys = append(keys, "post-media-id:"+mediaPublicID)
	}

	sort.Strings(keys)

	for _, key := range keys {
		if _, err := tx.Exec(
			`SELECT pg_advisory_xact_lock(hashtext($1))`, key,
		); err != nil {
			return err
		}
	}

	return nil
}

// cleanupDeletedPostMedia removes media only when no remaining post references it.
func cleanupDeletedPostMedia(mediaURL, mediaType, mediaPublicID string) {
	key := postMediaLockKey(mediaURL, mediaPublicID)
	if key == "" {
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Printf("Unable to start post media cleanup transaction: %v", err)
		return
	}
	defer tx.Rollback()

	if err := lockPostMedia(tx, mediaURL, mediaPublicID); err != nil {
		log.Printf("Unable to lock post media for cleanup: %v", err)
		return
	}

	var references int
	err = tx.QueryRow(`
		SELECT COUNT(*)
		FROM posts
		WHERE ($1 <> '' AND COALESCE(media_public_id, '') = $1)
		   OR ($2 <> '' AND COALESCE(media_url, '') = $2)
	`, mediaPublicID, mediaURL).Scan(&references)
	if err != nil {
		log.Printf("Unable to check remaining post media references; skipping cleanup: %v", err)
		return
	}

	if references > 0 {
		return
	}

	if mediaPublicID != "" {
		deletePostMediaFromCloudinary(mediaPublicID, mediaType)
	}

	prefix := "/static/uploads/posts/"
	if strings.HasPrefix(mediaURL, prefix) {
		filename := strings.TrimPrefix(mediaURL, prefix)
		if filename != "" && !strings.Contains(filename, "..") &&
			!strings.Contains(filename, "/") {
			filePath := filepath.Join(postUploadDir(), filename)
			if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
				log.Printf("Unable to delete post media file: %v", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Unable to finish post media cleanup transaction: %v", err)
	}
}
