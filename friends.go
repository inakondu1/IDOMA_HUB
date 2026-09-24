package main

import (
	"fmt"
	"html/template"
	"net/http"
)

type FriendUser struct {
	ID             int
	Username       string
	ProfilePicture string
	Status         string
}

type FriendRequest struct {
	ID             int
	UserID         int
	Username       string
	ProfilePicture string
}

type FriendsPageData struct {
	Suggestions   []FriendUser
	SearchResults []FriendUser
	SearchQuery   string
	Incoming      []FriendRequest
	Friends       []FriendUser
}

func friendsHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		action := r.FormValue("action")
		targetID := r.FormValue("user_id")

		var id int
		_, err := fmt.Sscanf(targetID, "%d", &id)

		if err == nil {
			switch action {
			case "send":
				if id != userID {
					_, _ = db.Exec(`
						INSERT OR IGNORE INTO friend_requests
						(sender_id, receiver_id, status)
						VALUES (?, ?, 'pending')
					`, userID, id)
				}

			case "accept":
				_, _ = db.Exec(`
					UPDATE friend_requests
					SET status = 'accepted'
					WHERE id = ? AND receiver_id = ? AND status = 'pending'
				`, id, userID)

			case "remove":
				_, _ = db.Exec(`
					DELETE FROM friend_requests
					WHERE status = 'accepted'
					AND ((sender_id = ? AND receiver_id = ?)
					OR (sender_id = ? AND receiver_id = ?))
				`, userID, id, id, userID)
			case "reject":
				_, _ = db.Exec(`
					UPDATE friend_requests
					SET status = 'rejected'
					WHERE id = ? AND receiver_id = ? AND status = 'pending'
				`, id, userID)
			}
		}

		http.Redirect(w, r, "/friends", http.StatusSeeOther)
		return
	}

	data := FriendsPageData{}

	search := r.URL.Query().Get("search")
	data.SearchQuery = search

	if search != "" {
		rows, err := db.Query(`
			SELECT id, username, COALESCE(profile_picture, '')
			FROM users
			WHERE id != ?
			AND username LIKE ?
			ORDER BY username
		`, userID, "%"+search+"%")

		if err == nil {
			for rows.Next() {
				var user FriendUser
				if rows.Scan(&user.ID, &user.Username, &user.ProfilePicture) == nil {
					var status string
					statusErr := db.QueryRow(`
                                                SELECT status
                                                FROM friend_requests
                                                WHERE
                                                        (sender_id = ? AND receiver_id = ?)
                                                        OR
                                                        (sender_id = ? AND receiver_id = ?)
                                                ORDER BY id DESC
                                                LIMIT 1
                                        `, userID, user.ID, user.ID, userID).Scan(&status)

					if statusErr == nil {
						if status == "accepted" {
							user.Status = "friends"
						} else if status == "pending" {
							var senderID int
							senderErr := db.QueryRow(`
                                                                SELECT sender_id
                                                                FROM friend_requests
                                                                WHERE
                                                                        ((sender_id = ? AND receiver_id = ?)
                                                                        OR
                                                                        (sender_id = ? AND receiver_id = ?))
                                                                        AND status = 'pending'
                                                                ORDER BY id DESC
                                                                LIMIT 1
                                                        `, userID, user.ID, user.ID, userID).Scan(&senderID)

							if senderErr == nil {
								if senderID == userID {
									user.Status = "sent"
								} else {
									user.Status = "received"
								}
							}
						}
					}

					data.SearchResults = append(data.SearchResults, user)
				}
			}
			rows.Close()
		}
	}

	// People You May Know
	rows, err := db.Query(`
		SELECT id, username, COALESCE(profile_picture, '')
		FROM users
		WHERE id != ?
		AND id NOT IN (
			SELECT receiver_id
			FROM friend_requests
			WHERE sender_id = ?
			AND status IN ('pending', 'accepted')
		)
		AND id NOT IN (
			SELECT sender_id
			FROM friend_requests
			WHERE receiver_id = ?
			AND status IN ('pending', 'accepted')
		)
		ORDER BY username
	`, userID, userID, userID)

	if err == nil {
		for rows.Next() {
			var user FriendUser
			if rows.Scan(&user.ID, &user.Username, &user.ProfilePicture) == nil {
				data.Suggestions = append(data.Suggestions, user)
			}
		}
		rows.Close()
	}

	// Incoming requests
	rows, err = db.Query(`
		SELECT fr.id, u.id, u.username, COALESCE(u.profile_picture, '')
		FROM friend_requests fr
		JOIN users u ON u.id = fr.sender_id
		WHERE fr.receiver_id = ?
		AND fr.status = 'pending'
		ORDER BY fr.created_at DESC
	`, userID)

	if err == nil {
		for rows.Next() {
			var request FriendRequest
			if rows.Scan(
				&request.ID,
				&request.UserID,
				&request.Username,
				&request.ProfilePicture,
			) == nil {
				data.Incoming = append(data.Incoming, request)
			}
		}
		rows.Close()
	}

	// Your friends
	rows, err = db.Query(`
		SELECT u.id, u.username, COALESCE(u.profile_picture, '')
		FROM friend_requests fr
		JOIN users u ON u.id =
			CASE
				WHEN fr.sender_id = ? THEN fr.receiver_id
				ELSE fr.sender_id
			END
		WHERE (fr.sender_id = ? OR fr.receiver_id = ?)
		AND fr.status = 'accepted'
		ORDER BY u.username
	`, userID, userID, userID)

	if err == nil {
		for rows.Next() {
			var friend FriendUser
			if rows.Scan(&friend.ID, &friend.Username, &friend.ProfilePicture) == nil {
				data.Friends = append(data.Friends, friend)
			}
		}
		rows.Close()
	}

	tmpl, err := template.ParseFiles("templates/friends.html")
	if err != nil {
		http.Error(
			w,
			"Unable to load friends page",
			http.StatusInternalServerError,
		)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(
			w,
			"Unable to display friends page",
			http.StatusInternalServerError,
		)
	}
}
