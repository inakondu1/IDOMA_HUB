package main

import (
	"html/template"
	"net/http"
)

type FarmProducePageData struct {
	Username string
}

func farmProduceHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Unable to load farm produce page", http.StatusInternalServerError)
		return
	}

	data := FarmProducePageData{
		Username: username,
	}

	tmpl, err := template.ParseFiles("templates/farm-produce.html")
	if err != nil {
		http.Error(w, "Unable to load farm produce page", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Unable to display farm produce page", http.StatusInternalServerError)
		return
	}
}
