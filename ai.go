package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
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

	requestData := AIRequest{
		Contents: []AIContent{
			{
				Parts: []AIPart{
					{
						Text: question,
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", err
	}

	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.5-flash:generateContent"

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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Gemini API returned status %s", resp.Status)
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
				http.Error(w, "Unable to get a response from IDOMA AI", http.StatusInternalServerError)
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
