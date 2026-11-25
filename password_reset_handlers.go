package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// forgotPasswordHandler handles password reset requests.
func forgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	if r.Method == http.MethodGet {
		renderForgotPasswordPage(w, PageData{
			Title: "Forgot Password - IDOMA HUB",
		})
		return
	}

	if !validSameOriginRequest(r) {
		http.Error(w, "Invalid request origin.", http.StatusForbidden)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))

	if email != "" {
		baseURL := strings.TrimRight(os.Getenv("APP_BASE_URL"), "/")

		smtpReady := os.Getenv("SMTP_HOST") != "" &&
			os.Getenv("SMTP_PORT") != "" &&
			os.Getenv("SMTP_USER") != "" &&
			os.Getenv("SMTP_PASSWORD") != "" &&
			os.Getenv("SMTP_FROM") != ""

		if baseURL == "" || !strings.HasPrefix(baseURL, "https://") {
			log.Println("Password reset email skipped: APP_BASE_URL is missing or invalid")
		} else if !smtpReady {
			log.Println("Password reset email skipped: SMTP configuration is incomplete")
		} else {
			var userID int

			err := db.QueryRow(
				"SELECT id FROM users WHERE LOWER(email) = LOWER($1)",
				email,
			).Scan(&userID)

			if err == nil {
				token, err := createPasswordResetToken(userID)

				if err != nil {
					log.Println("Password reset token creation failed:", err)
				} else {
					resetLink := baseURL + "/reset-password?token=" + token

					if err := sendPasswordResetEmail(email, resetLink); err != nil {
						log.Println("Password reset email delivery failed:", err)

						if cleanupErr := deletePasswordResetToken(token); cleanupErr != nil {
							log.Println("Password reset token cleanup failed:", cleanupErr)
						}
					} else {
						if err := invalidateOtherPasswordResetTokens(userID, token); err != nil {
							log.Println("Previous password reset token invalidation failed:", err)
						}
					}
				}
			} else if !errors.Is(err, sql.ErrNoRows) {
				log.Println("Password reset account lookup failed:", err)
			}
		}
	}

	renderForgotPasswordPage(w, PageData{
		Title:   "Forgot Password - IDOMA HUB",
		Message: "If an account exists with that email address, a password reset link will be sent shortly.",
	})
}

func renderForgotPasswordPage(w http.ResponseWriter, data PageData) {
	tmpl, err := template.ParseFiles("templates/forgot_password.html")
	if err != nil {
		log.Println("Unable to load forgot password page:", err)
		http.Error(w, "Unable to load page.", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Println("Unable to display forgot password page:", err)
	}
}

// resetPasswordHandler displays the reset form and securely updates passwords.
func resetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	if r.Method == http.MethodPost && !validSameOriginRequest(r) {
		http.Error(w, "Invalid request origin.", http.StatusForbidden)
		return
	}

	token := r.URL.Query().Get("token")

	if r.Method == http.MethodPost {
		token = r.FormValue("token")
	}

	token = strings.TrimSpace(token)

	if token == "" {
		http.Error(w, "Invalid or expired password reset link.", http.StatusBadRequest)
		return
	}

	_, err := getPasswordResetUserID(token)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Println("Password reset token lookup failed:", err)
		}

		renderResetPasswordPage(w, PageData{
			Title: "Reset Password - IDOMA HUB",
			Error: "This password reset link is invalid, expired, or has already been used.",
		})
		return
	}

	if r.Method == http.MethodGet {
		renderResetPasswordPage(w, PageData{
			Title: "Reset Password - IDOMA HUB",
			Token: token,
		})
		return
	}

	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if password != confirmPassword {
		renderResetPasswordPage(w, PageData{
			Title: "Reset Password - IDOMA HUB",
			Token: token,
			Error: "Your passwords do not match.",
		})
		return
	}

	if len(password) < 8 {
		renderResetPasswordPage(w, PageData{
			Title: "Reset Password - IDOMA HUB",
			Token: token,
			Error: "Your password must contain at least 8 characters.",
		})
		return
	}

	if len(password) > 72 {
		renderResetPasswordPage(w, PageData{
			Title: "Reset Password - IDOMA HUB",
			Token: token,
			Error: "Your password cannot exceed 72 bytes.",
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		log.Println("Password hashing failed:", err)
		http.Error(w, "Unable to reset password.", http.StatusInternalServerError)
		return
	}

	tokenHashBytes := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	tx, err := db.Begin()
	if err != nil {
		log.Println("Password reset transaction failed:", err)
		http.Error(w, "Unable to reset password.", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var userID int

	err = tx.QueryRow(`
		SELECT user_id
		FROM password_reset_tokens
		WHERE token_hash = $1
		AND used_at IS NULL
		AND expires_at > CURRENT_TIMESTAMP
		FOR UPDATE
	`, tokenHash).Scan(&userID)

	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Println("Password reset transaction token lookup failed:", err)
			http.Error(w, "Unable to reset password.", http.StatusInternalServerError)
			return
		}

		renderResetPasswordPage(w, PageData{
			Title: "Reset Password - IDOMA HUB",
			Error: "This password reset link is invalid, expired, or has already been used.",
		})
		return
	}

	_, err = tx.Exec(
		"UPDATE users SET password = $1 WHERE id = $2",
		string(hashedPassword),
		userID,
	)
	if err != nil {
		log.Println("Password update failed:", err)
		http.Error(w, "Unable to reset password.", http.StatusInternalServerError)
		return
	}

	result, err := tx.Exec(`
		UPDATE password_reset_tokens
		SET used_at = CURRENT_TIMESTAMP
		WHERE token_hash = $1 AND used_at IS NULL
	`, tokenHash)
	if err != nil {
		log.Println("Password reset token update failed:", err)
		http.Error(w, "Unable to reset password.", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected != 1 {
		log.Println("Password reset token could not be consumed.")
		http.Error(w, "Unable to reset password.", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Println("Password reset commit failed:", err)
		http.Error(w, "Unable to reset password.", http.StatusInternalServerError)
		return
	}

	deleteUserSessions(userID)

	http.Redirect(w, r, "/login?password_reset=1", http.StatusSeeOther)
}

func renderResetPasswordPage(w http.ResponseWriter, data PageData) {
	tmpl, err := template.ParseFiles("templates/reset_password.html")
	if err != nil {
		log.Println("Unable to load reset password page:", err)
		http.Error(w, "Unable to load page.", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Println("Unable to display reset password page:", err)
	}
}
