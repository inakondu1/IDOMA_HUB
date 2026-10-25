package main

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type Message struct {
	ID             int
	SenderID       int
	ReceiverID     int
	Content        string
	AudioURL       string
	IsVoice        bool
	VideoURL       string
	IsVideo        bool
	IsRead         int
	CreatedAt      string
	ReplyToID      int
	ReplyToContent string
	ReplyToIsVoice bool
	ReplyToIsVideo bool
}

type ForwardRecipient struct {
	UserID   int
	Username string
}

type MessagesPageData struct {
	OtherUserID             int
	OtherUsername           string
	OtherUserProfilePicture string
	Messages                []Message
	Recipients              []ForwardRecipient
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
		forwardMessageID, _ := strconv.Atoi(r.FormValue("forward_message_id"))
		if forwardMessageID > 0 {
			forwardToUserID, parseErr := strconv.Atoi(r.FormValue("forward_to_user_id"))
			if parseErr != nil || forwardToUserID <= 0 || forwardToUserID == userID {
				http.Error(w, "Invalid forwarding recipient", http.StatusBadRequest)
				return
			}

			var forwardedContent string
			err := db.QueryRow(`
				SELECT content
				FROM messages
				WHERE id = $1
				  AND (
					(sender_id = $2 AND receiver_id = $3)
					OR
					(sender_id = $3 AND receiver_id = $2)
				  )
			`, forwardMessageID, userID, otherUserID).Scan(&forwardedContent)

			if err != nil {
				http.Error(w, "Message to forward not found", http.StatusBadRequest)
				return
			}

			var recipientExists bool
			err = db.QueryRow(`
				SELECT EXISTS (
					SELECT 1 FROM users WHERE id = $1
				)
			`, forwardToUserID).Scan(&recipientExists)

			if err != nil {
				log.Println("Unable to validate forwarding recipient:", err)
				http.Error(w, "Unable to validate forwarding recipient", http.StatusInternalServerError)
				return
			}
			if !recipientExists {
				http.Error(w, "Forwarding recipient not found", http.StatusBadRequest)
				return
			}

			_, err = db.Exec(`
				INSERT INTO messages (sender_id, receiver_id, content)
				VALUES ($1, $2, $3)
			`, userID, forwardToUserID, forwardedContent)

			if err != nil {
				log.Println("Unable to forward message:", err)
				http.Error(w, "Unable to forward message", http.StatusInternalServerError)
				return
			}

			http.Redirect(
				w,
				r,
				"/messages?user_id="+strconv.Itoa(forwardToUserID),
				http.StatusSeeOther,
			)
			return
		}

		content := strings.TrimSpace(r.FormValue("content"))
		replyToID, _ := strconv.Atoi(r.FormValue("reply_to_id"))

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

			hasVoice := r.MultipartForm != nil && len(r.MultipartForm.File["voice"]) > 0
			hasVideo := r.MultipartForm != nil && len(r.MultipartForm.File["video"]) > 0

			if hasVoice && hasVideo {
				http.Error(w, "Send a video separately from a voice message", http.StatusBadRequest)
				return
			}

			if (hasVoice || hasVideo) && content != "" {
				http.Error(w, "Send recordings separately from text messages", http.StatusBadRequest)
				return
			}

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
			videoFile, videoHeader, videoErr := r.FormFile("video")
			if videoErr == nil {
				defer videoFile.Close()

				if content != "" {
					http.Error(w, "Send a video separately from text or voice", http.StatusBadRequest)
					return
				}

				if videoHeader.Size > 10<<20 {
					http.Error(w, "Video message must be 10 MB or smaller", http.StatusBadRequest)
					return
				}

				if !strings.HasPrefix(videoHeader.Header.Get("Content-Type"), "video/") {
					http.Error(w, "Unsupported video-message format", http.StatusBadRequest)
					return
				}

				videoURL, uploadErr := uploadToCloudinary(
					videoFile,
					"idoma_hub/video_messages",
					"video",
				)
				if uploadErr != nil {
					log.Println("Unable to upload video message:", uploadErr)
					http.Error(w, "Unable to upload video message", http.StatusInternalServerError)
					return
				}

				content = "VIDEO:" + videoURL
			} else if videoErr != http.ErrMissingFile {
				http.Error(w, "Unable to read video message", http.StatusBadRequest)
				return
			}
		}

		replyToID, _ = strconv.Atoi(r.FormValue("reply_to_id"))

		if replyToID != 0 {
			var validReply int
			err := db.QueryRow(`
				SELECT COUNT(*)
				FROM messages
				WHERE id = $1
				  AND (
					(sender_id = $2 AND receiver_id = $3)
					OR
					(sender_id = $3 AND receiver_id = $2)
				  )
			`, replyToID, userID, otherUserID).Scan(&validReply)

			if err != nil || validReply == 0 {
				http.Error(w, "Invalid message to reply to", http.StatusBadRequest)
				return
			}
		}

		if content != "" {
			_, err = db.Exec(`
                INSERT INTO messages
				(sender_id, receiver_id, content, reply_to_id)
				VALUES ($1, $2, $3, NULLIF($4, 0))
			`, userID, otherUserID, content, replyToID)

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
		SELECT m.id, m.sender_id, m.receiver_id, m.content, m.is_read,
                   TO_CHAR(m.created_at, 'YYYY-MM-DD HH24:MI'),
                   COALESCE(m.reply_to_id, 0),
                   COALESCE(replied.content, '')
            FROM messages m
            LEFT JOIN messages replied ON replied.id = m.reply_to_id
            WHERE
                    (m.sender_id = $1 AND m.receiver_id = $2)
                    OR
                    (m.sender_id = $2 AND m.receiver_id = $1)
            ORDER BY m.created_at ASC
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

	recipientRows, recipientErr := db.Query(`
		SELECT id, username
		FROM users
		WHERE id <> $1
		ORDER BY username
	`, userID)
	if recipientErr != nil {
		log.Println("Unable to load forwarding recipients:", recipientErr)
		http.Error(w, "Unable to load forwarding recipients", http.StatusInternalServerError)
		return
	}

	for recipientRows.Next() {
		var recipient ForwardRecipient
		if err := recipientRows.Scan(&recipient.UserID, &recipient.Username); err != nil {
			recipientRows.Close()
			http.Error(w, "Unable to read forwarding recipients", http.StatusInternalServerError)
			return
		}
		data.Recipients = append(data.Recipients, recipient)
	}

	if err := recipientRows.Err(); err != nil {
		recipientRows.Close()
		log.Println("Unable to read forwarding recipients:", err)
		http.Error(w, "Unable to read forwarding recipients", http.StatusInternalServerError)
		return
	}
	recipientRows.Close()

	for rows.Next() {
		var message Message

		if err := rows.Scan(
			&message.ID,
			&message.SenderID,
			&message.ReceiverID,
			&message.Content,
			&message.IsRead,
			&message.CreatedAt,
			&message.ReplyToID,
			&message.ReplyToContent,
		); err == nil {
			if strings.HasPrefix(message.Content, "VOICE:") {
				message.IsVoice = true
				message.AudioURL = strings.TrimPrefix(message.Content, "VOICE:")
			}

			if strings.HasPrefix(message.Content, "VIDEO:") {
				message.IsVideo = true
				message.VideoURL = strings.TrimPrefix(message.Content, "VIDEO:")
			}
			if strings.HasPrefix(message.ReplyToContent, "VOICE:") {
				message.ReplyToIsVoice = true
				message.ReplyToContent = "🎤 Voice message"
			}

			if strings.HasPrefix(message.ReplyToContent, "VIDEO:") {
				message.ReplyToIsVideo = true
				message.ReplyToContent = "🎬 Video message"
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
