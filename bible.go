package main

import (
	"html/template"
	"net/http"
	"strconv"
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
	var chapterVerses []BibleVerse
	var reference string
	var book string
	var chapter string
	var bibleError string

	reference = strings.TrimSpace(r.URL.Query().Get("ref"))
	book = strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("book")))
	chapter = strings.TrimSpace(r.URL.Query().Get("chapter"))

	// Individual verse search.
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

	// Full chapter reader.
	if book != "" && chapter != "" {
		chapterNumber, err := strconv.Atoi(chapter)

		if err != nil || chapterNumber < 1 {
			bibleError = "Please select a valid Bible chapter."
		} else {
			pattern := book + "." + strconv.Itoa(chapterNumber) + ".%"

			rows, err := db.Query(`
				SELECT verse_key, idu, en
				FROM translations
				WHERE UPPER(verse_key) LIKE UPPER($1)
				ORDER BY id ASC
			`, pattern)

			if err != nil {
				bibleError = "Unable to load this Bible chapter."
			} else {
				defer rows.Close()

				for rows.Next() {
					var v BibleVerse

					if err := rows.Scan(
						&v.VerseKey,
						&v.Idoma,
						&v.English,
					); err != nil {
						bibleError = "Unable to read this Bible chapter."
						chapterVerses = nil
						break
					}

					chapterVerses = append(chapterVerses, v)
				}

				if err := rows.Err(); err != nil {
					bibleError = "Unable to read this Bible chapter."
					chapterVerses = nil
				}

				if len(chapterVerses) == 0 && bibleError == "" {
					bibleError = "No verses were found for this chapter."
				}
			}
		}
	}

	data := struct {
		Reference     string
		Book          string
		Chapter       string
		Verse         BibleVerse
		ChapterVerses []BibleVerse
		Error         string
	}{
		Reference:     reference,
		Book:          book,
		Chapter:       chapter,
		Verse:         verse,
		ChapterVerses: chapterVerses,
		Error:         bibleError,
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
