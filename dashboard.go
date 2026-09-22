package main

import (
	"html/template"
	"log"
	"net/http"
)

type Post struct {
	ID        int
	Username  string
	Content   string
	CreatedAt string
}

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

	rows, err := db.Query(`
		SELECT posts.id, users.username, posts.content, posts.created_at
		FROM posts
		JOIN users ON users.id = posts.user_id
		ORDER BY posts.id DESC
	`)

	if err != nil {
		http.Error(w, "Unable to load posts.", http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer rows.Close()

	var posts []Post

	for rows.Next() {
		var post Post

		err := rows.Scan(
			&post.ID,
			&post.Username,
			&post.Content,
			&post.CreatedAt,
		)

		if err != nil {
			http.Error(w, "Unable to read posts.", http.StatusInternalServerError)
			log.Println(err)
			return
		}

		posts = append(posts, post)
	}

	data := struct {
		Title    string
		Username string
		Posts    []Post
	}{
		Title:    "IDOMA HUB - Dashboard",
		Username: username,
		Posts:    posts,
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

func createPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	content := r.FormValue("content")

	if content == "" {
		http.Error(w, "Post cannot be empty.", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(
		"INSERT INTO posts (user_id, content) VALUES (?, ?)",
		userID,
		content,
	)

	if err != nil {
		http.Error(w, "Unable to create post.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
