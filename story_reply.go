package main

import (
	"net/http"
	"strconv"
	"strings"
)

func storyReplyHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	statusID, err := strconv.Atoi(r.FormValue("status_id"))
	if err != nil || statusID <= 0 {
		http.Error(w, "Invalid story.", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))

	if content == "" {
		http.Error(w, "Reply cannot be empty.", http.StatusBadRequest)
		return
	}

	if len(content) > 1000 {
		http.Error(w, "Reply is too long.", http.StatusBadRequest)
		return
	}

	var ownerID int

	err = db.QueryRow(`
		SELECT user_id
		FROM statuses
		WHERE id = $1
		  AND expires_at > CURRENT_TIMESTAMP
	`, statusID).Scan(&ownerID)

	if err != nil {
		http.Error(w, "This story is no longer available.", http.StatusNotFound)
		return
	}

	if ownerID == userID {
		http.Error(w, "You cannot reply to your own story.", http.StatusBadRequest)
		return
	}

	message := "Reply to your story: " + content

	_, err = db.Exec(`
		INSERT INTO messages (sender_id, receiver_id, content)
		VALUES ($1, $2, $3)
	`, userID, ownerID, message)

	if err != nil {
		http.Error(w, "Unable to send story reply.", http.StatusInternalServerError)
		return
	}

	http.Redirect(
		w,
		r,
		"/messages?user_id="+strconv.Itoa(ownerID),
		http.StatusSeeOther,
	)
}
