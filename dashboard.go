package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Status struct {
	ID               int
	UserID           int
	Username         string
	ProfilePicture   string
	Content          string
	MediaURL         string
	MediaType        string
	CreatedAt        string
	OriginalStatusID int
	OriginalUserID   int
	OriginalUsername string
	ReshareMessage   string
}

type Post struct {
	ID             int
	UserID         int
	Username       string
	ProfilePicture string
	Content        string
	CreatedAt      string
	LikeCount      int
	CommentCount   int
	ShareCount     int
	LikedByMe      bool
	MediaURL       string
	MediaType      string
	Comments       []Comment
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
	ID             int
	PostID         int
	UserID         int
	Username       string
	ProfilePicture string
	Content        string
	CreatedAt      string
}

func newFeedPostCountHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Error(w, "Please log in.", http.StatusUnauthorized)
		return
	}
	_ = userID

	afterText := r.URL.Query().Get("after")
	afterID := 0

	if afterText != "" {
		parsedID, err := strconv.Atoi(afterText)
		if err != nil || parsedID < 0 {
			http.Error(w, "Invalid post ID.", http.StatusBadRequest)
			return
		}
		afterID = parsedID
	}

	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM posts WHERE id > $1",
		afterID,
	).Scan(&count)
	if err != nil {
		log.Printf("newFeedPostCountHandler: unable to count new posts: %v", err)
		http.Error(w, "Unable to check for new posts.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(struct {
		Count int `json:"count"`
	}{Count: count}); err != nil {
		log.Printf("newFeedPostCountHandler: unable to send response: %v", err)
	}
}

func latestDashboardPostID(posts []Post) int {
	if len(posts) == 0 {
		return 0
	}
	return posts[0].ID
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	updateLastActive(r)

	var username string
	var profilePicture string

	err := db.QueryRow(
		"SELECT username, COALESCE(profile_picture, '') FROM users WHERE id = $1",
		userID,
	).Scan(&username, &profilePicture)

	if err != nil {
		http.Error(w, "Unable to load your account.", http.StatusInternalServerError)
		log.Printf("dashboardHandler: account query failed for userID=%d: %v", userID, err)
		return
	}

	tmpl, err := template.ParseFiles("templates/dashboard.html")
	if err != nil {
		http.Error(w, "Unable to load dashboard.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	rows, err := db.Query(`
		SELECT posts.id, posts.user_id, users.username, COALESCE(users.profile_picture, ''), posts.content, posts.created_at,
                       posts.media_url, posts.media_type,
		       COUNT(post_likes.id) AS like_count,
                    (SELECT COUNT(*) FROM comments WHERE comments.post_id = posts.id) AS comment_count,
                    (SELECT COUNT(*) FROM posts shares WHERE shares.original_post_id = posts.id) AS share_count,
		       EXISTS (
			       SELECT 1
			       FROM post_likes user_like
			       WHERE user_like.post_id = posts.id
			       AND user_like.user_id = $1
		       ) AS liked_by_me
		FROM posts
		JOIN users ON users.id = posts.user_id
		LEFT JOIN post_likes ON post_likes.post_id = posts.id
		GROUP BY posts.id, posts.user_id, users.username, users.profile_picture, posts.content, posts.created_at, posts.media_url, posts.media_type
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
			&post.UserID,
			&post.Username,
			&post.ProfilePicture,
			&post.Content,
			&post.CreatedAt,
			&post.MediaURL,
			&post.MediaType,
			&post.LikeCount,
			&post.CommentCount,
			&post.ShareCount,
			&post.LikedByMe,
		)

		if err != nil {
			http.Error(w, "Unable to read posts.", http.StatusInternalServerError)
			log.Println(err)
			return
		}

		commentRows, err := db.Query(`
                        SELECT comments.id, comments.post_id, comments.user_id, users.username, COALESCE(users.profile_picture, ''),
                               comments.content, comments.created_at
                        FROM comments
                        JOIN users ON users.id = comments.user_id
                        WHERE comments.post_id = $1
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
				&comment.UserID,
				&comment.Username,
				&comment.ProfilePicture,
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

	var statuses []Status

	statusRows, err := db.Query(`
                SELECT statuses.id,
                       statuses.user_id,
                       users.username,
                       COALESCE(users.profile_picture, ''),
                       COALESCE(statuses.content, ''),
                       COALESCE(statuses.media_url, ''),
                       COALESCE(statuses.media_type, ''),
                       statuses.created_at,
                       COALESCE(statuses.original_status_id, 0),
                       COALESCE(statuses.original_user_id, 0),
                       COALESCE(statuses.original_username, ''),
                       COALESCE(statuses.reshare_message, '')
                FROM statuses
                JOIN users ON users.id = statuses.user_id
                WHERE statuses.expires_at > CURRENT_TIMESTAMP
                ORDER BY statuses.created_at DESC
        `)

	if err != nil {
		http.Error(w, "Unable to load statuses.", http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer statusRows.Close()

	for statusRows.Next() {
		var status Status

		err := statusRows.Scan(
			&status.ID,
			&status.UserID,
			&status.Username,
			&status.ProfilePicture,
			&status.Content,
			&status.MediaURL,
			&status.MediaType,
			&status.CreatedAt,
			&status.OriginalStatusID,
			&status.OriginalUserID,
			&status.OriginalUsername,
			&status.ReshareMessage,
		)

		if err != nil {
			http.Error(w, "Unable to read statuses.", http.StatusInternalServerError)
			log.Println(err)
			return
		}

		status.CreatedAt = formatDateTime(status.CreatedAt)
		statuses = append(statuses, status)
	}

	var notificationCount int

	err = db.QueryRow(`
            SELECT
                (SELECT COUNT(*) FROM friend_requests WHERE receiver_id = $1 AND status = 'pending')
                +
                (SELECT COUNT(*) FROM notifications WHERE recipient_id = $2 AND is_read = 0)
    `, userID, userID).Scan(&notificationCount)

	if err != nil {
		http.Error(w, "Unable to load notifications.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	var messageCount int

	err = db.QueryRow("SELECT COUNT(*) FROM messages WHERE receiver_id = $1 AND is_read = 0", userID).Scan(&messageCount)
	if err != nil {
		http.Error(w, "Unable to load message notifications.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	var suggestions []FriendUser

	suggestionRows, suggestionErr := db.Query(`
		SELECT u.id, u.username, COALESCE(u.profile_picture, '')
		FROM users u
		WHERE u.id <> $1
		AND NOT EXISTS (
			SELECT 1
			FROM friend_requests fr
			WHERE (
				(fr.sender_id = $1 AND fr.receiver_id = u.id)
				OR (fr.sender_id = u.id AND fr.receiver_id = $1)
			)
			AND fr.status IN ('pending', 'accepted')
		)
		ORDER BY u.username
		LIMIT 5
	`, userID)

	if suggestionErr == nil {
		for suggestionRows.Next() {
			var suggestion FriendUser
			if suggestionRows.Scan(&suggestion.ID, &suggestion.Username, &suggestion.ProfilePicture) == nil {
				suggestions = append(suggestions, suggestion)
			}
		}
		if err := suggestionRows.Err(); err != nil {
			log.Println("Unable to load dashboard suggestions:", err)
		}
		suggestionRows.Close()
	} else {
		log.Println("Unable to load dashboard suggestions:", suggestionErr)
	}

	var incomingRequests []FriendRequest
	requestRows, requestErr := db.Query(`
        SELECT fr.id, u.id, u.username, COALESCE(u.profile_picture, '')
        FROM friend_requests fr
        JOIN users u ON u.id = fr.sender_id
        WHERE fr.receiver_id = $1 AND fr.status = 'pending'
        ORDER BY fr.created_at DESC
        LIMIT 5
    `, userID)
	if requestErr != nil {
		log.Println("Unable to load incoming dashboard friend requests:", requestErr)
	} else {
		for requestRows.Next() {
			var request FriendRequest
			if scanErr := requestRows.Scan(
				&request.ID,
				&request.UserID,
				&request.Username,
				&request.ProfilePicture,
			); scanErr != nil {
				log.Println("Unable to read dashboard friend request:", scanErr)
				continue
			}
			incomingRequests = append(incomingRequests, request)
		}
		if rowsErr := requestRows.Err(); rowsErr != nil {
			log.Println("Error reading dashboard friend requests:", rowsErr)
		}
		requestRows.Close()
	}

	data := struct {
		Title                string
		UserID               int
		Username             string
		ProfilePicture       string
		Posts                []Post
		LatestPostID         int
		Statuses             []Status
		Suggestions          []FriendUser
		Incoming             []FriendRequest
		NotificationCount    int
		MessageCount         int
		OnlineCount          int
		AnonymousOnlineCount int
	}{
		Title:                "IDOMA HUB - Dashboard",
		UserID:               userID,
		Username:             username,
		ProfilePicture:       profilePicture,
		Posts:                posts,
		LatestPostID:         latestDashboardPostID(posts),
		Statuses:             statuses,
		Suggestions:          suggestions,
		Incoming:             incomingRequests,
		NotificationCount:    notificationCount,
		MessageCount:         messageCount,
		OnlineCount:          getOnlineCount(),
		AnonymousOnlineCount: anonymousOnlineCount(),
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

		updateLastActive(r)
	}

	err := r.ParseMultipartForm(50 << 20)
	if err != nil {
		http.Error(w, "Unable to process upload.", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))

	category := strings.TrimSpace(r.FormValue("category"))
	if category != "sports" && category != "gossip" {
		category = "general"
	}

	var mediaURL string
	var mediaType string

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

		mediaURL, err = uploadToCloudinary(photo, "idoma_hub/posts", "image")
		if err != nil {
			http.Error(w, "Unable to upload photo.", http.StatusInternalServerError)
			log.Println(err)
			return
		}
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

		mediaURL, err = uploadToCloudinary(video, "idoma_hub/posts", "video")
		if err != nil {
			http.Error(w, "Unable to upload video.", http.StatusInternalServerError)
			log.Println(err)
			return
		}
	}

	if content == "" && mediaURL == "" {
		http.Error(w, "Post cannot be empty.", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(
		"INSERT INTO posts (user_id, content, media_url, media_type, category) VALUES ($1, $2, $3, $4, $5)",
		userID,
		content,
		mediaURL,
		mediaType,
		category,
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

		updateLastActive(r)
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
                WHERE id = $1
        `, postID).Scan(&originalUserID, &content, &mediaURL, &mediaType)

	if err != nil {
		http.Error(w, "Original post not found.", http.StatusNotFound)
		return
	}

	_, err = db.Exec(`
                INSERT INTO posts
                (user_id, content, media_url, media_type, original_post_id)
                VALUES ($1, $2, $3, $4, $5)
        `, userID, content, mediaURL, mediaType, postID)

	if err != nil {
		http.Error(w, "Unable to reshare post.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	createPostNotification(originalUserID, userID, postID, "share")

	returnTo := r.FormValue("return_to")
	if returnTo != "/sports" && returnTo != "/gossip" &&
		!(len(returnTo) > len("/dashboard#post-") &&
			returnTo[:len("/dashboard#post-")] == "/dashboard#post-") {
		returnTo = "/dashboard"
	}

	http.Redirect(w, r, returnTo, http.StatusSeeOther)
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

	updateLastActive(r)

	postID := r.FormValue("post_id")

	if postID == "" {
		http.Error(w, "Post ID is required.", http.StatusBadRequest)
		return
	}

	var liked bool

	err := db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM post_likes WHERE post_id = $1 AND user_id = $2)",
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
			"DELETE FROM post_likes WHERE post_id = $1 AND user_id = $2",
			postID,
			userID,
		)
	} else {
		_, err = db.Exec(
			"INSERT INTO post_likes (post_id, user_id) VALUES ($1, $2)",
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
			"SELECT user_id FROM posts WHERE id = $1",
			postID,
		).Scan(&postOwnerID)

		if err == nil && postOwnerID != userID {
			recordRewardActivity(userID, postID, "like", "post:"+postID)
			_, err = db.Exec(`
				INSERT INTO notifications
				(recipient_id, sender_id, post_id, type)
				VALUES ($1, $2, $3, 'like')
			`, postOwnerID, userID, postID)

			if err != nil {
				log.Println("Unable to create like notification:", err)
			}
		}
	}

	returnTo := r.FormValue("return_to")
	if returnTo != "/sports" && returnTo != "/gossip" &&
		!(len(returnTo) > len("/dashboard#post-") &&
			returnTo[:len("/dashboard#post-")] == "/dashboard#post-") {
		returnTo = "/dashboard"
	}

	http.Redirect(w, r, returnTo, http.StatusSeeOther)
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

		updateLastActive(r)
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

	mediaURL, err := uploadToCloudinary(file, "idoma_hub/profile", "image")
	if err != nil {
		http.Error(w, "Unable to upload profile picture.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	var oldPicture string

	err = db.QueryRow(
		"SELECT COALESCE(profile_picture, '') FROM users WHERE id = $1",
		userID,
	).Scan(&oldPicture)

	if err != nil {
		http.Error(w, "Unable to load your current profile picture.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	_, err = db.Exec(
		"UPDATE users SET profile_picture = $1 WHERE id = $2",
		mediaURL,
		userID,
	)

	if err != nil {
		http.Error(w, "Unable to update your profile picture.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func editPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	postID := strings.TrimSpace(r.FormValue("post_id"))
	content := strings.TrimSpace(r.FormValue("content"))
	if postID == "" {
		http.Error(w, "Post ID is required.", http.StatusBadRequest)
		return
	}

	// Allow an empty caption only when the owned post already contains media.
	if content == "" {
		var mediaURL string
		err := db.QueryRow(
			"SELECT COALESCE(media_url, '') FROM posts WHERE id = $1 AND user_id = $2",
			postID, userID,
		).Scan(&mediaURL)
		if err != nil {
			http.Error(w, "Post not found or you do not own it.", http.StatusNotFound)
			return
		}
		if strings.TrimSpace(mediaURL) == "" {
			http.Error(w, "A post must contain text or media.", http.StatusBadRequest)
			return
		}
	}

	result, err := db.Exec(
		"UPDATE posts SET content = $1 WHERE id = $2 AND user_id = $3",
		content, postID, userID,
	)
	if err != nil {
		log.Println("Unable to edit post:", err)
		http.Error(w, "Unable to edit post.", http.StatusInternalServerError)
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		log.Println("Unable to verify edited post:", err)
		http.Error(w, "Unable to verify edited post.", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "Post not found or you do not own it.", http.StatusNotFound)
		return
	}

	returnTo := r.FormValue("return_to")
	if returnTo != "/dashboard" && !strings.HasPrefix(returnTo, "/dashboard#post-") {
		returnTo = "/dashboard"
	}
	http.Redirect(w, r, returnTo, http.StatusSeeOther)
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

		updateLastActive(r)
	}

	postID := r.FormValue("post_id")
	if postID == "" {
		http.Error(w, "Post ID is required.", http.StatusBadRequest)
		return
	}

	var mediaURL string

	err := db.QueryRow(
		"SELECT media_url FROM posts WHERE id = $1 AND user_id = $2",
		postID,
		userID,
	).Scan(&mediaURL)

	if err != nil {
		http.Error(w, "Post not found.", http.StatusNotFound)
		return
	}

	_, err = db.Exec("DELETE FROM comments WHERE post_id = $1", postID)
	if err != nil {
		http.Error(w, "Unable to delete post comments.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	_, err = db.Exec("DELETE FROM post_likes WHERE post_id = $1", postID)
	if err != nil {
		http.Error(w, "Unable to delete post likes.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	_, err = db.Exec(
		"DELETE FROM notifications WHERE post_id = $1", postID)
	if err != nil {
		http.Error(w, "Unable to delete post notifications.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	_, err = db.Exec(
		"DELETE FROM posts WHERE id = $1 AND user_id = $2",
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

	updateLastActive(r)

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

	var commentID int64
	err := db.QueryRow(
		"INSERT INTO comments (post_id, user_id, content) VALUES ($1, $2, $3) RETURNING id",
		postID,
		userID,
		content,
	).Scan(&commentID)

	if err != nil {
		http.Error(w, "Unable to create comment.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	var postOwnerID int
	err = db.QueryRow(
		"SELECT user_id FROM posts WHERE id = $1",
		postID,
	).Scan(&postOwnerID)

	if err == nil {
		if postOwnerID != userID {
			recordRewardActivity(
				userID,
				postID,
				"comment",
				"comment:"+strconv.FormatInt(commentID, 10),
			)
		}
		createPostNotification(postOwnerID, userID, postID, "comment")
	}

	returnTo := r.FormValue("return_to")
	if returnTo != "/sports" && returnTo != "/gossip" &&
		!(len(returnTo) > len("/dashboard#post-") &&
			returnTo[:len("/dashboard#post-")] == "/dashboard#post-") {
		returnTo = "/dashboard"
	}

	http.Redirect(w, r, returnTo, http.StatusSeeOther)
}
