package main

import (
	"html/template"
	"net/http"
)

type CulturePageData struct {
	Username string
}

func cultureHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Unable to load culture page", http.StatusInternalServerError)
		return
	}

	data := CulturePageData{
		Username: username,
	}

	tmpl, err := template.ParseFiles("templates/culture.html")
	if err != nil {
		http.Error(w, "Unable to load culture page", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Unable to display culture page", http.StatusInternalServerError)
		return
	}
}
