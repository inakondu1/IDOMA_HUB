package main

import (
	"html/template"
	"net/http"
)

type PracticeQuestion struct {
	Question string
	Options  []string
	Answer   string
}

type PracticePageData struct {
	Username  string
	Questions []PracticeQuestion
}

func practiceHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Unable to load practice page", http.StatusInternalServerError)
		return
	}

	questions := []PracticeQuestion{
		{
			Question: "What does Lohi mean?",
			Options:  []string{"Good", "Water", "Father", "Rain"},
			Answer:   "Good",
		},
		{
			Question: "What does A'da mean?",
			Options:  []string{"Mother", "Father", "Meat", "Rain"},
			Answer:   "Father",
		},
		{
			Question: "What does Ennkpo mean in Central Idoma?",
			Options:  []string{"Water", "Good", "Father", "Thanks"},
			Answer:   "Water",
		},
	}

	data := PracticePageData{
		Username:  username,
		Questions: questions,
	}

	tmpl, err := template.ParseFiles("templates/practice.html")
	if err != nil {
		http.Error(w, "Unable to load practice page", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Unable to display practice page", http.StatusInternalServerError)
		return
	}
}
