package main

import (
	"html/template"
	"net/http"
)

type IdomaWord struct {
	Word     string
	Meaning  string
	Example  string
	Category string
	Dialect  string
	Source   string
}

type WordsPageData struct {
	Username string
	Words    []IdomaWord
}

func wordsHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Unable to load words page", http.StatusInternalServerError)
		return
	}

	words := []IdomaWord{
		{
			Word:     "Lohi",
			Meaning:  "Good",
			Category: "Common",
			Dialect:  "Central Idoma",
			Source:   "IdomaLand",
		},
		{
			Word:     "A'da",
			Meaning:  "Father",
			Category: "Family",
			Dialect:  "Idoma",
			Source:   "IdomaLand",
		},
		{
			Word:     "Enem",
			Meaning:  "My mother",
			Category: "Family",
			Dialect:  "Idoma",
			Source:   "IdomaLand",
		},
		{
			Word:     "Ainya / Anya",
			Meaning:  "Thanks",
			Category: "Common",
			Dialect:  "Idoma",
			Source:   "IdomaLand",
		},
		{
			Word:     "Ennkpo",
			Meaning:  "Water",
			Category: "Common",
			Dialect:  "Central Idoma",
			Source:   "IdomaLand",
		},
		{
			Word:     "Enyi",
			Meaning:  "Water",
			Category: "Common",
			Dialect:  "Western Idoma",
			Source:   "IdomaLand",
		},
		{
			Word:     "Emie",
			Meaning:  "Hunger",
			Category: "Common",
			Dialect:  "Idoma",
			Source:   "IdomaLand",
		},
		{
			Word:     "Ebe",
			Meaning:  "Meat",
			Category: "Food",
			Dialect:  "Idoma",
			Source:   "IdomaLand",
		},
		{
			Word:     "Owo",
			Meaning:  "Rain",
			Category: "Nature",
			Dialect:  "Idoma",
			Source:   "IdomaLand",
		},
		{
			Word:     "Waa",
			Meaning:  "Come",
			Category: "Common",
			Dialect:  "Idoma",
			Source:   "IdomaLand",
		},
		{
			Word:     "Odi",
			Meaning:  "What",
			Category: "Conversation",
			Dialect:  "Central Idoma",
			Source:   "IdomaLand",
		},
	}

	data := WordsPageData{
		Username: username,
		Words:    words,
	}

	tmpl, err := template.ParseFiles("templates/words.html")
	if err != nil {
		http.Error(w, "Unable to load words page", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Unable to display words page", http.StatusInternalServerError)
		return
	}
}
