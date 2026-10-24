package main

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type Message struct {
	ID         int
	SenderID   int
	ReceiverID int
	Content    string
	AudioURL   string
	IsVoice    bool
	IsRead     int
	CreatedAt  string
}

type MessagesPageData struct {
	OtherUserID             int
	OtherUsername           string
	OtherUserProfilePicture string
	Messages                []Message
}

type Conversation struct {
	UserID         int
	Username       string
	ProfilePicture string
	LastMessage    string
	CreatedAt      string
	UnreadCount    int
}

type ConversationsPageData struct {
	Conversations []Conversation
}

func messagesHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	userIDParam := r.URL.Query().Get("user_id")

	if userIDParam == "" {
		rows, err := db.Query(`
                        SELECT user_id, username, profile_picture, last_message, created_at, unread_count
                        FROM (
                                SELECT DISTINCT ON (u.id)
                                        u.id AS user_id,
                                        u.username,
                                        COALESCE(u.profile_picture, '') AS profile_picture,
                                        m.content AS last_message,
                                        TO_CHAR(m.created_at, 'YYYY-MM-DD HH24:MI') AS created_at,
                                        (
                                                SELECT COUNT(*)
                                                FROM messages unread
                                                WHERE unread.sender_id = u.id
                                                AND unread.receiver_id = $1
                                                AND unread.is_read = 0
                                        ) AS unread_count
                                FROM users u
                                JOIN messages m
                                        ON (m.sender_id = $1 AND m.receiver_id = u.id)
                                        OR (m.sender_id = u.id AND m.receiver_id = $1)
                                WHERE u.id <> $1
                                ORDER BY u.id, m.created_at DESC
                        ) conversations
                        ORDER BY created_at DESC
                `, userID)

		if err != nil {
			http.Error(w, "Unable to load conversations", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		data := ConversationsPageData{}

		for rows.Next() {
			var conversation Conversation

			if err := rows.Scan(
				&conversation.UserID,
				&conversation.Username,
				&conversation.ProfilePicture,
				&conversation.LastMessage,
				&conversation.CreatedAt,
				&conversation.UnreadCount,
			); err != nil {
				http.Error(w, "Unable to read conversations", http.StatusInternalServerError)
				return
			}

			data.Conversations = append(data.Conversations, conversation)
		}

		tmpl, err := template.ParseFiles("templates/conversations.html")
		if err != nil {
			http.Error(w, "Unable to load conversations page", http.StatusInternalServerError)
			return
		}

		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, "Unable to display conversations page", http.StatusInternalServerError)
		}

		return
	}

	otherUserID, err := strconv.Atoi(r.URL.Query().Get("user_id"))
	if err != nil || otherUserID <= 0 || otherUserID == userID {
		http.Error(w, "Invalid user", http.StatusBadRequest)
		return
	}

	var otherUsername string
	var otherUserProfilePicture string
	err = db.QueryRow(`
		SELECT username, COALESCE(profile_picture, '')
		FROM users
		WHERE id = $1
	`, otherUserID).Scan(&otherUsername, &otherUserProfilePicture)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if r.Method == http.MethodPost {
		content := strings.TrimSpace(r.FormValue("content"))

		isMultipart := strings.HasPrefix(
			r.Header.Get("Content-Type"),
			"multipart/form-data",
		)

		if isMultipart {
			r.Body = http.MaxBytesReader(w, r.Body, 12<<20)

			if err := r.ParseMultipartForm(10 << 20); err != nil {
				http.Error(w, "Unable to process message or recording", http.StatusBadRequest)
				return
			}

			content = strings.TrimSpace(r.FormValue("content"))

			voiceFile, voiceHeader, voiceErr := r.FormFile("voice")
			if voiceErr == nil {
				defer voiceFile.Close()

				if voiceHeader.Size > 10<<20 {
					http.Error(w, "Voice message must be 10 MB or smaller", http.StatusBadRequest)
					return
				}

				if !strings.HasPrefix(voiceHeader.Header.Get("Content-Type"), "audio/") {
					http.Error(w, "Unsupported voice-message format", http.StatusBadRequest)
					return
				}

				audioURL, uploadErr := uploadToCloudinary(
					voiceFile,
					"idoma_hub/voice_messages",
					"video",
				)
				if uploadErr != nil {
					log.Println("Unable to upload voice message:", uploadErr)
					http.Error(w, "Unable to upload voice message", http.StatusInternalServerError)
					return
				}

				content = "VOICE:" + audioURL
			} else if voiceErr != http.ErrMissingFile {
				http.Error(w, "Unable to read voice message", http.StatusBadRequest)
				return
			}
		}

		if content != "" {
			_, err = db.Exec(`
                INSERT INTO messages
                (sender_id, receiver_id, content)
                VALUES ($1, $2, $3)
            `, userID, otherUserID, content)

			if err != nil {
				log.Println("Unable to save message:", err)
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
		OtherUserID:             otherUserID,
		OtherUsername:           otherUsername,
		OtherUserProfilePicture: otherUserProfilePicture,
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
			if strings.HasPrefix(message.Content, "VOICE:") {
				message.IsVoice = true
				message.AudioURL = strings.TrimPrefix(message.Content, "VOICE:")
			}
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
