package main

import (
	"html/template"
	"net/http"
)

type FarmProduce struct {
	ID          int
	Name        string
	Description string
	ImageURL    string
}

type FarmProducePageData struct {
	Username string
	Produce  []FarmProduce
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

	rows, err := db.Query(`
		SELECT id, name, description, image_url
		FROM farm_produce
		ORDER BY id ASC
	`)
	if err != nil {
		http.Error(w, "Unable to load farm produce", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var produce []FarmProduce

	for rows.Next() {
		var item FarmProduce

		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.ImageURL,
		)
		if err != nil {
			http.Error(w, "Unable to read farm produce", http.StatusInternalServerError)
			return
		}

		produce = append(produce, item)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Unable to read farm produce", http.StatusInternalServerError)
		return
	}

	data := FarmProducePageData{
		Username: username,
		Produce:  produce,
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
