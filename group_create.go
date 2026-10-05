package main

import (
	"html/template"
	"net/http"
	"strings"
)

func createGroupHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		name := strings.TrimSpace(r.FormValue("name"))
		description := strings.TrimSpace(r.FormValue("description"))

		if name == "" || description == "" {
			http.Error(w, "Group name and description are required", http.StatusBadRequest)
			return
		}

		var groupID int

		err := db.QueryRow(`
			INSERT INTO groups (name, description, creator_id)
			VALUES ($1, $2, $3)
			RETURNING id
		`, name, description, userID).Scan(&groupID)

		if err != nil {
			http.Error(w, "Unable to create group. The group name may already exist.", http.StatusBadRequest)
			return
		}

		_, err = db.Exec(`
			INSERT INTO group_members (group_id, user_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, groupID, userID)

		if err != nil {
			http.Error(w, "Group was created but membership could not be added.", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/groups", http.StatusSeeOther)
		return
	}

	tmpl, err := template.ParseFiles("templates/group_create.html")
	if err != nil {
		http.Error(w, "Unable to load group creation page", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "Unable to display group creation page", http.StatusInternalServerError)
		return
	}
}
