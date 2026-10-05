package main

import (
	"html/template"
	"net/http"
)

type Group struct {
	ID          int
	Name        string
	Description string
	Creator     string
	MemberCount int
	IsMember    bool
}

type GroupsPageData struct {
	Groups []Group
}

func groupsHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		action := r.FormValue("action")
		groupID := r.FormValue("group_id")

		switch action {
		case "join":
			_, _ = db.Exec(`
				INSERT INTO group_members (group_id, user_id)
				VALUES ($1, $2)
				ON CONFLICT DO NOTHING
			`, groupID, userID)

		case "leave":
			_, _ = db.Exec(`
				DELETE FROM group_members
				WHERE group_id = $1 AND user_id = $2
			`, groupID, userID)
		}

		http.Redirect(w, r, "/groups", http.StatusSeeOther)
		return
	}

	rows, err := db.Query(`
		SELECT
			g.id,
			g.name,
			g.description,
			u.username,
			COUNT(gm.id) AS member_count,
			EXISTS (
				SELECT 1
				FROM group_members gm2
				WHERE gm2.group_id = g.id
				AND gm2.user_id = $1
			) AS is_member
		FROM groups g
		JOIN users u ON u.id = g.creator_id
		LEFT JOIN group_members gm ON gm.group_id = g.id
		GROUP BY g.id, g.name, g.description, u.username
		ORDER BY g.created_at DESC
	`, userID)

	if err != nil {
		http.Error(w, "Unable to load groups", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	data := GroupsPageData{}

	for rows.Next() {
		var group Group

		if err := rows.Scan(
			&group.ID,
			&group.Name,
			&group.Description,
			&group.Creator,
			&group.MemberCount,
			&group.IsMember,
		); err == nil {
			data.Groups = append(data.Groups, group)
		}
	}

	tmpl, err := template.ParseFiles("templates/groups.html")
	if err != nil {
		http.Error(w, "Unable to load groups page", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Unable to display groups", http.StatusInternalServerError)
		return
	}
}
