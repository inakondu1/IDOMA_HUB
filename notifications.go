package main

import (
	"html/template"
	"log"
	"net/http"
)

type NotificationRequest struct {
	ID       int
	UserID   int
	Username string
}

type ActivityNotification struct {
	ID        int
	Username  string
	PostID    string
	Type      string
	CreatedAt string
}

type NotificationsPageData struct {
	Activity []ActivityNotification
	Title    string
	Requests []NotificationRequest
	Count    int
}

func notificationsHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		action := r.FormValue("action")
		requestID := r.FormValue("request_id")

		switch action {
		case "accept":
			_, _ = db.Exec(`
				UPDATE friend_requests
				SET status = 'accepted'
				WHERE id = ? AND receiver_id = ? AND status = 'pending'
			`, requestID, userID)

		case "reject":
			_, _ = db.Exec(`
				UPDATE friend_requests
				SET status = 'rejected'
				WHERE id = ? AND receiver_id = ? AND status = 'pending'
			`, requestID, userID)
		}

		http.Redirect(w, r, "/notifications", http.StatusSeeOther)
		return
	}

	rows, err := db.Query(`
		SELECT fr.id, u.id, u.username
		FROM friend_requests fr
		JOIN users u ON u.id = fr.sender_id
		WHERE fr.receiver_id = ?
		AND fr.status = 'pending'
		ORDER BY fr.created_at DESC
	`, userID)

	if err != nil {
		http.Error(w, "Unable to load notifications.", http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer rows.Close()

	var requests []NotificationRequest

	for rows.Next() {
		var request NotificationRequest

		if err := rows.Scan(
			&request.ID,
			&request.UserID,
			&request.Username,
		); err != nil {
			http.Error(w, "Unable to read notifications.", http.StatusInternalServerError)
			log.Println(err)
			return
		}

		requests = append(requests, request)
	}

	activityRows, err := db.Query(`
                SELECT n.id, u.username, n.post_id, n.type, n.created_at
                FROM notifications n
                JOIN users u ON u.id = n.sender_id
                WHERE n.recipient_id = ?
                ORDER BY n.created_at DESC
        `, userID)

	if err != nil {
		http.Error(w, "Unable to load activity notifications.", http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer activityRows.Close()

	var activities []ActivityNotification

	for activityRows.Next() {
		var activity ActivityNotification

		if err := activityRows.Scan(
			&activity.ID,
			&activity.Username,
			&activity.PostID,
			&activity.Type,
			&activity.CreatedAt,
		); err != nil {
			http.Error(w, "Unable to read activity notifications.", http.StatusInternalServerError)
			log.Println(err)
			return
		}

		activities = append(activities, activity)
	}

	data := NotificationsPageData{
		Title:    "Notifications - IDOMA HUB",
		Requests: requests,
		Activity: activities,
		Count:    len(requests),
	}

	tmpl, err := template.ParseFiles("templates/notifications.html")
	if err != nil {
		http.Error(w, "Unable to load notifications page.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Unable to display notifications page.", http.StatusInternalServerError)
		log.Println(err)
	}
}

func createPostNotification(recipientID, senderID int, postID string, notificationType string) {
	if recipientID == senderID {
		return
	}

	_, err := db.Exec(`
                INSERT INTO notifications
                (recipient_id, sender_id, post_id, type)
                VALUES (?, ?, ?, ?)
        `, recipientID, senderID, postID, notificationType)

	if err != nil {
		log.Println("Unable to create notification:", err)
	}
}
