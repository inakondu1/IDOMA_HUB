package main

import (
	"strings"
	"testing"
)

func TestIdomaCultureQuestionsReceiveCulturalReference(t *testing.T) {
	questions := []string{
		"Tell me about Idoma culture and traditions",
		"What is Alekwu?",
		"Tell me about Idoma festivals",
	}

	for _, question := range questions {
		t.Run(question, func(t *testing.T) {
			got := idomaReferenceContext(question)

			if !strings.Contains(got, "CURATED IDOMA CULTURAL REFERENCE NOTES") {
				t.Errorf("Expected cultural reference notes for question %q", question)
			}
		})
	}
}

func TestUnrelatedQuestionsDoNotReceiveCulturalReference(t *testing.T) {
	got := idomaCultureReferenceContext("How do I change my password?")

	if got != "" {
		t.Errorf("Expected no cultural reference for an unrelated question; got %q", got)
	}
}
