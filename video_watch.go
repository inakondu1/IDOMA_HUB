package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type videoWatchStartResponse struct {
	Token string `json:"token"`
}

func videoWatchStartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	if !validSameOriginRequest(r) {
		http.Error(w, "Invalid request origin.", http.StatusForbidden)
		return
	}

	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Error(w, "Please log in first.", http.StatusUnauthorized)
		return
	}

	updateLastActive(r)

	postID, err := strconv.Atoi(r.FormValue("post_id"))
	if err != nil || postID <= 0 {
		http.Error(w, "Invalid post ID.", http.StatusBadRequest)
		return
	}

	var creatorID int
	err = db.QueryRow(`
		SELECT user_id
		FROM posts
		WHERE id = $1 AND media_type = 'video'
	`, postID).Scan(&creatorID)

	if err == sql.ErrNoRows {
		http.Error(w, "Video post not found.", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println("Unable to validate video watch session:", err)
		http.Error(w, "Unable to start video watch session.", http.StatusInternalServerError)
		return
	}

	if creatorID == userID {
		http.Error(w, "You cannot earn credit for watching your own video.", http.StatusForbidden)
		return
	}

	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		log.Println("Unable to generate video watch token:", err)
		http.Error(w, "Unable to start video watch session.", http.StatusInternalServerError)
		return
	}

	token := hex.EncodeToString(randomBytes)

	_, err = db.Exec(`
		INSERT INTO video_watch_sessions
			(token, viewer_id, post_id, creator_id, expires_at)
		VALUES (
			$1, $2, $3, $4,
			CURRENT_TIMESTAMP + INTERVAL '10 minutes'
		)
	`, token, userID, postID, creatorID)

	if err != nil {
		log.Println("Unable to create video watch session:", err)
		http.Error(w, "Unable to start video watch session.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")

	if err := json.NewEncoder(w).Encode(videoWatchStartResponse{Token: token}); err != nil {
		log.Println("Unable to send video watch token:", err)
	}
}

func videoWatchCompleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	if !validSameOriginRequest(r) {
		http.Error(w, "Invalid request origin.", http.StatusForbidden)
		return
	}

	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Error(w, "Please log in first.", http.StatusUnauthorized)
		return
	}

	updateLastActive(r)

	token := strings.TrimSpace(r.FormValue("token"))
	if len(token) != 64 {
		http.Error(w, "Invalid video watch session.", http.StatusBadRequest)
		return
	}

	if _, err := hex.DecodeString(token); err != nil {
		http.Error(w, "Invalid video watch session.", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Println("Unable to begin video watch completion:", err)
		http.Error(w, "Unable to complete video watch session.", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var postID, creatorID int

	err = tx.QueryRow(`
		UPDATE video_watch_sessions
		SET completed_at = CURRENT_TIMESTAMP
		WHERE token = $1
		  AND viewer_id = $2
		  AND completed_at IS NULL
		  AND expires_at > CURRENT_TIMESTAMP
		  AND started_at <= CURRENT_TIMESTAMP - INTERVAL '30 seconds'
		RETURNING post_id, creator_id
	`, token, userID).Scan(&postID, &creatorID)

	if err == sql.ErrNoRows {
		http.Error(w, "The session is invalid, expired, already completed, or too short.", http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Println("Unable to validate video watch completion:", err)
		http.Error(w, "Unable to complete video watch session.", http.StatusInternalServerError)
		return
	}

	var currentCreatorID int
	err = tx.QueryRow(`
		SELECT user_id
		FROM posts
		WHERE id = $1 AND media_type = 'video'
	`, postID).Scan(&currentCreatorID)

	if err != nil || currentCreatorID != creatorID || creatorID == userID {
		http.Error(w, "The video is no longer valid for this session.", http.StatusForbidden)
		return
	}

	_, err = tx.Exec(`
		INSERT INTO reward_activity (
			user_id, creator_id, post_id, activity_type, activity_key,
			quantity, reward_month, status
		)
		SELECT
			$1,
			$2,
			$3,
			'video_watch',
			'post:' || $3::text || ':' ||
				TO_CHAR(CURRENT_TIMESTAMP, 'YYYY-MM'),
			30,
			DATE_TRUNC('month', CURRENT_TIMESTAMP)::date,
			CASE
				WHEN EXISTS (
					SELECT 1
					FROM creator_monetization_applications ca
					WHERE ca.user_id = $2
					  AND ca.status = 'approved'
				) THEN 'pending'
				ELSE 'rejected'
			END
		ON CONFLICT (user_id, activity_type, activity_key) DO NOTHING
	`, userID, creatorID, postID)

	if err != nil {
		log.Println("Unable to record completed video watch:", err)
		http.Error(w, "Unable to record video watch activity.", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Println("Unable to commit video watch completion:", err)
		http.Error(w, "Unable to complete video watch session.", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Retire the old endpoint so clients cannot submit their own watch duration.
func videoWatchHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Please use a server-verified video watch session.", http.StatusGone)
}
