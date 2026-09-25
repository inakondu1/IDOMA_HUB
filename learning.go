package main

import (
	"html/template"
	"net/http"
)

type LearningPageData struct {
	Username string
}

func learningHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Unable to load learning page", http.StatusInternalServerError)
		return
	}

	data := LearningPageData{
		Username: username,
	}

	tmpl, err := template.ParseFiles("templates/learning.html")
	if err != nil {
		http.Error(w, "Unable to load learning page", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Unable to display learning page", http.StatusInternalServerError)
		return
	}
}
