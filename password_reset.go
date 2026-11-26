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
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	return token, tokenHash, time.Now().Add(passwordResetExpiry), nil
}

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

func getPasswordResetUserID(token string) (int, error) {
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

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

// resetPasswordWithToken changes a password and consumes its token atomically.
func resetPasswordWithToken(token, hashedPassword string) (int, error) {
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var userID int

	err = tx.QueryRow(`
		SELECT user_id
		FROM password_reset_tokens
		WHERE token_hash = $1
		AND used_at IS NULL
		AND expires_at > CURRENT_TIMESTAMP
	`, tokenHash).Scan(&userID)

	if err != nil {
		return 0, err
	}

	// Lock the user first, matching the order used when creating tokens.
	var lockedUserID int

	err = tx.QueryRow(`
		SELECT id FROM users WHERE id = $1 FOR UPDATE
	`, userID).Scan(&lockedUserID)

	if err != nil {
		return 0, err
	}

	// Recheck and lock the token.
	var tokenID int

	err = tx.QueryRow(`
		SELECT id
		FROM password_reset_tokens
		WHERE token_hash = $1
		AND user_id = $2
		AND used_at IS NULL
		AND expires_at > CURRENT_TIMESTAMP
		FOR UPDATE
	`, tokenHash, userID).Scan(&tokenID)

	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`
		UPDATE users
		SET password = $1
		WHERE id = $2
	`, hashedPassword, userID)

	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`
		UPDATE password_reset_tokens
		SET used_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND used_at IS NULL
	`, userID)

	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return userID, nil
}
