package main

import (
	"html/template"
	"net/http"
	"strconv"
)

type Message struct {
	ID         int
	SenderID   int
	ReceiverID int
	Content    string
	IsRead     int
	CreatedAt  string
}

type MessagesPageData struct {
	OtherUserID   int
	OtherUsername string
	Messages      []Message
}

func messagesHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	otherUserID, err := strconv.Atoi(r.URL.Query().Get("user_id"))
	if err != nil || otherUserID <= 0 || otherUserID == userID {
		http.Error(w, "Invalid user", http.StatusBadRequest)
		return
	}

	var otherUsername string
	err = db.QueryRow(`
		SELECT username
		FROM users
		WHERE id = $1
	`, otherUserID).Scan(&otherUsername)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if r.Method == http.MethodPost {
		content := r.FormValue("content")

		if content != "" {
			_, err = db.Exec(`
				INSERT INTO messages
				(sender_id, receiver_id, content)
				VALUES ($1, $2, $3)
			`, userID, otherUserID, content)

			if err != nil {
				http.Error(w, "Unable to send message", http.StatusInternalServerError)
				return
			}
		}

		http.Redirect(
			w,
			r,
			"/messages?user_id="+strconv.Itoa(otherUserID),
			http.StatusSeeOther,
		)
		return
	}

	rows, err := db.Query(`
		SELECT id, sender_id, receiver_id, content, is_read,
		       TO_CHAR(created_at, 'YYYY-MM-DD HH24:MI')
		FROM messages
		WHERE
			(sender_id = $1 AND receiver_id = $2)
			OR
			(sender_id = $2 AND receiver_id = $1)
		ORDER BY created_at ASC
	`, userID, otherUserID)

	if err != nil {
		http.Error(w, "Unable to load messages", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	data := MessagesPageData{
		OtherUserID:   otherUserID,
		OtherUsername: otherUsername,
	}

	for rows.Next() {
		var message Message

		if err := rows.Scan(
			&message.ID,
			&message.SenderID,
			&message.ReceiverID,
			&message.Content,
			&message.IsRead,
			&message.CreatedAt,
		); err == nil {
			data.Messages = append(data.Messages, message)
		}
	}

	_, _ = db.Exec(`
		UPDATE messages
		SET is_read = 1
		WHERE sender_id = $1
		AND receiver_id = $2
		AND is_read = 0
	`, otherUserID, userID)

	tmpl, err := template.ParseFiles("templates/messages.html")
	if err != nil {
		http.Error(w, "Unable to load messages page", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Unable to display messages page", http.StatusInternalServerError)
	}
}
