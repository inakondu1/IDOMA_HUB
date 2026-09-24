package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Post struct {
	ID        int
	Username  string
	Content   string
	CreatedAt string
	LikeCount int
	LikedByMe bool
	MediaURL  string
	MediaType string
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
                       posts.media_url, posts.media_type,
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
			&post.MediaURL,
			&post.MediaType,
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
            SELECT
                (SELECT COUNT(*) FROM friend_requests WHERE receiver_id = ? AND status = 'pending')
                +
                (SELECT COUNT(*) FROM notifications WHERE recipient_id = ? AND is_read = 0)
    `, userID, userID).Scan(&notificationCount)

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

func postUploadDir() string {
	if dir := os.Getenv("UPLOAD_DIR"); dir != "" {
		return dir
	}
	return filepath.Join("static", "uploads", "posts")
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

	err := r.ParseMultipartForm(50 << 20)
	if err != nil {
		http.Error(w, "Unable to process upload.", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))

	var mediaURL string
	var mediaType string

	uploadDir := postUploadDir()
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		http.Error(w, "Unable to prepare upload storage.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	photo, photoHeader, photoErr := r.FormFile("photo")
	video, videoHeader, videoErr := r.FormFile("video")

	if photoErr == nil && videoErr == nil {
		photo.Close()
		video.Close()
		http.Error(w, "Please upload only one media file at a time.", http.StatusBadRequest)
		return
	}

	if photoErr == nil {
		defer photo.Close()

		ext := strings.ToLower(filepath.Ext(photoHeader.Filename))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".gif", ".webp":
			mediaType = "image"
		default:
			http.Error(w, "Unsupported photo format.", http.StatusBadRequest)
			return
		}

		filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
		filePath := filepath.Join(postUploadDir(), filename)

		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Unable to save photo.", http.StatusInternalServerError)
			log.Println(err)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, photo); err != nil {
			http.Error(w, "Unable to save photo.", http.StatusInternalServerError)
			log.Println(err)
			return
		}

		mediaURL = "/static/uploads/posts/" + filename
	}

	if videoErr == nil {
		defer video.Close()

		ext := strings.ToLower(filepath.Ext(videoHeader.Filename))
		switch ext {
		case ".mp4", ".webm", ".ogg":
			mediaType = "video"
		default:
			http.Error(w, "Unsupported video format.", http.StatusBadRequest)
			return
		}

		filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
		filePath := filepath.Join(postUploadDir(), filename)

		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Unable to save video.", http.StatusInternalServerError)
			log.Println(err)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, video); err != nil {
			http.Error(w, "Unable to save video.", http.StatusInternalServerError)
			log.Println(err)
			return
		}

		mediaURL = "/static/uploads/posts/" + filename
	}

	if content == "" && mediaURL == "" {
		http.Error(w, "Post cannot be empty.", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(
		"INSERT INTO posts (user_id, content, media_url, media_type) VALUES (?, ?, ?, ?)",
		userID,
		content,
		mediaURL,
		mediaType,
	)

	if err != nil {
		http.Error(w, "Unable to create post.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func sharePostHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	postID := r.FormValue("post_id")
	if postID == "" {
		http.Error(w, "Post ID is required.", http.StatusBadRequest)
		return
	}

	var originalUserID int
	var content string
	var mediaURL string
	var mediaType string

	err := db.QueryRow(`
                SELECT user_id, content, media_url, media_type
                FROM posts
                WHERE id = ?
        `, postID).Scan(&originalUserID, &content, &mediaURL, &mediaType)

	if err != nil {
		http.Error(w, "Original post not found.", http.StatusNotFound)
		return
	}

	_, err = db.Exec(`
                INSERT INTO posts
                (user_id, content, media_url, media_type, original_post_id)
                VALUES (?, ?, ?, ?, ?)
        `, userID, content, mediaURL, mediaType, postID)

	if err != nil {
		http.Error(w, "Unable to reshare post.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	createPostNotification(originalUserID, userID, postID, "share")

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

	if !liked {
		var postOwnerID int

		err = db.QueryRow(
			"SELECT user_id FROM posts WHERE id = ?",
			postID,
		).Scan(&postOwnerID)

		if err == nil && postOwnerID != userID {
			_, err = db.Exec(`
				INSERT INTO notifications
				(recipient_id, sender_id, post_id, type)
				VALUES (?, ?, ?, 'like')
			`, postOwnerID, userID, postID)

			if err != nil {
				log.Println("Unable to create like notification:", err)
			}
		}
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func profilePictureUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	err := r.ParseMultipartForm(5 << 20)
	if err != nil {
		http.Error(w, "Unable to process image upload.", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("profile_picture")
	if err != nil {
		http.Error(w, "Please choose an image.", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))

	allowed := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}

	if !allowed[ext] {
		http.Error(w, "Only JPG, JPEG, PNG, GIF, and WEBP images are allowed.", http.StatusBadRequest)
		return
	}

	if err := os.MkdirAll("static/uploads/profile", 0755); err != nil {
		http.Error(w, "Unable to create upload folder.", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("profile_%d_%d%s", userID, time.Now().UnixNano(), ext)
	filePath := filepath.Join("static/uploads/profile", filename)
	mediaURL := "/static/uploads/profile/" + filename

	destination, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Unable to save profile picture.", http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer destination.Close()

	if _, err := io.Copy(destination, file); err != nil {
		http.Error(w, "Unable to save profile picture.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	var oldPicture string

	err = db.QueryRow(
		"SELECT COALESCE(profile_picture, '' ) FROM users WHERE id = ?",
		userID,
	).Scan(&oldPicture)

	if err != nil {
		http.Error(w, "Unable to load your current profile picture.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	_, err = db.Exec(
		"UPDATE users SET profile_picture = ? WHERE id = ?",
		mediaURL,
		userID,
	)

	if err != nil {
		http.Error(w, "Unable to update your profile picture.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	if oldPicture != "" {
		prefix := "/static/uploads/profile/"
		if strings.HasPrefix(oldPicture, prefix) {
			oldFilename := strings.TrimPrefix(oldPicture, prefix)

			if !strings.Contains(oldFilename, "..") && !strings.Contains(oldFilename, "/") {
				oldPath := filepath.Join("static/uploads/profile", oldFilename)

				if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
					log.Println("Unable to delete old profile picture:", err)
				}
			}
		}
	}

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func deletePostHandler(w http.ResponseWriter, r *http.Request) {
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

	var mediaURL string

	err := db.QueryRow(
		"SELECT media_url FROM posts WHERE id = ? AND user_id = ?",
		postID,
		userID,
	).Scan(&mediaURL)

	if err != nil {
		http.Error(w, "Post not found.", http.StatusNotFound)
		return
	}

	_, err = db.Exec("DELETE FROM comments WHERE post_id = ?", postID)
	if err != nil {
		http.Error(w, "Unable to delete post comments.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	_, err = db.Exec("DELETE FROM post_likes WHERE post_id = ?", postID)
	if err != nil {
		http.Error(w, "Unable to delete post likes.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	_, err = db.Exec(
		"DELETE FROM posts WHERE id = ? AND user_id = ?",
		postID,
		userID,
	)
	if err != nil {
		http.Error(w, "Unable to delete post.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	if mediaURL != "" {
		prefix := "/static/uploads/posts/"
		if strings.HasPrefix(mediaURL, prefix) {
			filename := strings.TrimPrefix(mediaURL, prefix)

			if !strings.Contains(filename, "..") && !strings.Contains(filename, "/") {
				filePath := filepath.Join(postUploadDir(), filename)

				if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
					log.Println("Unable to delete post media:", err)
				}
			}
		}
	}

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
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

	var postOwnerID int
	err = db.QueryRow(
		"SELECT user_id FROM posts WHERE id = ?",
		postID,
	).Scan(&postOwnerID)

	if err == nil {
		createPostNotification(postOwnerID, userID, postID, "comment")
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
