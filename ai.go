package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type AIRequest struct {
	Contents []AIContent `json:"contents"`
}

type AIContent struct {
	Parts []AIPart `json:"parts"`
}

type AIPart struct {
	Text string `json:"text"`
}

type AIResponse struct {
	Candidates []AICandidate `json:"candidates"`
}

type AICandidate struct {
	Content AIContent `json:"content"`
}

func askGemini(question string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")

	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY is not set")
	}

	prompt := `You are IDOMA AI, the learning assistant inside IDOMA HUB.

Your main purpose is to help users learn the Idoma language, Idoma culture, Idoma history, Idoma names, phrases, traditions and heritage.

When a user asks how to say something in Idoma:
- Give the Idoma expression when you know it reliably.
- Give the English meaning.
- Give pronunciation guidance when useful.
- Do not invent Idoma words, translations or cultural facts.
- If you are unsure, clearly say that you are unsure instead of guessing.

Keep explanations simple, friendly and useful for someone learning Idoma.

If a question is unrelated to Idoma learning, politely guide the conversation back toward Idoma learning.

Learner's question:
` + question

	requestData := AIRequest{
		Contents: []AIContent{
			{
				Parts: []AIPart{
					{
						Text: prompt,
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", err
	}

	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.8-flash:generateContent"

	req, err := http.NewRequest(
		http.MethodPost,
		url,
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode == http.StatusServiceUnavailable {
		resp.Body.Close()
		time.Sleep(2 * time.Second)

		req, err = http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
		if err != nil {
			return "", err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-goog-api-key", apiKey)

		resp, err = client.Do(req)
		if err != nil {
			return "", err
		}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return "", fmt.Errorf("Gemini API returned status %s: %s", resp.Status, string(body))
	}

	var result AIResponse

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}

	if len(result.Candidates) == 0 ||
		len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("Gemini returned no answer")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}

func aiHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var username string

	err := db.QueryRow(
		"SELECT username FROM users WHERE id = $1",
		userID,
	).Scan(&username)

	if err != nil {
		http.Error(w, "Unable to load IDOMA AI", http.StatusInternalServerError)
		return
	}

	var question string
	var response string

	if r.Method == http.MethodPost {
		question = r.FormValue("question")

		if question != "" {
			response, err = askGemini(question)
			if err != nil {
				log.Printf("IDOMA AI Gemini error: %v", err)
				http.Error(w, fmt.Sprintf("IDOMA AI error: %v", err), http.StatusInternalServerError)
				return
			}
		}
	}

	data := struct {
		Username string
		Question string
		Response string
	}{
		Username: username,
		Question: question,
		Response: response,
	}

	tmpl, err := template.ParseFiles("templates/ai.html")
	if err != nil {
		http.Error(w, "Unable to load IDOMA AI page", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Unable to display IDOMA AI page", http.StatusInternalServerError)
		return
	}
}
