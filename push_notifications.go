package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
)

type pushSubscriptionRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

type pushNotificationPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
}

func pushPublicKeyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := strings.TrimSpace(os.Getenv("IDOMA_PUSH_VAPID_PUBLIC_KEY"))
	if key == "" {
		http.Error(w, "Push notifications are not configured yet", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"publicKey": key})
}

func pushSubscribeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getUserIDFromSession(r)
	if !ok {
		http.Error(w, "Please sign in first", http.StatusUnauthorized)
		return
	}

	var input pushSubscriptionRequest
	r.Body = http.MaxBytesReader(w, r.Body, 16384)
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid subscription data", http.StatusBadRequest)
		return
	}

	endpoint := strings.TrimSpace(input.Endpoint)
	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil || parsedEndpoint.Scheme != "https" || parsedEndpoint.Host == "" {
		http.Error(w, "Invalid push endpoint", http.StatusBadRequest)
		return
	}

	p256dh := strings.TrimSpace(input.Keys.P256dh)
	auth := strings.TrimSpace(input.Keys.Auth)
	if p256dh == "" || auth == "" {
		http.Error(w, "Missing subscription encryption keys", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(`
		INSERT INTO push_subscriptions
			(user_id, endpoint, p256dh, auth, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		ON CONFLICT (endpoint) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			p256dh = EXCLUDED.p256dh,
			auth = EXCLUDED.auth,
			updated_at = CURRENT_TIMESTAMP
	`, userID, endpoint, p256dh, auth)
	if err != nil {
		log.Println("Unable to save push subscription:", err)
		http.Error(w, "Unable to save notification settings", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func pushUnsubscribeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := getUserIDFromSession(r)
	if !ok {
		http.Error(w, "Please sign in first", http.StatusUnauthorized)
		return
	}

	var input struct {
		Endpoint string `json:"endpoint"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || strings.TrimSpace(input.Endpoint) == "" {
		http.Error(w, "Invalid subscription data", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(
		`DELETE FROM push_subscriptions WHERE user_id = $1 AND endpoint = $2`,
		userID,
		strings.TrimSpace(input.Endpoint),
	)
	if err != nil {
		log.Println("Unable to remove push subscription:", err)
		http.Error(w, "Unable to update notification settings", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func sendPushToUser(userID int, title, body, targetURL string) {
	rows, err := db.Query(`
		SELECT endpoint, p256dh, auth
		FROM push_subscriptions
		WHERE user_id = $1
	`, userID)
	if err != nil {
		log.Println("Unable to load push subscriptions:", err)
		return
	}
	defer rows.Close()

	var subscriptions []webpush.Subscription
	for rows.Next() {
		var endpoint, p256dh, auth string
		if err := rows.Scan(&endpoint, &p256dh, &auth); err != nil {
			log.Println("Unable to read push subscription:", err)
			return
		}
		subscriptions = append(subscriptions, webpush.Subscription{
			Endpoint: endpoint,
			Keys: webpush.Keys{
				P256dh: p256dh,
				Auth:   auth,
			},
		})
	}
	if err := rows.Err(); err != nil {
		log.Println("Error reading push subscriptions:", err)
		return
	}

	sendPushSubscriptions(subscriptions, title, body, targetURL)
}

func sendPushToAll(title, body, targetURL string) {
	rows, err := db.Query(`
		SELECT endpoint, p256dh, auth
		FROM push_subscriptions
	`)
	if err != nil {
		log.Println("Unable to load broadcast push subscriptions:", err)
		return
	}
	defer rows.Close()

	var subscriptions []webpush.Subscription
	for rows.Next() {
		var endpoint, p256dh, auth string
		if err := rows.Scan(&endpoint, &p256dh, &auth); err != nil {
			log.Println("Unable to read broadcast push subscription:", err)
			return
		}
		subscriptions = append(subscriptions, webpush.Subscription{
			Endpoint: endpoint,
			Keys: webpush.Keys{
				P256dh: p256dh,
				Auth:   auth,
			},
		})
	}
	if err := rows.Err(); err != nil {
		log.Println("Error reading broadcast push subscriptions:", err)
		return
	}

	sendPushSubscriptions(subscriptions, title, body, targetURL)
}

func sendPushSubscriptions(subscriptions []webpush.Subscription, title, body, targetURL string) {
	publicKey := strings.TrimSpace(os.Getenv("IDOMA_PUSH_VAPID_PUBLIC_KEY"))
	privateKey := strings.TrimSpace(os.Getenv("IDOMA_PUSH_VAPID_PRIVATE_KEY"))
	subject := strings.TrimSpace(os.Getenv("IDOMA_PUSH_SUBJECT"))
	if publicKey == "" || privateKey == "" || subject == "" {
		return
	}

	payload, err := json.Marshal(pushNotificationPayload{
		Title: title,
		Body:  body,
		URL:   targetURL,
	})
	if err != nil {
		log.Println("Unable to encode push notification:", err)
		return
	}

	options := &webpush.Options{
		Subscriber:      subject,
		VAPIDPublicKey:  publicKey,
		VAPIDPrivateKey: privateKey,
		TTL:             int((24 * time.Hour).Seconds()),
	}

	for _, subscription := range subscriptions {
		response, sendErr := webpush.SendNotificationWithContext(
			context.Background(),
			payload,
			&subscription,
			options,
		)
		if sendErr != nil {
			log.Println("Unable to send push notification:", sendErr)
			continue
		}

		if response == nil {
			continue
		}
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()

		if response.StatusCode == http.StatusGone || response.StatusCode == http.StatusNotFound {
			_, deleteErr := db.Exec(
				`DELETE FROM push_subscriptions WHERE endpoint = $1`,
				subscription.Endpoint,
			)
			if deleteErr != nil {
				log.Println("Unable to delete expired push subscription:", deleteErr)
			}
		} else if response.StatusCode < 200 || response.StatusCode >= 300 {
			log.Printf("Push service returned HTTP %d\n", response.StatusCode)
		}
	}
}
