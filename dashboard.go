package main

import (
	"html/template"
	"log"
	"net/http"
	"time"
)

type Post struct {
	ID        int
	Username  string
	Content   string
	CreatedAt string
	LikeCount int
	LikedByMe bool
	Comments  []Comment
}

func formatDateTime(value string) string {
	parsed, err := time.Parse("2006-01-02T15:04:05Z", value)
	if err != nil {
		return value
	}

	wat := time.FixedZone("WAT", 60*60)
	localTime := parsed.In(wat)
	now := time.Now().In(wat)

	if localTime.Year() == now.Year() &&
		localTime.YearDay() == now.YearDay() {
		return "Today at " + localTime.Format("3:04 PM")
	}

	yesterday := now.AddDate(0, 0, -1)
	if localTime.Year() == yesterday.Year() &&
		localTime.YearDay() == yesterday.YearDay() {
		return "Yesterday at " + localTime.Format("3:04 PM")
	}

	return localTime.Format("Jan 2 at 3:04 PM")
}

type Comment struct {
	ID        int
	PostID    int
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
		SELECT posts.id, users.username, posts.content, posts.created_at,
		       COUNT(post_likes.id) AS like_count,
		       EXISTS (
			       SELECT 1
			       FROM post_likes user_like
			       WHERE user_like.post_id = posts.id
			       AND user_like.user_id = ?
		       ) AS liked_by_me
		FROM posts
		JOIN users ON users.id = posts.user_id
		LEFT JOIN post_likes ON post_likes.post_id = posts.id
		GROUP BY posts.id
		ORDER BY posts.id DESC
	`, userID)

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
			&post.LikeCount,
			&post.LikedByMe,
		)

		if err != nil {
			http.Error(w, "Unable to read posts.", http.StatusInternalServerError)
			log.Println(err)
			return
		}

		commentRows, err := db.Query(`
                        SELECT comments.id, comments.post_id, users.username,
                               comments.content, comments.created_at
                        FROM comments
                        JOIN users ON users.id = comments.user_id
                        WHERE comments.post_id = ?
                        ORDER BY comments.id ASC
                `, post.ID)

		if err != nil {
			http.Error(w, "Unable to load comments.", http.StatusInternalServerError)
			log.Println(err)
			return
		}

		for commentRows.Next() {
			var comment Comment

			err := commentRows.Scan(
				&comment.ID,
				&comment.PostID,
				&comment.Username,
				&comment.Content,
				&comment.CreatedAt,
			)

			if err != nil {
				commentRows.Close()
				http.Error(w, "Unable to read comments.", http.StatusInternalServerError)
				log.Println(err)
				return
			}

			post.Comments = append(post.Comments, comment)
		}

		commentRows.Close()

		post.CreatedAt = formatDateTime(post.CreatedAt)

		for i := range post.Comments {
			post.Comments[i].CreatedAt = formatDateTime(post.Comments[i].CreatedAt)
		}

		posts = append(posts, post)
	}

	var notificationCount int

	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM friend_requests
		WHERE receiver_id = ?
		AND status = 'pending'
	`, userID).Scan(&notificationCount)

	if err != nil {
		http.Error(w, "Unable to load notifications.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	data := struct {
		Title             string
		Username          string
		Posts             []Post
		NotificationCount int
	}{
		Title:             "IDOMA HUB - Dashboard",
		Username:          username,
		Posts:             posts,
		NotificationCount: notificationCount,
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

func likePostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	postID := r.FormValue("post_id")

	if postID == "" {
		http.Error(w, "Post ID is required.", http.StatusBadRequest)
		return
	}

	var liked bool

	err := db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM post_likes WHERE post_id = ? AND user_id = ?)",
		postID,
		userID,
	).Scan(&liked)

	if err != nil {
		http.Error(w, "Unable to check like.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	if liked {
		_, err = db.Exec(
			"DELETE FROM post_likes WHERE post_id = ? AND user_id = ?",
			postID,
			userID,
		)
	} else {
		_, err = db.Exec(
			"INSERT INTO post_likes (post_id, user_id) VALUES (?, ?)",
			postID,
			userID,
		)
	}

	if err != nil {
		http.Error(w, "Unable to update like.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func createCommentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	postID := r.FormValue("post_id")
	content := r.FormValue("content")

	if postID == "" {
		http.Error(w, "Post ID is required.", http.StatusBadRequest)
		return
	}

	if content == "" {
		http.Error(w, "Comment cannot be empty.", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(
		"INSERT INTO comments (post_id, user_id, content) VALUES (?, ?, ?)",
		postID,
		userID,
		content,
	)

	if err != nil {
		http.Error(w, "Unable to create comment.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
