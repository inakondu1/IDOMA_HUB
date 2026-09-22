package main

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
)

var (
	sessions   = make(map[string]int)
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
	sessions[sessionID] = userID
	sessionMux.Unlock()

	return sessionID, nil
}

func getUserIDFromSession(r *http.Request) (int, bool) {
	cookie, err := r.Cookie("idoma_session")
	if err != nil {
		return 0, false
	}

	sessionMux.RLock()
	userID, exists := sessions[cookie.Value]
	sessionMux.RUnlock()

	return userID, exists
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
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
