package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
)

type ProfilePost struct {
	ID        int
	Content   string
	CreatedAt string
	LikeCount int
	MediaURL  string
	MediaType string
}

func profileHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var username string
	var email string
	var profilePicture sql.NullString

	err := db.QueryRow(
		"SELECT username, email, profile_picture FROM users WHERE id = ?",
		userID,
	).Scan(&username, &email, &profilePicture)

	if err != nil {
		http.Error(w, "Unable to load your profile.", http.StatusInternalServerError)
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
	`, userID)

	if err != nil {
		http.Error(w, "Unable to load your posts.", http.StatusInternalServerError)
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
			http.Error(w, "Unable to read your posts.", http.StatusInternalServerError)
			log.Println(err)
			return
		}

		post.CreatedAt = formatDateTime(post.CreatedAt)
		posts = append(posts, post)
	}

	tmpl, err := template.ParseFiles("templates/profile.html")

	if err != nil {
		http.Error(w, "Unable to load profile.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	data := struct {
		Title          string
		Username       string
		ProfilePicture string
		Email          string
		Posts          []ProfilePost
	}{
		Title:          "My Profile - IDOMA HUB",
		Username:       username,
		ProfilePicture: profilePictureURL,
		Email:          email,
		Posts:          posts,
	}

	err = tmpl.Execute(w, data)

	if err != nil {
		http.Error(w, "Unable to display profile.", http.StatusInternalServerError)
		log.Println(err)
	}
}
