package main

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const sessionLifetime = 24 * time.Hour

type sessionData struct {
	userID    int
	expiresAt time.Time
}

var (
	sessions   = make(map[string]sessionData)
	sessionMux sync.RWMutex
)

func createSession(userID int) (string, error) {
	bytes := make([]byte, 32)

	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	sessionID := hex.EncodeToString(bytes)

	sessionMux.Lock()
	defer sessionMux.Unlock()

	now := time.Now()

	for id, session := range sessions {
		if now.After(session.expiresAt) {
			delete(sessions, id)
		}
	}

	sessions[sessionID] = sessionData{
		userID:    userID,
		expiresAt: now.Add(sessionLifetime),
	}

	return sessionID, nil
}

func getUserIDFromSession(r *http.Request) (int, bool) {
	cookie, err := r.Cookie("idoma_session")
	if err != nil {
		return 0, false
	}

	sessionMux.Lock()
	defer sessionMux.Unlock()

	session, exists := sessions[cookie.Value]
	if !exists {
		return 0, false
	}

	if time.Now().After(session.expiresAt) {
		delete(sessions, cookie.Value)
		return 0, false
	}

	return session.userID, true
}

func deleteSession(r *http.Request) {
	cookie, err := r.Cookie("idoma_session")
	if err != nil {
		return
	}

	sessionMux.Lock()
	delete(sessions, cookie.Value)
	sessionMux.Unlock()
}

func setSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "idoma_session",
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(sessionLifetime.Seconds()),
		Expires:  time.Now().Add(sessionLifetime),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// deleteUserSessions removes all active sessions belonging to a user.
func deleteUserSessions(userID int) {
	sessionMux.Lock()
	defer sessionMux.Unlock()

	for sessionID, session := range sessions {
		if session.userID == userID {
			delete(sessions, sessionID)
		}
	}
}
