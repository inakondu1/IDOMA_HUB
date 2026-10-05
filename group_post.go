package main

import (
	"net/http"
	"strconv"
	"strings"
)

func groupPostHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/groups", http.StatusSeeOther)
		return
	}

	groupID, err := strconv.Atoi(r.FormValue("group_id"))
	if err != nil || groupID <= 0 {
		http.Error(w, "Invalid group", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))

	if content == "" {
		http.Error(w, "Post cannot be empty", http.StatusBadRequest)
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

	if !isMember {
		http.Error(w, "You must join the group before posting", http.StatusForbidden)
		return
	}

	_, err = db.Exec(`
		INSERT INTO group_posts (group_id, user_id, content)
		VALUES ($1, $2, $3)
	`, groupID, userID, content)

	if err != nil {
		http.Error(w, "Unable to create group post", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/groups/view?id="+strconv.Itoa(groupID), http.StatusSeeOther)
}
