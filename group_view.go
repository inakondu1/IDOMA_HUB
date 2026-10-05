package main

import (
	"html/template"
	"net/http"
	"strconv"
)

type GroupPost struct {
	Username       string
	ProfilePicture string
	Content        string
	CreatedAt      string
}

type GroupMember struct {
	Username       string
	ProfilePicture string
}

type GroupViewData struct {
	Group    Group
	Members  []GroupMember
	Posts    []GroupPost
	IsMember bool
	CanPost  bool
}

func groupViewHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	groupID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || groupID <= 0 {
		http.Error(w, "Invalid group", http.StatusBadRequest)
		return
	}

	var group Group

	err = db.QueryRow(`
		SELECT
			g.id,
			g.name,
			g.description,
			u.username,
			COUNT(gm.id)
		FROM groups g
		JOIN users u ON u.id = g.creator_id
		LEFT JOIN group_members gm ON gm.group_id = g.id
		WHERE g.id = $1
		GROUP BY g.id, g.name, g.description, u.username
	`, groupID).Scan(
		&group.ID,
		&group.Name,
		&group.Description,
		&group.Creator,
		&group.MemberCount,
	)

	if err != nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	var isMember bool

	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM group_members
			WHERE group_id = $1 AND user_id = $2
		)
	`, groupID, userID).Scan(&isMember)

	if err != nil {
		http.Error(w, "Unable to check group membership", http.StatusInternalServerError)
		return
	}

	group.IsMember = isMember

	memberRows, err := db.Query(`
		SELECT u.username, COALESCE(u.profile_picture, '')
		FROM group_members gm
		JOIN users u ON u.id = gm.user_id
		WHERE gm.group_id = $1
		ORDER BY gm.joined_at ASC
	`, groupID)

	if err != nil {
		http.Error(w, "Unable to load group members", http.StatusInternalServerError)
		return
	}
	defer memberRows.Close()

	data := GroupViewData{
		Group:    group,
		IsMember: isMember,
		CanPost:  isMember,
	}

	for memberRows.Next() {
		var member GroupMember

		if err := memberRows.Scan(
			&member.Username,
			&member.ProfilePicture,
		); err == nil {
			data.Members = append(data.Members, member)
		}
	}

	postRows, err := db.Query(`
		SELECT
			u.username,
			COALESCE(u.profile_picture, ''),
			gp.content,
			TO_CHAR(gp.created_at, 'Mon DD, YYYY HH24:MI')
		FROM group_posts gp
		JOIN users u ON u.id = gp.user_id
		WHERE gp.group_id = $1
		ORDER BY gp.created_at DESC
	`, groupID)

	if err != nil {
		http.Error(w, "Unable to load group posts", http.StatusInternalServerError)
		return
	}
	defer postRows.Close()

	for postRows.Next() {
		var post GroupPost

		if err := postRows.Scan(
			&post.Username,
			&post.ProfilePicture,
			&post.Content,
			&post.CreatedAt,
		); err == nil {
			data.Posts = append(data.Posts, post)
		}
	}

	tmpl, err := template.ParseFiles("templates/group_view.html")
	if err != nil {
		http.Error(w, "Unable to load group page", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Unable to display group page", http.StatusInternalServerError)
		return
	}
}
