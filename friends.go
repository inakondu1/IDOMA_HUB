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
	Following      bool
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
	Followers     []FriendUser
	Following     []FriendUser
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
						INSERT INTO friend_requests
						(sender_id, receiver_id, status)
						VALUES ($1, $2, 'pending')
                                                ON CONFLICT DO NOTHING
					`, userID, id)
				}

			case "accept":
				_, _ = db.Exec(`
					UPDATE friend_requests
					SET status = 'accepted'
					WHERE id = $1 AND receiver_id = $2 AND status = 'pending'
				`, id, userID)

			case "remove":
				_, _ = db.Exec(`
					DELETE FROM friend_requests
					WHERE status = 'accepted'
					AND ((sender_id = $1 AND receiver_id = $2)
					OR (sender_id = $3 AND receiver_id = $4))
				`, userID, id, id, userID)
			case "reject":
				_, _ = db.Exec(`
					UPDATE friend_requests
					SET status = 'rejected'
					WHERE id = $1 AND receiver_id = $2 AND status = 'pending'
				`, id, userID)
                        case "follow":
                                if id != userID {
                                        _, _ = db.Exec(`
                                                INSERT INTO follows (follower_id, following_id)
                                                SELECT $1, id
                                                FROM users
                                                WHERE id = $2
                                                ON CONFLICT (follower_id, following_id) DO NOTHING
                                        `, userID, id)
                                }

                        case "unfollow":
                                if id != userID {
                                        _, _ = db.Exec(`
                                                DELETE FROM follows
                                                WHERE follower_id = $1
                                                AND following_id = $2
                                        `, userID, id)
                                }
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
			WHERE id != $1
			AND username LIKE $2
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
                                                        (sender_id = $1 AND receiver_id = $2)
                                                        OR
                                                        (sender_id = $2 AND receiver_id = $1)
                                                ORDER BY id DESC
                                                LIMIT 1
                                        `, userID, user.ID).Scan(&status)

					if statusErr == nil {
						if status == "accepted" {
							user.Status = "friends"
						} else if status == "pending" {
							var senderID int
							senderErr := db.QueryRow(`
                                                                SELECT sender_id
                                                                FROM friend_requests
                                                                WHERE
                                                                        ((sender_id = $1 AND receiver_id = $2)
                                                                        OR
                                                                        (sender_id = $2 AND receiver_id = $1))
                                                                        AND status = 'pending'
                                                                ORDER BY id DESC
                                                                LIMIT 1
                                                        `, userID, user.ID).Scan(&senderID)

							if senderErr == nil {
								if senderID == userID {
									user.Status = "sent"
								} else {
									user.Status = "received"
								}
							}
						}
					}


                                        var following int
                                        followErr := db.QueryRow(`
                                                SELECT 1
                                                FROM follows
                                                WHERE follower_id = $1
                                                AND following_id = $2
                                                LIMIT 1
                                        `, userID, user.ID).Scan(&following)

                                        if followErr == nil {
                                                user.Following = true
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
		WHERE id != $1
		AND id NOT IN (
			SELECT receiver_id
			FROM friend_requests
			WHERE sender_id = $1
			AND status IN ('pending', 'accepted')
		)
		AND id NOT IN (
			SELECT sender_id
			FROM friend_requests
			WHERE receiver_id = $1
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
		WHERE fr.receiver_id = $1
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
				WHEN fr.sender_id = $1 THEN fr.receiver_id
				ELSE fr.sender_id
			END
		WHERE (fr.sender_id = $2 OR fr.receiver_id = $3)
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
	// People you are following
	rows, err = db.Query(`
            SELECT u.id, u.username, COALESCE(u.profile_picture, '')
            FROM follows f
            JOIN users u ON u.id = f.following_id
            WHERE f.follower_id = $1
            ORDER BY u.username
    `, userID)

	if err == nil {
		for rows.Next() {
			var person FriendUser
			if rows.Scan(&person.ID, &person.Username, &person.ProfilePicture) == nil {
				data.Following = append(data.Following, person)
			}
		}
		rows.Close()
	}

	// People who follow you
	rows, err = db.Query(`
            SELECT u.id, u.username, COALESCE(u.profile_picture, '')
            FROM follows f
            JOIN users u ON u.id = f.follower_id
            WHERE f.following_id = $1
            ORDER BY u.username
    `, userID)

	if err == nil {
		for rows.Next() {
			var person FriendUser
			if rows.Scan(&person.ID, &person.Username, &person.ProfilePicture) == nil {
				data.Followers = append(data.Followers, person)
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
