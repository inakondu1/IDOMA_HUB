package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strconv"
)

func profileViewHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idText := r.URL.Query().Get("id")
	profileID, err := strconv.Atoi(idText)
	if err != nil || profileID <= 0 {
		http.Error(w, "Invalid profile.", http.StatusBadRequest)
		return
	}

	if profileID == userID {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}

	var username string
	var profilePicture sql.NullString

	err = db.QueryRow(
		"SELECT username, profile_picture FROM users WHERE id = ?",
		profileID,
	).Scan(&username, &profilePicture)

	if err != nil {
		http.Error(w, "User not found.", http.StatusNotFound)
		log.Println(err)
		return
	}

	profilePictureURL := ""
	if profilePicture.Valid {
		profilePictureURL = profilePicture.String
	}

	rows, err := db.Query(`
                SELECT posts.id, posts.content, posts.created_at,
                       posts.media_url, posts.media_type,
                       COUNT(post_likes.id) AS like_count
                FROM posts
                LEFT JOIN post_likes ON post_likes.post_id = posts.id
                WHERE posts.user_id = ?
                GROUP BY posts.id
                ORDER BY posts.id DESC
        `, profileID)

	if err != nil {
		http.Error(w, "Unable to load posts.", http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer rows.Close()

	var posts []ProfilePost

	for rows.Next() {
		var post ProfilePost

		err := rows.Scan(
			&post.ID,
			&post.Content,
			&post.CreatedAt,
			&post.MediaURL,
			&post.MediaType,
			&post.LikeCount,
		)

		if err != nil {
			http.Error(w, "Unable to read posts.", http.StatusInternalServerError)
			log.Println(err)
			return
		}

		post.CreatedAt = formatDateTime(post.CreatedAt)
		posts = append(posts, post)
	}

	tmpl, err := template.ParseFiles("templates/profile_view.html")
	if err != nil {
		http.Error(w, "Unable to load profile.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	data := struct {
		Title          string
		Username       string
		ProfilePicture string
		Posts          []ProfilePost
	}{
		Title:          username + " - IDOMA HUB",
		Username:       username,
		ProfilePicture: profilePictureURL,
		Posts:          posts,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Unable to display profile.", http.StatusInternalServerError)
		log.Println(err)
	}
}
