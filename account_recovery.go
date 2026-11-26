package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func recoveryHash(value string) (string, error) {
	secret := os.Getenv("RATE_LIMIT_SECRET")
	if len(secret) < 32 {
		return "", fmt.Errorf("RATE_LIMIT_SECRET must contain at least 32 characters")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(value))

	return hex.EncodeToString(mac.Sum(nil)), nil
}

func allowRecoveryRequest(r *http.Request, action, email string) (bool, error) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false, fmt.Errorf("unable to determine request IP")
	}

	ipHash, err := recoveryHash("ip:" + ip.String())
	if err != nil {
		return false, err
	}

	emailHash, err := recoveryHash("email:" + strings.ToLower(email))
	if err != nil {
		return false, err
	}

	tx, err := db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	checks := []struct {
		scope string
		hash  string
		limit int
	}{
		{action + ":ip", ipHash, 5},
		{action + ":email", emailHash, 3},
	}

	allowed := true

	for _, check := range checks {
		var count int

		err := tx.QueryRow(`
                        INSERT INTO recovery_rate_limits
                                (scope, identifier_hash, window_started_at, request_count)
                        VALUES ($1, $2, CURRENT_TIMESTAMP, 1)
                        ON CONFLICT (scope, identifier_hash)
                        DO UPDATE SET
                                request_count = CASE
                                        WHEN recovery_rate_limits.window_started_at
                                                <= CURRENT_TIMESTAMP - INTERVAL '15 minutes'
                                        THEN 1
                                        ELSE recovery_rate_limits.request_count + 1
                                END,
                                window_started_at = CASE
                                        WHEN recovery_rate_limits.window_started_at
                                                <= CURRENT_TIMESTAMP - INTERVAL '15 minutes'
                                        THEN CURRENT_TIMESTAMP
                                        ELSE recovery_rate_limits.window_started_at
                                END
                        RETURNING request_count
                `, check.scope, check.hash).Scan(&count)

		if err != nil {
			return false, err
		}

		if count > check.limit {
			allowed = false
		}
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}

	return allowed, nil
}

func sendRecoveryEmail(recipient, subject, message string) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	username := os.Getenv("SMTP_USERNAME")
	password := os.Getenv("SMTP_PASSWORD")

	if host == "" || port == "" || username == "" || password == "" {
		return fmt.Errorf("SMTP configuration is incomplete")
	}

	to, err := mail.ParseAddress(recipient)
	if err != nil {
		return fmt.Errorf("invalid recipient address")
	}

	from, err := mail.ParseAddress(username)
	if err != nil {
		return fmt.Errorf("invalid sender address")
	}

	auth := smtp.PlainAuth("", username, password, host)

	email := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from.Address,
		to.Address,
		subject,
		message,
	)

	return smtp.SendMail(
		host+":"+port,
		auth,
		from.Address,
		[]string{to.Address},
		[]byte(email),
	)
}

func renderRecoveryPage(w http.ResponseWriter, filename string, data PageData) {
	tmpl, err := template.ParseFiles("templates/" + filename)
	if err != nil {
		log.Println("Recovery template error:", err)
		http.Error(w, "Unable to load page.", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Println("Recovery page error:", err)
	}
}

func forgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		renderRecoveryPage(w, "forgot_password.html", PageData{
			Title: "Forgot Password - IDOMA HUB",
			Error: map[string]string{
				"invalid": "Please enter a valid email address.",
			}[r.URL.Query().Get("error")],
		})
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !validSameOriginRequest(r) {
		http.Error(w, "Invalid request origin.", http.StatusForbidden)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))

	parsedEmail, parseErr := mail.ParseAddress(email)
	if email == "" || parseErr != nil || parsedEmail.Address != email {
		http.Redirect(w, r, "/forgot-password?error=invalid", http.StatusSeeOther)
		return
	}

	allowed, limitErr := allowRecoveryRequest(r, "forgot_password", email)
	if limitErr != nil {
		log.Println("Password recovery rate-limit error:", limitErr)
		http.Error(w, "Recovery is temporarily unavailable.", http.StatusServiceUnavailable)
		return
	}
	if !allowed {
		w.Header().Set("Retry-After", "900")
		w.WriteHeader(http.StatusTooManyRequests)
		renderRecoveryPage(w, "forgot_password.html", PageData{
			Title: "Forgot Password - IDOMA HUB",
			Error: "Too many requests. Please try again in 15 minutes.",
		})
		return
	}

	var userID int

	err := db.QueryRow(`
		SELECT id FROM users
		WHERE LOWER(email) = LOWER($1)
	`, email).Scan(&userID)

	if err == nil {
		token, tokenErr := createPasswordResetToken(userID)

		if tokenErr != nil {
			log.Println("Password reset token creation failed:", tokenErr)
		} else {
			baseURL := strings.TrimRight(os.Getenv("APP_BASE_URL"), "/")

			parsed, parseErr := url.Parse(baseURL)

			if parseErr != nil || parsed.Host == "" ||
				parsed.Scheme != "https" {
				log.Println("APP_BASE_URL is missing or invalid")
			} else {
				resetLink := baseURL + "/reset-password?token=" +
					url.QueryEscape(token)

				message := "Hello,\n\n" +
					"We received a request to reset your IDOMA HUB password.\n\n" +
					"Use this link to create a new password:\n" +
					resetLink + "\n\n" +
					"This link expires in 30 minutes and can only be used once.\n\n" +
					"If you did not request this, you can ignore this email.\n\n" +
					"IDOMA HUB Team"

				if sendErr := sendRecoveryEmail(
					email,
					"IDOMA HUB Password Reset",
					message,
				); sendErr != nil {
					log.Println("Password recovery email could not be sent:", sendErr)
				}
			}
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		log.Println("Password recovery lookup failed:", err)
	}

	// Same response whether the email exists or not.
	renderRecoveryPage(w, "forgot_password.html", PageData{
		Title:      "Forgot Password - IDOMA HUB",
		Registered: true,
	})
}

func forgotUsernameHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		renderRecoveryPage(w, "forgot_username.html", PageData{
			Title: "Forgot Username - IDOMA HUB",
		})
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !validSameOriginRequest(r) {
		http.Error(w, "Invalid request origin.", http.StatusForbidden)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))

	parsedEmail, parseErr := mail.ParseAddress(email)
	if email == "" || parseErr != nil || parsedEmail.Address != email {
		renderRecoveryPage(w, "forgot_username.html", PageData{
			Title: "Forgot Username - IDOMA HUB",
			Error: "Please enter your registered email address.",
		})
		return
	}

	allowed, limitErr := allowRecoveryRequest(r, "forgot_username", email)
	if limitErr != nil {
		log.Println("Username recovery rate-limit error:", limitErr)
		http.Error(w, "Recovery is temporarily unavailable.", http.StatusServiceUnavailable)
		return
	}
	if !allowed {
		w.Header().Set("Retry-After", "900")
		w.WriteHeader(http.StatusTooManyRequests)
		renderRecoveryPage(w, "forgot_username.html", PageData{
			Title: "Forgot Username - IDOMA HUB",
			Error: "Too many requests. Please try again in 15 minutes.",
		})
		return
	}

	var username string

	err := db.QueryRow(`
		SELECT username FROM users
		WHERE LOWER(email) = LOWER($1)
	`, email).Scan(&username)

	if err == nil {
		message := "Hello,\n\n" +
			"Your IDOMA HUB username is: " + username + "\n\n" +
			"If you did not request this reminder, you can ignore this email.\n\n" +
			"IDOMA HUB Team"

		if sendErr := sendRecoveryEmail(
			email,
			"Your IDOMA HUB Username",
			message,
		); sendErr != nil {
			log.Println("Username recovery email could not be sent:", sendErr)
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		log.Println("Username recovery lookup failed:", err)
	}

	renderRecoveryPage(w, "forgot_username.html", PageData{
		Title:      "Forgot Username - IDOMA HUB",
		Registered: true,
	})
}

func resetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		token := r.URL.Query().Get("token")

		_, err := getPasswordResetUserID(token)
		if token == "" || err != nil {
			renderRecoveryPage(w, "reset_password.html", PageData{
				Title: "Reset Password - IDOMA HUB",
				Error: "This reset link is invalid or has expired. Please request another one.",
			})
			return
		}

		tmpl, err := template.ParseFiles("templates/reset_password.html")
		if err != nil {
			log.Println("Reset template error:", err)
			http.Error(w, "Unable to load page.", http.StatusInternalServerError)
			return
		}

		data := struct {
			Title string
			Token string
			Error string
		}{
			Title: "Reset Password - IDOMA HUB",
			Token: token,
		}

		if err := tmpl.Execute(w, data); err != nil {
			log.Println("Reset page error:", err)
		}
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !validSameOriginRequest(r) {
		http.Error(w, "Invalid request origin.", http.StatusForbidden)
		return
	}

	token := r.FormValue("token")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if len(password) < 8 {
		renderRecoveryPage(w, "reset_password.html", PageData{
			Title: "Reset Password - IDOMA HUB",
			Token: token,
			Error: "Your password must contain at least 8 characters.",
		})
		return
	}

	if password != confirmPassword {
		renderRecoveryPage(w, "reset_password.html", PageData{
			Title: "Reset Password - IDOMA HUB",
			Token: token,
			Error: "Your passwords do not match.",
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		log.Println("Password hashing failed:", err)
		http.Error(w, "Unable to secure your password.", http.StatusInternalServerError)
		return
	}

	userID, err := resetPasswordWithToken(token, string(hashedPassword))
	if err != nil {
		renderRecoveryPage(w, "reset_password.html", PageData{
			Title: "Reset Password - IDOMA HUB",
			Error: "This reset link is invalid or has expired. Please request another one.",
		})
		return
	}

	deleteUserSessions(userID)

	http.Redirect(w, r, "/login?reset=1", http.StatusSeeOther)
}

// cleanupRecoveryRateLimits removes expired rate-limit records.
func cleanupRecoveryRateLimits() {
        _, err := db.Exec(`
                DELETE FROM recovery_rate_limits
                WHERE window_started_at <= CURRENT_TIMESTAMP - INTERVAL '24 hours'
        `)
        if err != nil {
                log.Println("Unable to clean recovery rate limits:", err)
        }
}
