package main

import (
	"net/http"
	"strconv"
)

func videoWatchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Error(w, "Please log in first.", http.StatusUnauthorized)
		return
	}

	updateLastActive(r)

	postID := r.FormValue("post_id")
	watchedSeconds, err := strconv.Atoi(r.FormValue("watched_seconds"))
	if err != nil || watchedSeconds != 30 {
		http.Error(w, "A valid 30-second watch report is required.", http.StatusBadRequest)
		return
	}

	postNumber, err := strconv.Atoi(postID)
	if err != nil || postNumber <= 0 {
		http.Error(w, "Invalid post ID.", http.StatusBadRequest)
		return
	}

	var ownerID int
	err = db.QueryRow(
		"SELECT user_id FROM posts WHERE id = $1 AND media_type = 'video'",
		postNumber,
	).Scan(&ownerID)

	if err != nil {
		http.Error(w, "Video post not found.", http.StatusNotFound)
		return
	}

	if ownerID == userID {
		http.Error(w, "You cannot earn credit for watching your own video.", http.StatusForbidden)
		return
	}

	recordVideoWatchActivity(
		userID,
		postNumber,
		watchedSeconds,
	)

	w.WriteHeader(http.StatusNoContent)
}
