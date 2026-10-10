package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type CreatorApplication struct {
	ID         int64
	UserID     int
	Username   string
	Email      string
	Status     string
	Note       string
	ReviewNote string
	CreatedAt  string
}

type CreatorStudioData struct {
	Title  string
	Status string
	Note   string
}

type AdminMonetizationData struct {
	Title        string
	Applications []CreatorApplication
	Role         string
	Message      string
}

func establishConfiguredOwner() {
	email := strings.TrimSpace(os.Getenv("IDOMA_HUB_OWNER_EMAIL"))
	if email == "" {
		return
	}

	_, err := db.Exec(`
		INSERT INTO admin_users (user_id, role)
		SELECT id, 'owner'
		FROM users
		WHERE LOWER(email) = LOWER($1)
		ON CONFLICT (user_id) DO UPDATE SET role = 'owner'
	`, email)
	if err != nil {
		log.Println("Unable to establish configured IDOMA HUB owner:", err)
	}
}

func getAdminRole(userID int) string {
	establishConfiguredOwner()

	var role string
	err := db.QueryRow(
		"SELECT role FROM admin_users WHERE user_id = $1",
		userID,
	).Scan(&role)
	if err != nil {
		return ""
	}
	return role
}

func creatorStudioHandler(w http.ResponseWriter, r *http.Request) {
	userID, loggedIn := getUserIDFromSession(r)
	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		if !validSameOriginRequest(r) {
			http.Error(w, "Invalid request origin.", http.StatusForbidden)
			return
		}
		if r.FormValue("action") != "apply" {
			http.Error(w, "Invalid application action.", http.StatusBadRequest)
			return
		}

		note := strings.TrimSpace(r.FormValue("application_note"))
		if len(note) > 1500 {
			http.Error(w, "Your application note must be 1,500 characters or fewer.", http.StatusBadRequest)
			return
		}

		result, err := db.Exec(`
			INSERT INTO creator_monetization_applications
				(user_id, application_note)
			VALUES ($1, $2)
			ON CONFLICT (user_id) DO UPDATE
			SET status = 'pending',
			    application_note = EXCLUDED.application_note,
			    reviewed_by = NULL,
			    reviewed_at = NULL,
			    review_note = '',
			    updated_at = CURRENT_TIMESTAMP
			WHERE creator_monetization_applications.status = 'rejected'
		`, userID, note)
		if err != nil {
			log.Println("Unable to submit creator monetization application:", err)
			http.Error(w, "Unable to submit your application.", http.StatusInternalServerError)
			return
		}

		changed, err := result.RowsAffected()
		if err != nil {
			http.Error(w, "Unable to confirm your application.", http.StatusInternalServerError)
			return
		}
		if changed == 0 {
			http.Error(w, "You already have an application. Check its current status below.", http.StatusConflict)
			return
		}

		http.Redirect(w, r, "/creator-studio?submitted=1", http.StatusSeeOther)
		return
	}

	var status, note string
	err := db.QueryRow(`
		SELECT status, application_note
		FROM creator_monetization_applications
		WHERE user_id = $1
	`, userID).Scan(&status, &note)

	if err != nil && err != sql.ErrNoRows {
		log.Println("Unable to load creator application:", err)
		http.Error(w, "Unable to load Creator Studio.", http.StatusInternalServerError)
		return
	}
	if err != nil {
		status = "not_applied"
	}

	tmpl, err := template.ParseFiles("templates/creator_studio.html")
	if err != nil {
		log.Println("Unable to load Creator Studio template:", err)
		http.Error(w, "Unable to load Creator Studio.", http.StatusInternalServerError)
		return
	}

	message := ""
	if r.URL.Query().Get("submitted") == "1" {
		message = "Your application has been submitted."
	}

	data := struct {
		Title   string
		Status  string
		Note    string
		Message string
	}{
		Title:   "Creator Studio - IDOMA HUB",
		Status:  status,
		Note:    note,
		Message: message,
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Println("Unable to display Creator Studio:", err)
	}
}

func adminMonetizationHandler(w http.ResponseWriter, r *http.Request) {
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

	message := ""

	if r.Method == http.MethodPost {
		if !validSameOriginRequest(r) {
			http.Error(w, "Invalid request origin.", http.StatusForbidden)
			return
		}
		action := r.FormValue("action")

		switch action {
		case "review":
			id, err := strconv.ParseInt(r.FormValue("application_id"), 10, 64)
			if err != nil || id <= 0 {
				http.Error(w, "Invalid application ID.", http.StatusBadRequest)
				return
			}

			decision := strings.TrimSpace(r.FormValue("decision"))
			if decision != "approved" && decision != "rejected" && decision != "suspended" {
				http.Error(w, "Invalid review decision.", http.StatusBadRequest)
				return
			}

			reviewNote := strings.TrimSpace(r.FormValue("review_note"))
			if len(reviewNote) > 1500 {
				http.Error(w, "Review notes must be 1,500 characters or fewer.", http.StatusBadRequest)
				return
			}

			tx, err := db.Begin()
			if err != nil {
				http.Error(w, "Unable to start review.", http.StatusInternalServerError)
				return
			}
			defer tx.Rollback()

			var targetUserID int
			err = tx.QueryRow(`
				UPDATE creator_monetization_applications
				SET status = $1,
				    reviewed_by = $2,
				    reviewed_at = CURRENT_TIMESTAMP,
				    review_note = $3,
				    updated_at = CURRENT_TIMESTAMP
				WHERE id = $4
				RETURNING user_id
			`, decision, userID, reviewNote, id).Scan(&targetUserID)
			if err != nil {
				http.Error(w, "Application not found or unable to update it.", http.StatusBadRequest)
				return
			}

			if targetUserID == userID {
				http.Error(w, "You cannot review your own monetization application.", http.StatusForbidden)
				return
			}

			_, err = tx.Exec(`
				INSERT INTO admin_audit_log (actor_id, target_user_id, action, details)
				VALUES ($1, $2, $3, $4)
			`, userID, targetUserID, "monetization_"+decision, reviewNote)
			if err != nil {
				log.Println("Unable to record monetization review:", err)
				http.Error(w, "Unable to record the review.", http.StatusInternalServerError)
				return
			}

			if err := tx.Commit(); err != nil {
				http.Error(w, "Unable to complete the review.", http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/admin/monetization?updated=1", http.StatusSeeOther)
			return

		case "add_admin":
			if role != "owner" {
				http.Error(w, "Only the platform owner can authorize administrators.", http.StatusForbidden)
				return
			}

			username := strings.TrimSpace(r.FormValue("username"))
			if username == "" {
				http.Error(w, "Enter the account username.", http.StatusBadRequest)
				return
			}

			var targetID int
			err := db.QueryRow(
				"SELECT id FROM users WHERE LOWER(username) = LOWER($1)",
				username,
			).Scan(&targetID)
			if err != nil {
				http.Error(w, "No account was found with that username.", http.StatusBadRequest)
				return
			}
			if targetID == userID {
				http.Error(w, "You are already the platform owner.", http.StatusBadRequest)
				return
			}

			tx, err := db.Begin()
			if err != nil {
				http.Error(w, "Unable to begin administrator authorization.", http.StatusInternalServerError)
				return
			}

			result, err := tx.Exec(`
                                INSERT INTO admin_users (user_id, role, granted_by)
                                VALUES ($1, 'admin', $2)
                                ON CONFLICT (user_id) DO UPDATE
                                SET role = 'admin', granted_by = $2
                                WHERE admin_users.role <> 'owner'
                        `, targetID, userID)
			if err != nil {
				_ = tx.Rollback()
				log.Println("Unable to authorize administrator:", err)
				http.Error(w, "Unable to authorize this administrator.", http.StatusInternalServerError)
				return
			}

			affected, err := result.RowsAffected()
			if err != nil {
				_ = tx.Rollback()
				log.Println("Unable to verify administrator authorization:", err)
				http.Error(w, "Unable to verify administrator authorization.", http.StatusInternalServerError)
				return
			}
			if affected == 0 {
				_ = tx.Rollback()
				http.Error(w, "This account cannot be authorized as an administrator.", http.StatusBadRequest)
				return
			}

			_, err = tx.Exec(`
                                INSERT INTO admin_audit_log (actor_id, target_user_id, action, details)
                                VALUES ($1, $2, 'administrator_authorized', $3)
                        `, userID, targetID, "Administrator authorized by platform owner.")
			if err != nil {
				_ = tx.Rollback()
				log.Println("Unable to record administrator authorization:", err)
				http.Error(w, "Unable to record the authorization. No changes were saved.", http.StatusInternalServerError)
				return
			}

			if err := tx.Commit(); err != nil {
				log.Println("Unable to commit administrator authorization:", err)
				http.Error(w, "Unable to complete administrator authorization.", http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, "/admin/monetization?updated=1", http.StatusSeeOther)
			return

		case "remove_admin":
			if role != "owner" {
				http.Error(w, "Only the platform owner can remove administrators.", http.StatusForbidden)
				return
			}

			targetID, err := strconv.Atoi(r.FormValue("user_id"))
			if err != nil || targetID <= 0 || targetID == userID {
				http.Error(w, "Invalid administrator account.", http.StatusBadRequest)
				return
			}

			tx, err := db.Begin()
			if err != nil {
				http.Error(w, "Unable to begin administrator removal.", http.StatusInternalServerError)
				return
			}

			result, err := tx.Exec(
				"DELETE FROM admin_users WHERE user_id = $1 AND role = 'admin'",
				targetID,
			)
			if err != nil {
				_ = tx.Rollback()
				log.Println("Unable to remove administrator:", err)
				http.Error(w, "Unable to remove administrator.", http.StatusInternalServerError)
				return
			}

			count, err := result.RowsAffected()
			if err != nil {
				_ = tx.Rollback()
				log.Println("Unable to verify administrator removal:", err)
				http.Error(w, "Unable to verify administrator removal.", http.StatusInternalServerError)
				return
			}
			if count == 0 {
				_ = tx.Rollback()
				http.Error(w, "Administrator not found or cannot be removed.", http.StatusBadRequest)
				return
			}

			_, err = tx.Exec(`
                                INSERT INTO admin_audit_log (actor_id, target_user_id, action, details)
                                VALUES ($1, $2, 'administrator_removed', 'Administrator access removed by platform owner.')
                        `, userID, targetID)
			if err != nil {
				_ = tx.Rollback()
				log.Println("Unable to record administrator removal:", err)
				http.Error(w, "Unable to record the removal. No changes were saved.", http.StatusInternalServerError)
				return
			}

			if err := tx.Commit(); err != nil {
				log.Println("Unable to commit administrator removal:", err)
				http.Error(w, "Unable to complete administrator removal.", http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, "/admin/monetization?updated=1", http.StatusSeeOther)
			return

		default:
			http.Error(w, "Invalid administrator action.", http.StatusBadRequest)
			return
		}
	}

	rows, err := db.Query(`
		SELECT a.id, a.user_id, u.username, u.email, a.status,
		       a.application_note, a.review_note,
		       TO_CHAR(a.created_at, 'YYYY-MM-DD HH24:MI')
		FROM creator_monetization_applications a
		JOIN users u ON u.id = a.user_id
		ORDER BY
			CASE WHEN a.status = 'pending' THEN 0 ELSE 1 END,
			a.created_at DESC
	`)
	if err != nil {
		log.Println("Unable to load monetization applications:", err)
		http.Error(w, "Unable to load applications.", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var applications []CreatorApplication
	for rows.Next() {
		var a CreatorApplication
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.Username, &a.Email, &a.Status,
			&a.Note, &a.ReviewNote, &a.CreatedAt,
		); err != nil {
			http.Error(w, "Unable to read applications.", http.StatusInternalServerError)
			return
		}
		applications = append(applications, a)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Unable to finish reading applications.", http.StatusInternalServerError)
		return
	}

	if r.URL.Query().Get("updated") == "1" {
		message = "Your change has been saved."
	}

	tmpl, err := template.ParseFiles("templates/admin_monetization.html")
	if err != nil {
		log.Println("Unable to load administrator template:", err)
		http.Error(w, "Unable to load administrator page.", http.StatusInternalServerError)
		return
	}

	data := AdminMonetizationData{
		Title:        "Monetization Administration - IDOMA HUB",
		Applications: applications,
		Role:         role,
		Message:      message,
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Println("Unable to display administrator page:", err)
	}
}
