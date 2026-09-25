package main

import (
	"net/http"
)

func updateLastActive(r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		return
	}

	_, err := db.Exec(
		"UPDATE users SET last_active = CURRENT_TIMESTAMP WHERE id = $1",
		userID,
	)

	if err != nil {
		return
	}
}

func getOnlineCount() int {
	var count int

	err := db.QueryRow(
		"SELECT COUNT(*) FROM users WHERE last_active >= CURRENT_TIMESTAMP - INTERVAL '5 minutes'",
	).Scan(&count)

	if err != nil {
		return 0
	}

	return count
}
