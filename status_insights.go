package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type statusViewDetail struct {
	Username string    `json:"username"`
	ViewedAt time.Time `json:"viewed_at"`
}

type statusShareDetail struct {
	Username string    `json:"username"`
	SharedAt time.Time `json:"shared_at"`
}

type statusInsightsResponse struct {
	ViewCount  int                 `json:"view_count"`
	ShareCount int                 `json:"share_count"`
	Viewers    []statusViewDetail  `json:"viewers"`
	Shares     []statusShareDetail `json:"shares"`
}

func statusInsightsHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Error(w, "Please log in.", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	statusID, err := strconv.Atoi(r.URL.Query().Get("status_id"))
	if err != nil || statusID <= 0 {
		http.Error(w, "Invalid status.", http.StatusBadRequest)
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
		http.Error(w, "Status not found.", http.StatusNotFound)
		return
	}

	if ownerID != userID {
		http.Error(w, "Status not found.", http.StatusNotFound)
		return
	}

	response := statusInsightsResponse{
		Viewers: []statusViewDetail{},
		Shares:  []statusShareDetail{},
	}

	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM status_views
		WHERE status_id = $1
	`, statusID).Scan(&response.ViewCount)
	if err != nil {
		http.Error(w, "Unable to load status views.", http.StatusInternalServerError)
		return
	}

	rows, err := db.Query(`
		SELECT users.username, status_views.viewed_at
		FROM status_views
		JOIN users ON users.id = status_views.viewer_id
		WHERE status_views.status_id = $1
		ORDER BY status_views.viewed_at DESC
	`, statusID)
	if err != nil {
		http.Error(w, "Unable to load viewers.", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var detail statusViewDetail
		if err := rows.Scan(&detail.Username, &detail.ViewedAt); err != nil {
			http.Error(w, "Unable to read viewers.", http.StatusInternalServerError)
			return
		}
		response.Viewers = append(response.Viewers, detail)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Unable to read viewers.", http.StatusInternalServerError)
		return
	}

	if err := rows.Close(); err != nil {
		http.Error(w, "Unable to close viewer rows.", http.StatusInternalServerError)
		return
	}

	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM status_shares
		WHERE status_id = $1
	`, statusID).Scan(&response.ShareCount)
	if err != nil {
		http.Error(w, "Unable to load reshares.", http.StatusInternalServerError)
		return
	}

	rows, err = db.Query(`
		SELECT users.username, status_shares.shared_at
		FROM status_shares
		JOIN users ON users.id = status_shares.sharer_id
		WHERE status_shares.status_id = $1
		ORDER BY status_shares.shared_at DESC
	`, statusID)
	if err != nil {
		http.Error(w, "Unable to load resharers.", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var detail statusShareDetail
		if err := rows.Scan(&detail.Username, &detail.SharedAt); err != nil {
			http.Error(w, "Unable to read resharers.", http.StatusInternalServerError)
			return
		}
		response.Shares = append(response.Shares, detail)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Unable to read resharers.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}
