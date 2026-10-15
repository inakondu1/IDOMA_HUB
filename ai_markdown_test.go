package main

import (
	"strings"
	"testing"
)

func TestRenderMarkdownFormatsHeadingsAndBoldText(t *testing.T) {
	got := string(renderMarkdown("### Idoma Culture\n\nIdoma **culture** matters."))

	checks := []string{
		"<h3>Idoma Culture</h3>",
		"<p>Idoma <strong>culture</strong> matters.</p>",
	}

	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Errorf("Expected output to contain %q; got: %s", check, got)
		}
	}
}

func TestRenderMarkdownFormatsLists(t *testing.T) {
	input := "- First item\n- Second item\n\n1. First step\n2. Second step\n\n• Third item"
	got := string(renderMarkdown(input))

	checks := []string{
		"<ul>",
		"<li>First item</li>",
		"<li>Second item</li>",
		"</ul>",
		"<ol>",
		"<li>First step</li>",
		"<li>Second step</li>",
		"<li>Third item</li>",
	}

	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Errorf("Expected output to contain %q; got: %s", check, got)
		}
	}
}

func TestRenderMarkdownEscapesUnsafeHTML(t *testing.T) {
	got := string(renderMarkdown(`<script>alert("unsafe")</script>`))

	if strings.Contains(got, "<script>") {
		t.Fatalf("Unsafe script tag was not escaped: %s", got)
	}

	if !strings.Contains(got, "&lt;script&gt;") {
		t.Errorf("Expected escaped HTML; got: %s", got)
	}
}

func TestRenderMarkdownEscapesCodeBlocks(t *testing.T) {
	got := string(renderMarkdown("```html\n<script>alert(1)</script>\n```"))

	if strings.Contains(got, "<script>") {
		t.Fatalf("Script tag in code block was not escaped: %s", got)
	}

	if !strings.Contains(got, "<pre><code>") ||
		!strings.Contains(got, "&lt;script&gt;") {
		t.Errorf("Expected an escaped code block; got: %s", got)
	}
}
