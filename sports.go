package main

import (
	"html/template"
	"log"
	"net/http"
)

func sportsHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)

	var username string
	var profilePicture string

	if loggedIn {
		updateLastActive(r)

		err := db.QueryRow(
			"SELECT username, COALESCE(profile_picture, '') FROM users WHERE id = $1",
			userID,
		).Scan(&username, &profilePicture)

		if err != nil {
			http.Error(w, "Unable to load your account.", http.StatusInternalServerError)
			log.Println(err)
			return
		}
	}

	rows, err := db.Query(`
		SELECT posts.id,
		       users.username,
		       COALESCE(users.profile_picture, ''),
		       posts.content,
		       posts.created_at,
		       posts.media_url,
		       posts.media_type,
		       COUNT(post_likes.id) AS like_count,
		       (SELECT COUNT(*) FROM comments WHERE comments.post_id = posts.id) AS comment_count,
		       (SELECT COUNT(*) FROM posts shares WHERE shares.original_post_id = posts.id) AS share_count,
		       CASE
		           WHEN $1 > 0 THEN EXISTS (
		               SELECT 1
		               FROM post_likes user_like
		               WHERE user_like.post_id = posts.id
		               AND user_like.user_id = $1
		           )
		           ELSE FALSE
		       END AS liked_by_me
		FROM posts
		JOIN users ON users.id = posts.user_id
		LEFT JOIN post_likes ON post_likes.post_id = posts.id
		WHERE posts.category = 'sports'
		GROUP BY posts.id,
		         users.username,
		         users.profile_picture,
		         posts.content,
		         posts.created_at,
		         posts.media_url,
		         posts.media_type
		ORDER BY posts.id DESC
	`, userID)

	if err != nil {
		http.Error(w, "Unable to load sports posts.", http.StatusInternalServerError)
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
			http.Error(w, "Unable to read sports posts.", http.StatusInternalServerError)
			log.Println(err)
			return
		}

		commentRows, err := db.Query(`
			SELECT comments.id,
			       comments.post_id,
			       users.username,
			       COALESCE(users.profile_picture, ''),
			       comments.content,
			       comments.created_at
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
		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Unable to read sports posts.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	data := struct {
		LoggedIn      bool
		Username      string
		ProfilePicture string
		Posts         []Post
	}{
		LoggedIn:       loggedIn,
		Username:       username,
		ProfilePicture: profilePicture,
		Posts:          posts,
	}

	tmpl, err := template.ParseFiles("templates/sports.html")
	if err != nil {
		http.Error(w, "Unable to load Sports page.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Println(err)
	}
}
