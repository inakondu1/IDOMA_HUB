package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"html/template"
	"net/http"
	"strings"
	"time"
)

const anonymousCookieName = "idoma_anonymous"

var anonymousAdjectives = []string{
	"Quiet",
	"Hidden",
	"Silent",
	"Kind",
	"Calm",
	"Brave",
	"Wise",
	"Free",
	"Bright",
	"Peaceful",
}

var anonymousNouns = []string{
	"Star",
	"River",
	"Tree",
	"Moon",
	"Bird",
	"Cloud",
	"Stone",
	"Leaf",
	"Light",
	"Voice",
}

type AnonymousMessage struct {
	ID            int
	AnonymousName string
	Content       string
	CreatedAt     string
}

type AnonymousPageData struct {
	AnonymousName string
	OnlineCount   int
	Messages      []AnonymousMessage
}

func anonymousRandomBytes(length int) ([]byte, error) {
	b := make([]byte, length)

	if _, err := rand.Read(b); err != nil {
		return nil, err
	}

	return b, nil
}

func anonymousSessionHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func generateAnonymousName() string {
	random, err := anonymousRandomBytes(4)

	if err != nil {
		return "Anonymous"
	}

	adj := anonymousAdjectives[int(random[0])%len(anonymousAdjectives)]
	noun := anonymousNouns[int(random[1])%len(anonymousNouns)]
	number := int(random[2])*256 + int(random[3])

	return adj + " " + noun + " " + formatAnonymousNumber(number)
}

func formatAnonymousNumber(number int) string {
	return strings.TrimSpace(
		template.HTMLEscapeString(
			strings.TrimSpace(
				formatNumber(number),
			),
		),
	)
}

func formatNumber(number int) string {
	if number < 1000 {
		return strings.Repeat("0", 4-len(intToString(number))) + intToString(number)
	}

	return intToString(number)
}

func intToString(number int) string {
	if number == 0 {
		return "0"
	}

	digits := ""

	for number > 0 {
		digits = string(rune('0'+number%10)) + digits
		number /= 10
	}

	return digits
}

func getAnonymousSession(w http.ResponseWriter, r *http.Request) (string, string, error) {
	cookie, err := r.Cookie(anonymousCookieName)

	if err == nil && cookie.Value != "" {
		sessionHash := anonymousSessionHash(cookie.Value)

		var anonymousName string
		var expiresAt time.Time

		err = db.QueryRow(`
			SELECT anonymous_name, expires_at
			FROM anonymous_sessions
			WHERE session_hash = $1
			  AND expires_at > CURRENT_TIMESTAMP
		`, sessionHash).Scan(&anonymousName, &expiresAt)

		if err == nil {
			return sessionHash, anonymousName, nil
		}
	}

	tokenBytes, err := anonymousRandomBytes(32)

	if err != nil {
		return "", "", err
	}

	token := hex.EncodeToString(tokenBytes)
	sessionHash := anonymousSessionHash(token)
	anonymousName := generateAnonymousName()
	expiresAt := time.Now().Add(24 * time.Hour)

	_, err = db.Exec(`
		INSERT INTO anonymous_sessions (
			session_hash,
			anonymous_name,
			expires_at
		)
		VALUES ($1, $2, $3)
		ON CONFLICT (session_hash) DO NOTHING
	`, sessionHash, anonymousName, expiresAt)

	if err != nil {
		return "", "", err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     anonymousCookieName,
		Value:    token,
		Path:     "/anonymous",
		Expires:  expiresAt,
		MaxAge:   60 * 60 * 24,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})

	return sessionHash, anonymousName, nil
}

func cleanupAnonymousData() {
	_, _ = db.Exec(`
		DELETE FROM anonymous_views
		WHERE message_id IN (
			SELECT id
			FROM anonymous_messages
			WHERE expires_at <= CURRENT_TIMESTAMP
		)
	`)

	_, _ = db.Exec(`
		DELETE FROM anonymous_messages
		WHERE expires_at <= CURRENT_TIMESTAMP
	`)

	_, _ = db.Exec(`
		DELETE FROM anonymous_sessions
		WHERE expires_at <= CURRENT_TIMESTAMP
	`)

	_, _ = db.Exec(`
		DELETE FROM anonymous_presence
		WHERE expires_at <= CURRENT_TIMESTAMP
	`)
}

func anonymousOnlineCount() int {
	var count int

	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM anonymous_presence
		WHERE expires_at > CURRENT_TIMESTAMP
	`).Scan(&count)

	if err != nil {
		return 0
	}

	return count
}

func anonymousHandler(w http.ResponseWriter, r *http.Request) {
	// Authentication is required, but the real account ID is deliberately
	// discarded and is never written into Anonymous data.
	_, loggedIn := getUserIDFromSession(r)

	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	cleanupAnonymousData()

	sessionHash, anonymousName, err := getAnonymousSession(w, r)

	if err != nil {
		http.Error(w, "Unable to create Anonymous session", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(`
		INSERT INTO anonymous_presence (session_hash, expires_at)
		VALUES ($1, CURRENT_TIMESTAMP + INTERVAL '2 minutes')
		ON CONFLICT (session_hash)
		DO UPDATE SET expires_at = CURRENT_TIMESTAMP + INTERVAL '2 minutes'
	`, sessionHash)

	if err != nil {
		http.Error(w, "Unable to update Anonymous presence", http.StatusInternalServerError)
		return
	}

	if r.Method == http.MethodPost {
		content := strings.TrimSpace(r.FormValue("content"))

		if content == "" {
			http.Redirect(w, r, "/anonymous", http.StatusSeeOther)
			return
		}

		_, err = db.Exec(`
			INSERT INTO anonymous_messages (
				anonymous_name,
				content,
				expires_at
			)
			VALUES (
				$1,
				$2,
				CURRENT_TIMESTAMP + INTERVAL '48 hours'
			)
		`, anonymousName, content)

		if err != nil {
			http.Error(w, "Unable to post anonymously", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/anonymous", http.StatusSeeOther)
		return
	}

	rows, err := db.Query(`
		SELECT
			id,
			anonymous_name,
			content,
			TO_CHAR(created_at, 'Mon DD, YYYY HH24:MI')
		FROM anonymous_messages
		WHERE expires_at > CURRENT_TIMESTAMP
		ORDER BY created_at ASC
	`)

	if err != nil {
		http.Error(w, "Unable to load Anonymous messages", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	data := AnonymousPageData{
		AnonymousName: anonymousName,
		OnlineCount:   anonymousOnlineCount(),
	}

	for rows.Next() {
		var message AnonymousMessage

		if err := rows.Scan(
			&message.ID,
			&message.AnonymousName,
			&message.Content,
			&message.CreatedAt,
		); err == nil {
			data.Messages = append(data.Messages, message)
		}
	}

	tmpl, err := template.ParseFiles("templates/anonymous.html")

	if err != nil {
		http.Error(w, "Unable to load Anonymous page", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Unable to display Anonymous page", http.StatusInternalServerError)
		return
	}
}
