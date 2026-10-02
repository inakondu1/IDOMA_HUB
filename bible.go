package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

type BibleVerse struct {
	VerseKey string
	Idoma    string
	English  string
}

func bibleHandler(w http.ResponseWriter, r *http.Request) {
	_, loggedIn := getUserIDFromSession(r)

	if !loggedIn {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var verse BibleVerse
	var reference string
	var bibleError string

	reference = strings.TrimSpace(r.URL.Query().Get("ref"))

	if reference != "" {
		err := db.QueryRow(`
			SELECT verse_key, idu, en
			FROM translations
			WHERE UPPER(verse_key) = UPPER($1)
			LIMIT 1
		`, reference).Scan(
			&verse.VerseKey,
			&verse.Idoma,
			&verse.English,
		)

		if err != nil {
			bibleError = "Bible verse not found. Please check the reference and try again."
		}
	}

	data := struct {
		Reference string
		Verse     BibleVerse
		Error     string
	}{
		Reference: reference,
		Verse:     verse,
		Error:     bibleError,
	}

	tmpl, err := template.ParseFiles("templates/bible.html")
	if err != nil {
		http.Error(w, "Unable to load IDOMA Bible page", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Unable to display IDOMA Bible page", http.StatusInternalServerError)
		return
	}
}

func bibleDataCheckHandler(w http.ResponseWriter, r *http.Request) {
	var total int
	var firstVerse, lastVerse string

	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM translations
	`).Scan(&total)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = db.QueryRow(`
		SELECT verse_key
		FROM translations
		ORDER BY id ASC
		LIMIT 1
	`).Scan(&firstVerse)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = db.QueryRow(`
		SELECT verse_key
		FROM translations
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&lastVerse)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(
		w,
		"Total translation records: %d\nFirst verse: %s\nLast verse: %s\n",
		total,
		firstVerse,
		lastVerse,
	)
}
