package main

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

func createStatusHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
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

		mediaURL, err = uploadToCloudinary(
			photo,
			"idoma_hub/statuses",
			"image",
		)

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

		mediaURL, err = uploadToCloudinary(
			video,
			"idoma_hub/statuses",
			"video",
		)

		if err != nil {
			http.Error(w, "Unable to upload video.", http.StatusInternalServerError)
			log.Println(err)
			return
		}
	}

	if content == "" && mediaURL == "" {
		http.Error(w, "Status cannot be empty.", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(`
                INSERT INTO statuses (
                        user_id,
                        content,
                        media_url,
                        media_type,
                        expires_at
                )
                VALUES ($1, $2, $3, $4, $5)
        `,
		userID,
		content,
		mediaURL,
		mediaType,
		time.Now().Add(24*time.Hour),
	)

	if err != nil {
		http.Error(w, "Unable to create status.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
