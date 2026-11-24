package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

const passwordResetExpiry = 30 * time.Minute

func generatePasswordResetToken() (string, string, time.Time, error) {
	tokenBytes := make([]byte, 32)

	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", time.Time{}, err
	}

	token := hex.EncodeToString(tokenBytes)

	tokenHashBytes := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	expiresAt := time.Now().Add(passwordResetExpiry)

	return token, tokenHash, expiresAt, nil
}

// createPasswordResetToken creates a new reset token and invalidates
// previous unused tokens for the same user.
func createPasswordResetToken(userID int) (string, error) {
	token, tokenHash, expiresAt, err := generatePasswordResetToken()
	if err != nil {
		return "", err
	}

	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// Lock the user row to prevent simultaneous reset requests
	// from creating multiple active tokens.
	var lockedUserID int
	err = tx.QueryRow(
		`SELECT id FROM users WHERE id = $1 FOR UPDATE`,
		userID,
	).Scan(&lockedUserID)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(`
		UPDATE password_reset_tokens
		SET used_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND used_at IS NULL
	`, userID)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(`
		INSERT INTO password_reset_tokens
		(user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return token, nil
}

// getPasswordResetUserID checks whether a reset token is valid.
func getPasswordResetUserID(token string) (int, error) {
	tokenHashBytes := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	var userID int

	err := db.QueryRow(`
                SELECT user_id
                FROM password_reset_tokens
                WHERE token_hash = $1
                AND used_at IS NULL
                AND expires_at > CURRENT_TIMESTAMP
        `, tokenHash).Scan(&userID)

	return userID, err
}
