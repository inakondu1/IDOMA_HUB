package main

import (
	"net/http"
	"strconv"
)

func statusViewHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Error(w, "Please log in to view this status.", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	statusID, err := strconv.Atoi(r.FormValue("status_id"))
	if err != nil || statusID <= 0 {
		http.Error(w, "Invalid status.", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(`
		INSERT INTO status_views (status_id, viewer_id)
		SELECT id, $2
		FROM statuses
		WHERE id = $1
		  AND expires_at > CURRENT_TIMESTAMP
		  AND user_id <> $2
		ON CONFLICT (status_id, viewer_id) DO NOTHING
	`, statusID, userID)
	if err != nil {
		http.Error(w, "Unable to record status view.", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
