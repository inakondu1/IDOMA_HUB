package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const translationDBPath = "idoma_translation_db/idoma_translation_search.db"

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

func findIdomaTranslation(englishText string) (string, string, error) {
	db, err := sql.Open("sqlite3", translationDBPath)
	if err != nil {
		return "", "", err
	}
	defer db.Close()

	var verseKey string
	var idu string

	err = db.QueryRow(`
		SELECT verse_key, idu
		FROM translations
		WHERE en = ?
		LIMIT 1
	`, englishText).Scan(&verseKey, &idu)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", nil
		}
		return "", "", err
	}

	return verseKey, idu, nil
}

func askGemini(prompt string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
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

	models := []string{
		"gemini-3.5-flash",
		"gemini-3.7-flash",
		"gemini-3.6-flash",
	}

	client := &http.Client{}
	var lastErr error

	for _, model := range models {
		url := fmt.Sprintf(
			"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent",
			model,
		)

		for attempt := 1; attempt <= 2; attempt++ {
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

			resp, err := client.Do(req)
			if err != nil {
				return "", err
			}

			if resp.StatusCode == http.StatusOK {
				var result AIResponse

				err = json.NewDecoder(resp.Body).Decode(&result)
				resp.Body.Close()

				if err != nil {
					return "", err
				}

				if len(result.Candidates) == 0 ||
					len(result.Candidates[0].Content.Parts) == 0 {
					return "", fmt.Errorf("Gemini returned no answer")
				}

				return result.Candidates[0].Content.Parts[0].Text, nil
			}

			body, _ := io.ReadAll(resp.Body)
			status := resp.Status
			resp.Body.Close()

			lastErr = fmt.Errorf(
				"Gemini model %s returned status %s: %s",
				model,
				status,
				string(body),
			)

			transient := resp.StatusCode == http.StatusRequestTimeout ||
				resp.StatusCode == http.StatusTooManyRequests ||
				resp.StatusCode >= 500

			if !transient {
				break
			}

			if attempt < 2 {
				time.Sleep(time.Duration(1<<(attempt-1)) * time.Second)
			}
		}
	}

	if lastErr != nil {
		return "", lastErr
	}

	return "", fmt.Errorf("Gemini request failed")
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
	var aiError string

	if r.Method == http.MethodPost {
		question = r.FormValue("question")

		if question != "" {
			verseKey, idomaText, lookupErr := findIdomaTranslation(question)

			if lookupErr != nil {
				log.Printf("IDOMA translation database error: %v", lookupErr)
			}

			if idomaText != "" {
				response = fmt.Sprintf(
					"Idoma translation (%s):\n%s",
					verseKey,
					idomaText,
				)
			} else {
				response, err = askGemini(question)
				if err != nil {
					log.Printf("IDOMA AI Gemini error: %v", err)
					aiError = "IDOMA AI is temporarily unavailable. Please try again shortly."
				}
			}
		}
	}

	data := struct {
		Username string
		Question string
		Response string
		Error    string
	}{
		Username: username,
		Question: question,
		Response: response,
		Error:    aiError,
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
