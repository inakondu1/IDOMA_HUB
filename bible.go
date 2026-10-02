package main

import (
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
