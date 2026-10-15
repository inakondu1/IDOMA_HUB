package main

import (
	stdhtml "html"
	"html/template"
	"regexp"
	"strings"
)

var (
	markdownHeadingPattern   = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*#*\s*$`)
	markdownOrderedPattern   = regexp.MustCompile(`^\d+[.)]\s+(.+)$`)
	markdownBoldPattern      = regexp.MustCompile(`\*\*(.+?)\*\*`)
	markdownBoldAltPattern   = regexp.MustCompile(`__(.+?)__`)
	markdownItalicPattern    = regexp.MustCompile(`\*([^*]+)\*`)
	markdownItalicAltPattern = regexp.MustCompile(`_([^_]+)_`)
)

func renderMarkdown(markdown string) template.HTML {
	markdown = strings.ReplaceAll(markdown, "\r\n", "\n")
	lines := strings.Split(markdown, "\n")

	var output strings.Builder
	var paragraph []string
	listType := ""
	inCodeBlock := false

	flushParagraph := func() {
		if len(paragraph) == 0 {
			return
		}

		output.WriteString("<p>")
		output.WriteString(renderMarkdownInline(strings.Join(paragraph, " ")))
		output.WriteString("</p>\n")
		paragraph = nil
	}

	closeList := func() {
		if listType != "" {
			output.WriteString("</")
			output.WriteString(listType)
			output.WriteString(">\n")
			listType = ""
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			flushParagraph()
			closeList()

			if inCodeBlock {
				output.WriteString("</code></pre>\n")
				inCodeBlock = false
			} else {
				output.WriteString("<pre><code>")
				inCodeBlock = true
			}
			continue
		}

		if inCodeBlock {
			output.WriteString(stdhtml.EscapeString(line))
			output.WriteByte('\n')
			continue
		}

		if trimmed == "" {
			flushParagraph()
			closeList()
			continue
		}

		if heading := markdownHeadingPattern.FindStringSubmatch(trimmed); heading != nil {
			flushParagraph()
			closeList()

			level := len(heading[1])
			output.WriteString("<h")
			output.WriteString(headingNumber(level))
			output.WriteString(">")
			output.WriteString(renderMarkdownInline(heading[2]))
			output.WriteString("</h")
			output.WriteString(headingNumber(level))
			output.WriteString(">\n")
			continue
		}

		listItem := ""
		switch {
		case strings.HasPrefix(trimmed, "- "),
			strings.HasPrefix(trimmed, "* "),
			strings.HasPrefix(trimmed, "+ "):
			listItem = strings.TrimSpace(trimmed[2:])
		case strings.HasPrefix(trimmed, "• "):
			listItem = strings.TrimSpace(strings.TrimPrefix(trimmed, "• "))
		}

		if listItem != "" {
			flushParagraph()

			if listType != "ul" {
				closeList()
				output.WriteString("<ul>\n")
				listType = "ul"
			}

			output.WriteString("<li>")
			output.WriteString(renderMarkdownInline(listItem))
			output.WriteString("</li>\n")
			continue
		}
		if ordered := markdownOrderedPattern.FindStringSubmatch(trimmed); ordered != nil {
			flushParagraph()

			if listType != "ol" {
				closeList()
				output.WriteString("<ol>\n")
				listType = "ol"
			}

			output.WriteString("<li>")
			output.WriteString(renderMarkdownInline(ordered[1]))
			output.WriteString("</li>\n")
			continue
		}

		closeList()
		paragraph = append(paragraph, trimmed)
	}

	flushParagraph()
	closeList()

	if inCodeBlock {
		output.WriteString("</code></pre>\n")
	}

	return template.HTML(output.String())
}

func headingNumber(level int) string {
	return []string{"", "1", "2", "3", "4", "5", "6"}[level]
}

func renderMarkdownInline(text string) string {
	// Escape AI-provided text before adding any formatting tags.
	text = stdhtml.EscapeString(text)

	text = markdownBoldPattern.ReplaceAllString(text, "<strong>$1</strong>")
	text = markdownBoldAltPattern.ReplaceAllString(text, "<strong>$1</strong>")
	text = markdownItalicPattern.ReplaceAllString(text, "<em>$1</em>")
	text = markdownItalicAltPattern.ReplaceAllString(text, "<em>$1</em>")

	return text
}
