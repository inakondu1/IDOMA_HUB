package main

import (
	"html/template"
	"net/http"
)

type IdomaPhrase struct {
	Phrase   string
	Meaning  string
	Category string
	Dialect  string
	Source   string
}

type PhrasesPageData struct {
	Username string
	Phrases  []IdomaPhrase
}

func phrasesHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var username string

	err := db.QueryRow(
		"SELECT username FROM users WHERE id = ?",
		userID,
	).Scan(&username)

	if err != nil {
		http.Error(w, "Unable to load phrases page", http.StatusInternalServerError)
		return
	}

	phrases := []IdomaPhrase{
		{
			Phrase:   "Nmaochi",
			Meaning:  "Good morning",
			Category: "Greetings",
			Dialect:  "Idoma",
			Source:   "IdomaLand",
		},
		{
			Phrase:   "Abo le? / Abole?",
			Meaning:  "How are you?",
			Category: "Greetings",
			Dialect:  "Idoma",
			Source:   "IdomaLand",
		},
		{
			Phrase:   "A gbe hii",
			Meaning:  "Are you OK? / Are you doing OK?",
			Category: "Greetings",
			Dialect:  "Idoma",
			Source:   "IdomaLand",
		},
		{
			Phrase:   "Nmoo",
			Meaning:  "Goodbye",
			Category: "Greetings",
			Dialect:  "Idoma",
			Source:   "IdomaLand",
		},
		{
			Phrase:   "Ahinya / Anya",
			Meaning:  "Thank you / Thanks",
			Category: "Common",
			Dialect:  "Idoma",
			Source:   "IdomaLand",
		},
	}

	data := PhrasesPageData{
		Username: username,
		Phrases:  phrases,
	}

	tmpl, err := template.ParseFiles("templates/phrases.html")
	if err != nil {
		http.Error(w, "Unable to load phrases page", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Unable to display phrases page", http.StatusInternalServerError)
		return
	}
}
