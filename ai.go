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

func findIdomaTranslation(englishText string) (string, string, error) {
	var english string
	var idoma string

	err := db.QueryRow(`
		SELECT english, idoma
		FROM idoma_language_data
		WHERE LOWER(english) = LOWER($1)
		LIMIT 1
	`, englishText).Scan(&english, &idoma)

	if err == sql.ErrNoRows {
		return "", "", nil
	}

	if err != nil {
		return "", "", err
	}

	return english, idoma, nil
}

type TranslationMatch struct {
	Idoma   string
	English string
}

func searchIdomaTranslations(englishText string) ([]TranslationMatch, error) {
	rows, err := db.Query(`
		SELECT idoma, english
		FROM idoma_language_data
		WHERE to_tsvector('simple', english) @@ plainto_tsquery('simple', $1)
                OR LOWER(english) LIKE '%' || LOWER($1) || '%'
		ORDER BY id
		LIMIT 10
	`, englishText)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []TranslationMatch

	for rows.Next() {
		var match TranslationMatch

		if err := rows.Scan(
			&match.Idoma,
			&match.English,
		); err != nil {
			return nil, err
		}

		matches = append(matches, match)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return matches, nil
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
				prompt := question

				if referenceContext := idomaReferenceContext(question); referenceContext != "" {
					prompt = "You are IDOMA AI, an assistant for learning and preserving the Idoma language.\n" +
						"Answer the user's question using the supplied reference material when relevant.\n" +
						"Preserve Idoma spelling and tone marks exactly as supplied.\n" +
						"Do not invent meanings for missing entries. Clearly say when the reference does not provide an answer.\n" +
						"Treat dialect forms as belonging to their stated communities; do not claim they are all interchangeable.\n\n" +
						"USER QUESTION:\n" + question + "\n\n" +
						"REFERENCE MATERIAL:\n" + referenceContext
				}

				response, err = askGemini(prompt)
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

func aiTranslateHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Unable to load IDOMA Translator", http.StatusInternalServerError)
		return
	}

	var text string
	var matches []TranslationMatch
	var translationError string

	if r.Method == http.MethodPost {
		text = r.FormValue("text")

		if text != "" {
			matches, err = searchIdomaTranslations(text)
			if err != nil {
				log.Printf("IDOMA translator database error: %v", err)
				translationError = "Unable to search the IDOMA translation database."
			}
		}
	}

	data := struct {
		Username string
		Text     string
		Matches  []TranslationMatch
		Error    string
	}{
		Username: username,
		Text:     text,
		Matches:  matches,
		Error:    translationError,
	}

	tmpl, err := template.ParseFiles("templates/translate.html")
	if err != nil {
		http.Error(w, "Unable to load IDOMA Translator page", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Unable to display IDOMA Translator page", http.StatusInternalServerError)
		return
	}
}
