package main

import (
	"net/http"
	"time"
)

func createStatusHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	content := r.FormValue("content")
	if content == "" {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	_, err := db.Exec(`
		INSERT INTO statuses (user_id, content, expires_at)
		VALUES ($1, $2, $3)
	`, userID, content, time.Now().Add(24*time.Hour))

	if err != nil {
		http.Error(w, "Unable to create status.", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
