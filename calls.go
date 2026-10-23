package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type callSignal struct {
	ID         int64           `json:"id"`
	SenderID   int             `json:"sender_id"`
	SignalType string          `json:"signal_type"`
	Payload    json.RawMessage `json:"payload"`
}

func callJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func callError(w http.ResponseWriter, status int, message string) {
	callJSON(w, status, map[string]string{"error": message})
}

func callForm(r *http.Request) error {
	return r.ParseForm()
}

func callUserID(w http.ResponseWriter, r *http.Request) (int, bool) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		callError(w, http.StatusUnauthorized, "Please log in to use calls.")
		return 0, false
	}
	return userID, true
}

// POST /calls/start
func callStartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		callError(w, http.StatusMethodNotAllowed, "POST required.")
		return
	}

	userID, ok := callUserID(w, r)
	if !ok {
		return
	}
	if err := callForm(r); err != nil {
		callError(w, http.StatusBadRequest, "Invalid form.")
		return
	}

	peerID, err := strconv.Atoi(r.FormValue("peer_id"))
	if err != nil || peerID <= 0 || peerID == userID {
		callError(w, http.StatusBadRequest, "Invalid call recipient.")
		return
	}

	callType := strings.ToLower(r.FormValue("call_type"))
	if callType != "audio" && callType != "video" {
		callError(w, http.StatusBadRequest, "Call type must be audio or video.")
		return
	}

	var exists bool
	err = db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`,
		peerID,
	).Scan(&exists)
	if err != nil {
		callError(w, http.StatusInternalServerError, "Unable to check recipient.")
		return
	}
	if !exists {
		callError(w, http.StatusNotFound, "Call recipient not found.")
		return
	}

	tx, err := db.Begin()
	if err != nil {
		callError(w, http.StatusInternalServerError, "Unable to start the call.")
		return
	}
	defer tx.Rollback()

	// Serialize call creation for this pair of users.
	var lockResult any
	err = tx.QueryRow(`
                SELECT pg_advisory_xact_lock(
                    LEAST($1::integer, $2::integer),
                    GREATEST($1::integer, $2::integer)
                )
        `, userID, peerID).Scan(&lockResult)
	if err != nil {
		log.Printf("Unable to reserve call: %v", err)
		callError(w, http.StatusInternalServerError, "Unable to reserve the call.")
		return
	}

	// IDOMA HUB stale-call cleanup: expire unanswered calls for this pair.
	_, err = tx.Exec(`
		UPDATE call_sessions
		SET status = 'missed', updated_at = CURRENT_TIMESTAMP
		WHERE status = 'ringing'
		  AND created_at < CURRENT_TIMESTAMP - INTERVAL '45 seconds'
		  AND ((caller_id = $1 AND callee_id = $2) OR (caller_id = $2 AND callee_id = $1))
	`, userID, peerID)
	if err != nil {
		log.Printf("Unable to expire stale ringing call: %v", err)
		callError(w, http.StatusInternalServerError, "Unable to clear expired calls.")
		return
	}

	// Release accepted calls whose event heartbeat has been absent for three minutes.
	_, err = tx.Exec(`
		UPDATE call_sessions
		SET status = 'ended', updated_at = CURRENT_TIMESTAMP
		WHERE status = 'accepted'
		  AND updated_at < CURRENT_TIMESTAMP - INTERVAL '3 minutes'
		  AND ((caller_id = $1 AND callee_id = $2) OR (caller_id = $2 AND callee_id = $1))
	`, userID, peerID)
	if err != nil {
		log.Printf("Unable to expire stale accepted call: %v", err)
		callError(w, http.StatusInternalServerError, "Unable to clear abandoned calls.")
		return
	}

	var activeCount int
	err = tx.QueryRow(`
		SELECT COUNT(*)
		FROM call_sessions
		WHERE status IN ('ringing', 'accepted')
		AND (
			(caller_id = $1 AND callee_id = $2)
			OR
			(caller_id = $2 AND callee_id = $1)
		)
	`, userID, peerID).Scan(&activeCount)
	if err != nil {
		callError(w, http.StatusInternalServerError, "Unable to check active calls.")
		return
	}
	if activeCount > 0 {
		callError(w, http.StatusConflict, "There is already an active call between these users.")
		return
	}

	var callID int64
	err = tx.QueryRow(`
		INSERT INTO call_sessions (caller_id, callee_id, call_type, status)
		VALUES ($1, $2, $3, 'ringing')
		RETURNING id
	`, userID, peerID, callType).Scan(&callID)
	if err != nil {
		callError(w, http.StatusInternalServerError, "Unable to start the call.")
		return
	}

	if err = tx.Commit(); err != nil {
		callError(w, http.StatusInternalServerError, "Unable to confirm the call.")
		return
	}

	callJSON(w, http.StatusCreated, map[string]any{
		"call_id":   callID,
		"status":    "ringing",
		"call_type": callType,
	})
}

// GET /calls/incoming?peer_id=123
func callIncomingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		callError(w, http.StatusMethodNotAllowed, "GET required.")
		return
	}

	userID, ok := callUserID(w, r)
	if !ok {
		return
	}

	peerID, err := strconv.Atoi(r.URL.Query().Get("peer_id"))
	if err != nil || peerID <= 0 || peerID == userID {
		callError(w, http.StatusBadRequest, "Invalid conversation partner.")
		return
	}

	// Expire unanswered calls and create missed-call notifications.
	if _, err := db.Exec(`
            WITH expired AS (
                    UPDATE call_sessions
                    SET status = 'missed', updated_at = CURRENT_TIMESTAMP
                    WHERE status = 'ringing'
                      AND created_at < CURRENT_TIMESTAMP - INTERVAL '45 seconds'
                    RETURNING caller_id, callee_id
            )
            INSERT INTO notifications (recipient_id, sender_id, post_id, type)
            SELECT callee_id, caller_id, NULL, 'missed_call'
            FROM expired
    `); err != nil {
		log.Printf("Unable to expire calls or create missed-call notifications: %v", err)
		callError(w, http.StatusInternalServerError, "Unable to check for expired calls.")
		return
	}

	var result struct {
		CallID   int64  `json:"call_id"`
		CallerID int    `json:"caller_id"`
		CalleeID int    `json:"callee_id"`
		CallType string `json:"call_type"`
		Status   string `json:"status"`
	}

	err = db.QueryRow(`
		SELECT id, caller_id, callee_id, call_type, status
		FROM call_sessions
		WHERE caller_id = $1
		  AND callee_id = $2
		  AND status = 'ringing'
		  AND created_at >= CURRENT_TIMESTAMP - INTERVAL '45 seconds'
		ORDER BY created_at DESC
		LIMIT 1
	`, peerID, userID).Scan(
		&result.CallID,
		&result.CallerID,
		&result.CalleeID,
		&result.CallType,
		&result.Status,
	)

	if err == sql.ErrNoRows {
		callJSON(w, http.StatusOK, map[string]any{"call": nil})
		return
	}
	if err != nil {
		callError(w, http.StatusInternalServerError, "Unable to check incoming calls.")
		return
	}

	callJSON(w, http.StatusOK, map[string]any{"call": result})
}

// GET /calls/incoming-any
// Checks for calls to the logged-in user without requiring a conversation partner.
func callIncomingAnyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		callError(w, http.StatusMethodNotAllowed, "GET required.")
		return
	}

	userID, ok := callUserID(w, r)
	if !ok {
		return
	}

	// Expire unanswered calls and create one missed-call notification per call.
	_, err := db.Exec(`
		WITH expired AS (
			UPDATE call_sessions
			SET status = 'missed', updated_at = CURRENT_TIMESTAMP
			WHERE status = 'ringing'
			  AND created_at < CURRENT_TIMESTAMP - INTERVAL '45 seconds'
			RETURNING caller_id, callee_id
		)
		INSERT INTO notifications (recipient_id, sender_id, post_id, type)
		SELECT callee_id, caller_id, NULL, 'missed_call'
		FROM expired
	`)
	if err != nil {
		log.Printf("Unable to expire calls or create missed-call notifications: %v", err)
		callError(w, http.StatusInternalServerError, "Unable to check incoming calls.")
		return
	}

	var result struct {
		CallID   int64  `json:"call_id"`
		CallerID int    `json:"caller_id"`
		CalleeID int    `json:"callee_id"`
		CallType string `json:"call_type"`
		Status   string `json:"status"`
		Username string `json:"username"`
	}

	err = db.QueryRow(`
		SELECT c.id, c.caller_id, c.callee_id, c.call_type, c.status, u.username
		FROM call_sessions c
		JOIN users u ON u.id = c.caller_id
		WHERE c.callee_id = $1
		  AND c.status = 'ringing'
		  AND c.created_at >= CURRENT_TIMESTAMP - INTERVAL '45 seconds'
		ORDER BY c.created_at DESC
		LIMIT 1
	`, userID).Scan(
		&result.CallID,
		&result.CallerID,
		&result.CalleeID,
		&result.CallType,
		&result.Status,
		&result.Username,
	)

	if err == sql.ErrNoRows {
		callJSON(w, http.StatusOK, map[string]any{"call": nil})
		return
	}
	if err != nil {
		log.Printf("Unable to find incoming call: %v", err)
		callError(w, http.StatusInternalServerError, "Unable to check incoming calls.")
		return
	}

	callJSON(w, http.StatusOK, map[string]any{"call": result})
}

// POST /calls/action
// Actions: accept, reject, end.
func callActionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		callError(w, http.StatusMethodNotAllowed, "POST required.")
		return
	}

	userID, ok := callUserID(w, r)
	if !ok {
		return
	}
	if err := callForm(r); err != nil {
		callError(w, http.StatusBadRequest, "Invalid form.")
		return
	}

	callID, err := strconv.ParseInt(r.FormValue("call_id"), 10, 64)
	if err != nil || callID <= 0 {
		callError(w, http.StatusBadRequest, "Invalid call ID.")
		return
	}

	action := strings.ToLower(r.FormValue("action"))

	var callerID, calleeID int
	var currentStatus string
	err = db.QueryRow(`
		SELECT caller_id, callee_id, status
		FROM call_sessions
		WHERE id = $1
	`, callID).Scan(&callerID, &calleeID, &currentStatus)
	if err == sql.ErrNoRows {
		callError(w, http.StatusNotFound, "Call not found.")
		return
	}
	if err != nil {
		callError(w, http.StatusInternalServerError, "Unable to load call.")
		return
	}
	if userID != callerID && userID != calleeID {
		callError(w, http.StatusForbidden, "You are not a participant in this call.")
		return
	}

	var result sql.Result

	switch action {
	case "accept":
		if userID != calleeID {
			callError(w, http.StatusForbidden, "Only the recipient can answer this call.")
			return
		}
		result, err = db.Exec(`
			UPDATE call_sessions
			SET status = 'accepted', updated_at = CURRENT_TIMESTAMP
			WHERE id = $1 AND status = 'ringing'
		`, callID)

	case "reject":
		if userID != calleeID {
			callError(w, http.StatusForbidden, "Only the recipient can reject this call.")
			return
		}
		result, err = db.Exec(`
			UPDATE call_sessions
			SET status = 'rejected', updated_at = CURRENT_TIMESTAMP
			WHERE id = $1 AND status = 'ringing'
		`, callID)

	case "end":
		result, err = db.Exec(`
			UPDATE call_sessions
			SET status = 'ended', updated_at = CURRENT_TIMESTAMP
			WHERE id = $1 AND status IN ('ringing', 'accepted')
		`, callID)

	case "heartbeat":
		if currentStatus != "accepted" {
			callError(w, http.StatusConflict, "Only an accepted call can send a heartbeat.")
			return
		}
		result, err = db.Exec(`
			UPDATE call_sessions
			SET updated_at = CURRENT_TIMESTAMP
			WHERE id = $1 AND status = 'accepted'
		`, callID)
	default:
		callError(w, http.StatusBadRequest, "Unknown call action.")
		return
	}

	if err != nil {
		callError(w, http.StatusInternalServerError, "Unable to update call.")
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		callError(w, http.StatusInternalServerError, "Unable to confirm call action.")
		return
	}
	if affected == 0 {
		callError(w, http.StatusConflict, "This call is no longer available.")
		return
	}

	callJSON(w, http.StatusOK, map[string]string{"status": action})
}

// POST /calls/signal
// Form fields: call_id, signal_type, payload.
func callSignalHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		callError(w, http.StatusMethodNotAllowed, "POST required.")
		return
	}

	userID, ok := callUserID(w, r)
	if !ok {
		return
	}
	if err := callForm(r); err != nil {
		callError(w, http.StatusBadRequest, "Invalid form.")
		return
	}

	callID, err := strconv.ParseInt(r.FormValue("call_id"), 10, 64)
	if err != nil || callID <= 0 {
		callError(w, http.StatusBadRequest, "Invalid call ID.")
		return
	}

	signalType := strings.ToLower(r.FormValue("signal_type"))
	if signalType != "offer" && signalType != "answer" && signalType != "ice" {
		callError(w, http.StatusBadRequest, "Invalid signal type.")
		return
	}

	payload := r.FormValue("payload")
	if len(payload) == 0 || len(payload) > 65536 || !json.Valid([]byte(payload)) {
		callError(w, http.StatusBadRequest, "Invalid signaling payload.")
		return
	}

	var callerID, calleeID int
	var status string
	err = db.QueryRow(`
		SELECT caller_id, callee_id, status
		FROM call_sessions
		WHERE id = $1
	`, callID).Scan(&callerID, &calleeID, &status)
	if err == sql.ErrNoRows {
		callError(w, http.StatusNotFound, "Call not found.")
		return
	}
	if err != nil {
		callError(w, http.StatusInternalServerError, "Unable to load call.")
		return
	}
	if userID != callerID && userID != calleeID {
		callError(w, http.StatusForbidden, "You are not a participant in this call.")
		return
	}
	if status != "accepted" {
		callError(w, http.StatusConflict, "The call must be answered before signaling.")
		return
	}

	recipientID := callerID
	if userID == callerID {
		recipientID = calleeID
	}

	_, err = db.Exec(`
		INSERT INTO call_signals
			(call_id, sender_id, recipient_id, signal_type, payload)
		VALUES ($1, $2, $3, $4, $5::jsonb)
	`, callID, userID, recipientID, signalType, payload)
	if err != nil {
		callError(w, http.StatusInternalServerError, "Unable to send call signal.")
		return
	}

	callJSON(w, http.StatusCreated, map[string]string{"status": "sent"})
}

// GET /calls/events?call_id=123&after=0
func callEventsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		callError(w, http.StatusMethodNotAllowed, "GET required.")
		return
	}

	userID, ok := callUserID(w, r)
	if !ok {
		return
	}

	callID, err := strconv.ParseInt(r.URL.Query().Get("call_id"), 10, 64)
	if err != nil || callID <= 0 {
		callError(w, http.StatusBadRequest, "Invalid call ID.")
		return
	}

	after, err := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	if err != nil || after < 0 {
		after = 0
	}

	var callerID, calleeID int
	var callType, status string
	err = db.QueryRow(`
		SELECT caller_id, callee_id, call_type, status
		FROM call_sessions
		WHERE id = $1
	`, callID).Scan(&callerID, &calleeID, &callType, &status)
	if err == sql.ErrNoRows {
		callError(w, http.StatusNotFound, "Call not found.")
		return
	}
	if err != nil {
		callError(w, http.StatusInternalServerError, "Unable to load call.")
		return
	}
	if userID != callerID && userID != calleeID {
		callError(w, http.StatusForbidden, "You are not a participant in this call.")
		return
	}

	rows, err := db.Query(`
		SELECT id, sender_id, signal_type, payload::text
		FROM call_signals
		WHERE call_id = $1 AND recipient_id = $2 AND id > $3
		ORDER BY id ASC
		LIMIT 100
	`, callID, userID, after)
	if err != nil {
		callError(w, http.StatusInternalServerError, "Unable to load call signals.")
		return
	}
	defer rows.Close()

	signals := make([]callSignal, 0)
	for rows.Next() {
		var signal callSignal
		var payload []byte

		if err := rows.Scan(
			&signal.ID,
			&signal.SenderID,
			&signal.SignalType,
			&payload,
		); err != nil {
			callError(w, http.StatusInternalServerError, "Unable to read call signals.")
			return
		}
		signal.Payload = json.RawMessage(payload)
		signals = append(signals, signal)
	}
	if err := rows.Err(); err != nil {
		callError(w, http.StatusInternalServerError, "Unable to read call signals.")
		return
	}

	callJSON(w, http.StatusOK, map[string]any{
		"call_id":   callID,
		"caller_id": callerID,
		"callee_id": calleeID,
		"call_type": callType,
		"status":    status,
		"signals":   signals,
	})
}
