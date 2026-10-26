package main

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func statusReshareHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	statusID, err := strconv.Atoi(r.FormValue("status_id"))
	if err != nil || statusID <= 0 {
		http.Error(w, "Invalid status.", http.StatusBadRequest)
		return
	}

	message := strings.TrimSpace(r.FormValue("message"))
	if len(message) > 1000 {
		http.Error(w, "Your reshare message is too long.", http.StatusBadRequest)
		return
	}

	var originalUserID int
	var originalUsername string
	var rootStatusID int
	var content string
	var mediaURL string
	var mediaType string

	err = db.QueryRow(`
		SELECT COALESCE(NULLIF(statuses.original_user_id, 0), statuses.user_id),
		       COALESCE(NULLIF(statuses.original_username, ''), users.username),
		       COALESCE(NULLIF(statuses.original_status_id, 0), statuses.id),
		       COALESCE(statuses.content, ''),
		       COALESCE(statuses.media_url, ''),
		       COALESCE(statuses.media_type, '')
		FROM statuses
		JOIN users ON users.id = statuses.user_id
		WHERE statuses.id = $1
		  AND statuses.expires_at > CURRENT_TIMESTAMP
	`, statusID).Scan(
		&originalUserID,
		&originalUsername,
		&rootStatusID,
		&content,
		&mediaURL,
		&mediaType,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "This status is no longer available.", http.StatusNotFound)
			return
		}
		http.Error(w, "Unable to load the original status.", http.StatusInternalServerError)
		return
	}

	if originalUserID == userID {
		http.Error(w, "You cannot reshare your own status.", http.StatusBadRequest)
		return
	}

	if content == "" && mediaURL == "" {
		http.Error(w, "This status cannot be reshared.", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Unable to reshare this status.", http.StatusInternalServerError)
		return
	}

	var resharedStatusID int
	err = tx.QueryRow(`
		INSERT INTO statuses (
			user_id, content, media_url, media_type, expires_at,
			original_status_id, original_user_id, original_username, reshare_message
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`,
		userID, content, mediaURL, mediaType, time.Now().Add(24*time.Hour),
		rootStatusID, originalUserID, originalUsername, message,
	).Scan(&resharedStatusID)

	if err != nil {
		_ = tx.Rollback()
		http.Error(w, "Unable to reshare this status.", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(`
		INSERT INTO status_shares (status_id, reshared_status_id, sharer_id)
		VALUES ($1, $2, $3)
	`, rootStatusID, resharedStatusID, userID)
	if err != nil {
		_ = tx.Rollback()
		http.Error(w, "Unable to record this reshare.", http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, "Unable to complete this reshare.", http.StatusInternalServerError)
		return
	}

	createStoryNotification(originalUserID, userID, "status_reshare")
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
