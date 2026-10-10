package main

import (
	"html/template"
	"net/http"
)

type IdomaReferencePage struct {
	Title       string
	Description string
	Username    string
	Path        string
	Homographs  []IdomaHomograph
	Dialects    []IdomaDialect
	SpeechWork  []IdomaSpeechSentence
}

func idomaReferenceHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Unable to load Idoma learning material", http.StatusInternalServerError)
		return
	}

	pages := map[string]IdomaReferencePage{
		"/homographs": {
			Title:       "Idoma Homographs",
			Description: "Study Idoma words with different meanings and tone distinctions.",
			Path:        "/homographs",
			Homographs:  idomaHomographs,
		},
		"/dialects": {
			Title:       "Idoma Dialectal Variations",
			Description: "Compare vocabulary used in different Idoma-speaking communities.",
			Path:        "/dialects",
			Dialects:    idomaDialects,
		},
		"/speech-work": {
			Title:       "Idoma Speech Work",
			Description: "Learn from Idoma sentences and their English translations.",
			Path:        "/speech-work",
			SpeechWork:  idomaSpeechWork,
		},
	}

	pages["/build-sentences"] = IdomaReferencePage{
		Title:       "Build Idoma Sentences",
		Description: "Learn how to form Idoma sentences through guided lessons and practice.",
		Path:        "/build-sentences",
		SpeechWork:  idomaSpeechWork,
	}

	page, ok := pages[r.URL.Path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	page.Username = username

	tmpl, err := template.ParseFiles("templates/idoma_reference.html")
	if err != nil {
		http.Error(w, "Unable to load Idoma lesson", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, page); err != nil {
		http.Error(w, "Unable to display Idoma lesson", http.StatusInternalServerError)
		return
	}
}
