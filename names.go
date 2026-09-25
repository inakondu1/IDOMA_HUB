package main

import (
	"html/template"
	"net/http"
)

type IdomaName struct {
	Name     string
	Gender   string
	Meaning  string
	Category string
	Source   string
}

type NamesPageData struct {
	Username string
	Names    []IdomaName
}

func namesHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Unable to load names page", http.StatusInternalServerError)
		return
	}

	names := []IdomaName{
		{
			Name:     "Adakole",
			Gender:   "Male / Female",
			Meaning:  "Father of the family",
			Category: "Family",
			Source:   "Idoma Voice",
		},
		{
			Name:     "Adah",
			Gender:   "Male",
			Meaning:  "Father of nations",
			Category: "Family",
			Source:   "Idoma Voice",
		},
		{
			Name:     "Agaba",
			Gender:   "Male",
			Meaning:  "Lion",
			Category: "Strength",
			Source:   "Idoma Voice",
		},
		{
			Name:     "Anyebe",
			Gender:   "Male",
			Meaning:  "Victory",
			Category: "Achievement",
			Source:   "Idoma Voice",
		},
		{
			Name:     "Ebo",
			Gender:   "Female",
			Meaning:  "Peace",
			Category: "Values",
			Source:   "Idoma Voice",
		},
		{
			Name:     "Ehi",
			Gender:   "Female",
			Meaning:  "Gift",
			Category: "Values",
			Source:   "Idoma Voice",
		},
		{
			Name:     "Ene",
			Gender:   "Female",
			Meaning:  "Mother",
			Category: "Family",
			Source:   "Idoma Voice",
		},
		{
			Name:     "Ehik'owoicho",
			Gender:   "Female",
			Meaning:  "God's Gift",
			Category: "Faith",
			Source:   "Idoma Voice",
		},
		{
			Name:     "Ekowo",
			Gender:   "Female",
			Meaning:  "God's time",
			Category: "Faith",
			Source:   "Idoma Voice",
		},
		{
			Name:     "Ihotu",
			Gender:   "Unisex",
			Meaning:  "Love",
			Category: "Values",
			Source:   "Idoma Voice",
		},
		{
			Name:     "Ochanya",
			Gender:   "Female",
			Meaning:  "Queen",
			Category: "Leadership",
			Source:   "Idoma Voice",
		},
		{
			Name:     "Ngbede",
			Gender:   "Male",
			Meaning:  "Happiness",
			Category: "Values",
			Source:   "Soluap",
		},
	}

	data := NamesPageData{
		Username: username,
		Names:    names,
	}

	tmpl, err := template.ParseFiles("templates/names.html")
	if err != nil {
		http.Error(w, "Unable to load names page", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Unable to display names page", http.StatusInternalServerError)
		return
	}
}
