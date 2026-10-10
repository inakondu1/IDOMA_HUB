package main

import (
	"fmt"
	"strings"
)

func idomaReferenceContext(question string) string {
	q := strings.ToLower(strings.TrimSpace(question))
	if q == "" {
		return ""
	}

	var sections []string

	// Match homographs by their English headwords or supplied meanings.
	homographRequested := strings.Contains(q, "homograph") ||
		strings.Contains(q, "tone mark") ||
		strings.Contains(q, "different meanings") ||
		strings.Contains(q, "meanings of")

	var matchingHomographs []IdomaHomograph

	for _, item := range idomaHomographs {
		word := strings.ToLower(strings.TrimSpace(item.Word))
		details := strings.ToLower(item.Details)

		if word != "" && (q == word ||
			strings.HasPrefix(q, word+" ") ||
			strings.Contains(q, " "+word+" ") ||
			strings.HasSuffix(q, " "+word) ||
			strings.Contains(q, " "+word+"'") ||
			strings.Contains(q, "'"+word+" ") ||
			strings.Contains(q, details) ||
			strings.Contains(details, q) && len(q) >= 4) {
			matchingHomographs = append(matchingHomographs, item)
			continue
		}

		// Also match a tone-marked form or meaning written in Details.
		if len(q) >= 4 && strings.Contains(details, q) {
			matchingHomographs = append(matchingHomographs, item)
		}
	}

	if homographRequested || len(matchingHomographs) > 0 {
		var b strings.Builder
		b.WriteString("SUPPLIED IDOMA HOMOGRAPH REFERENCE:\n")

		items := matchingHomographs
		if homographRequested && len(items) == 0 {
			limit := len(idomaHomographs)
			if limit > 12 {
				limit = 12
			}
			items = idomaHomographs[:limit]
		}

		for _, item := range items {
			fmt.Fprintf(&b, "- %s: %s\n", item.Word, item.Details)
		}

		sections = append(sections, b.String())
	}

	// Include dialect comparisons when relevant.
	communities := []string{
		"otukpo", "edumoga", "oglewu", "agatu", "ado",
		"okpoga", "otukpa", "orokam", "owukpa",
	}
	dialectCategories := []string{
		"food", "okra", "clothes", "play", "cutlass", "salt", "rat",
	}

	dialectRequested := strings.Contains(q, "dialect") ||
		strings.Contains(q, "community") ||
		strings.Contains(q, "communities") ||
		strings.Contains(q, "variation")

	for _, name := range communities {
		if strings.Contains(q, name) {
			dialectRequested = true
		}
	}
	for _, category := range dialectCategories {
		if strings.Contains(q, category) {
			dialectRequested = true
		}
	}

	if dialectRequested {
		var b strings.Builder
		b.WriteString("SUPPLIED IDOMA DIALECT REFERENCE:\n")

		categorySpecified := false
		for _, name := range dialectCategories {
			if strings.Contains(q, name) {
				categorySpecified = true
				break
			}
		}

		for _, item := range idomaDialects {
			category := strings.ToLower(item.Category)
			if categorySpecified && !strings.Contains(q, category) {
				continue
			}

			fmt.Fprintf(
				&b,
				"- %s | Otukpo: %s | Edumoga: %s | Oglewu: %s | Agatu: %s | Ado: %s | Okpoga: %s | Otukpa: %s | Orokam: %s | Owukpa: %s\n",
				item.Category,
				item.Otukpo,
				item.Edumoga,
				item.Oglewu,
				item.Agatu,
				item.Ado,
				item.Okpoga,
				item.Otukpa,
				item.Orokam,
				item.Owukpa,
			)
		}

		sections = append(sections, b.String())
	}

	// Include the supplied sentence examples when requested.
	speechRequested := strings.Contains(q, "speech work") ||
		strings.Contains(q, "sentence") ||
		strings.Contains(q, "sentences")

	if speechRequested {
		var b strings.Builder
		b.WriteString("SUPPLIED IDOMA SPEECH-WORK EXAMPLES:\n")

		for _, item := range idomaSpeechWork {
			fmt.Fprintf(
				&b,
				"- Sentence %d: %s — English: %s\n",
				item.Number,
				item.Idoma,
				item.English,
			)
		}

		sections = append(sections, b.String())
	}

	if len(sections) == 0 {
		return ""
	}

	return strings.Join(sections, "\n")
}
