package main

import (
	"net/http"
	"net/url"
	"strings"
)

// validSameOriginRequest rejects POST requests initiated from another origin.
// Requests without an Origin header must provide a valid Referer.
func validSameOriginRequest(r *http.Request) bool {
	source := strings.TrimSpace(r.Header.Get("Origin"))

	if source == "" {
		source = strings.TrimSpace(r.Referer())
		if source == "" {
			return false
		}
	}

	parsed, err := url.Parse(source)
	if err != nil || parsed.Host == "" {
		return false
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}

	return strings.EqualFold(parsed.Host, r.Host)
}
