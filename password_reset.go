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

// createPasswordResetToken creates a new reset token without
// invalidating previously issued tokens.
func createPasswordResetToken(userID int) (string, error) {
	token, tokenHash, expiresAt, err := generatePasswordResetToken()
	if err != nil {
		return "", err
	}

	_, err = db.Exec(`
		INSERT INTO password_reset_tokens
		(user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	if err != nil {
		return "", err
	}

	return token, nil
}

// invalidateOtherPasswordResetTokens invalidates all other
// unused tokens belonging to the user.
func invalidateOtherPasswordResetTokens(userID int, currentToken string) error {
	tokenHashBytes := sha256.Sum256([]byte(currentToken))
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	_, err := db.Exec(`
		UPDATE password_reset_tokens
		SET used_at = CURRENT_TIMESTAMP
		WHERE user_id = $1
		AND token_hash != $2
		AND used_at IS NULL
	`, userID, tokenHash)

	return err
}

// deletePasswordResetToken removes a newly created token if
// email delivery fails.
func deletePasswordResetToken(token string) error {
	tokenHashBytes := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	_, err := db.Exec(`
		DELETE FROM password_reset_tokens
		WHERE token_hash = $1
		AND used_at IS NULL
	`, tokenHash)

	return err
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
