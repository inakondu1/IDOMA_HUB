package main

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
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

func deleteUserSessions(userID int) {
	sessionMux.Lock()
	defer sessionMux.Unlock()

	for sessionID, id := range sessions {
		if id == userID {
			delete(sessions, sessionID)
		}
	}
}

func setSessionCookie(w http.ResponseWriter, sessionID string) {
	secure := strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true")

	http.SetCookie(w, &http.Cookie{
		Name:     "idoma_session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
