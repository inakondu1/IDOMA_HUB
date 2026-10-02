package main

import (
	"html/template"
	"net/http"
)

func bibleHandler(w http.ResponseWriter, r *http.Request) {
	_, loggedIn := getUserIDFromSession(r)

	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	tmpl, err := template.ParseFiles("templates/bible.html")
	if err != nil {
		http.Error(w, "Unable to load IDOMA Bible page", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "Unable to display IDOMA Bible page", http.StatusInternalServerError)
		return
	}
}
