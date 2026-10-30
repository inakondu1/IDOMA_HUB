package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type AdminDashboardData struct {
	Title       string
	Role        string
	Message     string
	Users       int64
	Posts       int64
	Comments    int64
	PendingApps int64
	Query       string
	SearchUsers []AdminUserRow
	RecentPosts []AdminPostRow
	Audit       []AdminAuditRow
}

type AdminUserRow struct {
	ID         int
	Username   string
	Email      string
	CreatedAt  string
	LastActive string
}

type AdminPostRow struct {
	ID        int64
	UserID    int
	Username  string
	Content   string
	MediaURL  string
	MediaType string
	Category  string
	CreatedAt string
	Comments  int64
	Likes     int64
}

type AdminAuditRow struct {
	Actor     string
	Target    string
	Action    string
	Details   string
	CreatedAt string
}

func adminDashboardHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	role := getAdminRole(userID)
	if role != "owner" && role != "admin" {
		http.Error(w, "You are not authorized to access this page.", http.StatusForbidden)
		return
	}

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	data := AdminDashboardData{
		Title: "Admin Dashboard - IDOMA HUB",
		Role:  role,
		Query: strings.TrimSpace(r.URL.Query().Get("q")),
	}

	if r.Method == http.MethodPost {
		if !validSameOriginRequest(r) {
			http.Error(w, "Invalid request origin.", http.StatusForbidden)
			return
		}

		action := strings.TrimSpace(r.FormValue("action"))
		switch action {
		case "delete_comment":
			commentID, err := strconv.ParseInt(r.FormValue("comment_id"), 10, 64)
			if err != nil || commentID <= 0 {
				http.Error(w, "Invalid comment ID.", http.StatusBadRequest)
				return
			}
			if err := adminDeleteComment(userID, commentID); err != nil {
				log.Println("Admin comment moderation failed:", err)
				http.Error(w, "Unable to remove this comment. Check the server log.", http.StatusInternalServerError)
				return
			}
			data.Message = "Comment removed and action recorded."

		case "delete_post":
			postID, err := strconv.ParseInt(r.FormValue("post_id"), 10, 64)
			if err != nil || postID <= 0 {
				http.Error(w, "Invalid post ID.", http.StatusBadRequest)
				return
			}
			if err := adminDeletePost(userID, postID); err != nil {
				log.Println("Admin post moderation failed:", err)
				http.Error(w, "Unable to remove this post. Check the server log.", http.StatusInternalServerError)
				return
			}
			data.Message = "Post removed and action recorded."

		default:
			http.Error(w, "Unknown admin action.", http.StatusBadRequest)
			return
		}
	}

	if err := loadAdminDashboard(&data); err != nil {
		log.Println("Unable to load admin dashboard:", err)
		http.Error(w, "Unable to load admin dashboard. Check the server log.", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("templates/admin.html")
	if err != nil {
		log.Println("Unable to parse admin dashboard template:", err)
		http.Error(w, "Unable to load admin dashboard.", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Println("Unable to render admin dashboard:", err)
	}
}

func loadAdminDashboard(data *AdminDashboardData) error {
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&data.Users); err != nil {
		return err
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM posts").Scan(&data.Posts); err != nil {
		return err
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM comments").Scan(&data.Comments); err != nil {
		return err
	}
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM creator_monetization_applications
		WHERE status = 'pending'
	`).Scan(&data.PendingApps); err != nil {
		return err
	}

	if data.Query != "" {
		rows, err := db.Query(`
			SELECT id, username, email,
			       COALESCE(TO_CHAR(created_at, 'YYYY-MM-DD HH24:MI'), ''),
			       COALESCE(TO_CHAR(last_active, 'YYYY-MM-DD HH24:MI'), '')
			FROM users
			WHERE username ILIKE $1 OR email ILIKE $1
			ORDER BY created_at DESC
			LIMIT 25
		`, "%"+data.Query+"%")
		if err != nil {
			return err
		}
		for rows.Next() {
			var u AdminUserRow
			if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.CreatedAt, &u.LastActive); err != nil {
				rows.Close()
				return err
			}
			data.SearchUsers = append(data.SearchUsers, u)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
	}

	rows, err := db.Query(`
		SELECT p.id, p.user_id, u.username,
		       COALESCE(p.content, ''),
		       COALESCE(p.media_url, ''),
		       COALESCE(p.media_type, ''),
		       COALESCE(p.category, 'general'),
		       COALESCE(TO_CHAR(p.created_at, 'YYYY-MM-DD HH24:MI'), ''),
		       (SELECT COUNT(*) FROM comments c WHERE c.post_id = p.id),
		       (SELECT COUNT(*) FROM post_likes l WHERE l.post_id = p.id)
		FROM posts p
		JOIN users u ON u.id = p.user_id
		ORDER BY p.created_at DESC
		LIMIT 20
	`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var p AdminPostRow
		if err := rows.Scan(
			&p.ID, &p.UserID, &p.Username, &p.Content,
			&p.MediaURL, &p.MediaType, &p.Category, &p.CreatedAt,
			&p.Comments, &p.Likes,
		); err != nil {
			rows.Close()
			return err
		}
		data.RecentPosts = append(data.RecentPosts, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	auditRows, err := db.Query(`
		SELECT COALESCE(actor.username, 'System'),
		       COALESCE(target.username, 'Unknown user'),
		       a.action,
		       COALESCE(a.details, ''),
		       COALESCE(TO_CHAR(a.created_at, 'YYYY-MM-DD HH24:MI'), '')
		FROM admin_audit_log a
		LEFT JOIN users actor ON actor.id = a.actor_id
		LEFT JOIN users target ON target.id = a.target_user_id
		ORDER BY a.created_at DESC
		LIMIT 15
	`)
	if err != nil {
		return err
	}
	for auditRows.Next() {
		var a AdminAuditRow
		if err := auditRows.Scan(&a.Actor, &a.Target, &a.Action, &a.Details, &a.CreatedAt); err != nil {
			auditRows.Close()
			return err
		}
		data.Audit = append(data.Audit, a)
	}
	if err := auditRows.Err(); err != nil {
		auditRows.Close()
		return err
	}
	return auditRows.Close()
}

func adminDeleteComment(actorID int, commentID int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var targetUserID int
	var postID int64
	err = tx.QueryRow(`
		DELETE FROM comments
		WHERE id = $1
		RETURNING user_id, post_id
	`, commentID).Scan(&targetUserID, &postID)
	if err != nil {
		return err
	}

	details := "comment_id=" + strconv.FormatInt(commentID, 10) +
		"; post_id=" + strconv.FormatInt(postID, 10)
	if _, err := tx.Exec(`
		INSERT INTO admin_audit_log (actor_id, target_user_id, action, details)
		VALUES ($1, $2, 'delete_comment', $3)
	`, actorID, targetUserID, details); err != nil {
		return err
	}

	return tx.Commit()
}

func adminDeletePost(actorID int, postID int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var targetUserID int
	var mediaURL, mediaType, mediaPublicID string
	err = tx.QueryRow(`
		SELECT user_id,
		       COALESCE(media_url, ''),
		       COALESCE(media_type, ''),
		       COALESCE(media_public_id, '')
		FROM posts
		WHERE id = $1
		FOR UPDATE
	`, postID).Scan(&targetUserID, &mediaURL, &mediaType, &mediaPublicID)
	if err != nil {
		return err
	}

	for _, query := range []string{
		`DELETE FROM comments WHERE post_id = $1`,
		`DELETE FROM post_likes WHERE post_id = $1`,
		`DELETE FROM notifications WHERE post_id = $1`,
		`DELETE FROM reward_activity WHERE post_id = $1`,
		`DELETE FROM video_watch_sessions WHERE post_id = $1`,
	} {
		if _, err := tx.Exec(query, postID); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`DELETE FROM posts WHERE id = $1`, postID); err != nil {
		return err
	}

	details := "post_id=" + strconv.FormatInt(postID, 10)
	if _, err := tx.Exec(`
		INSERT INTO admin_audit_log (actor_id, target_user_id, action, details)
		VALUES ($1, $2, 'delete_post', $3)
	`, actorID, targetUserID, details); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	cleanupDeletedPostMedia(mediaURL, mediaType, mediaPublicID)
	return nil
}

// Keep this reference so database/sql remains explicit in this file's imports
// when checking query errors against sql.ErrNoRows in future moderation work.
var _ = sql.ErrNoRows
