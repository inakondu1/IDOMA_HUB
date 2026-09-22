package main

import (
	"html/template"
	"log"
	"net/http"
)

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Unable to load your account.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	tmpl, err := template.ParseFiles("templates/dashboard.html")
	if err != nil {
		http.Error(w, "Unable to load dashboard.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	data := struct {
		Title    string
		Username string
	}{
		Title:    "IDOMA HUB - Dashboard",
		Username: username,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Unable to display dashboard.", http.StatusInternalServerError)
		log.Println(err)
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	deleteSession(r)

	http.SetCookie(w, &http.Cookie{
		Name:     "idoma_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
