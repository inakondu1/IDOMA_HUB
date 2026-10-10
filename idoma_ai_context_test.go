package main

import (
	"strings"
	"testing"
)

func TestIdomaReferenceContext(t *testing.T) {
	tests := []struct {
		name     string
		question string
		want     []string
		dontWant []string
	}{
		{
			name:     "homograph headword",
			question: "What does Ebe mean?",
			want: []string{
				"SUPPLIED IDOMA HOMOGRAPH REFERENCE",
				"Ebe:",
				"Ɛ̀bɛ́",
				"Meat",
				"Ɛ̄bɛ̄",
				"Slap",
			},
		},
		{
			name:     "homograph topic",
			question: "Explain Idoma homographs",
			want: []string{
				"SUPPLIED IDOMA HOMOGRAPH REFERENCE",
			},
		},
		{
			name:     "dialect category",
			question: "How do dialects differ for salt?",
			want: []string{
				"SUPPLIED IDOMA DIALECT REFERENCE",
				"Salt",
				"Otukpo:",
				"Orokam:",
			},
			dontWant: []string{
				"- Food |",
			},
		},
		{
			name:     "speech work",
			question: "Show me the speech work sentences",
			want: []string{
				"SUPPLIED IDOMA SPEECH-WORK EXAMPLES",
				"Sentence 1:",
				"Sentence 9:",
				"Agbo is clapping for them.",
			},
		},
		{
			name:     "unrelated question",
			question: "Tell me about the weather",
			dontWant: []string{
				"SUPPLIED IDOMA HOMOGRAPH REFERENCE",
				"SUPPLIED IDOMA DIALECT REFERENCE",
				"SUPPLIED IDOMA SPEECH-WORK EXAMPLES",
			},
		},
		{
			name:     "empty question",
			question: "   ",
			dontWant: []string{
				"SUPPLIED IDOMA HOMOGRAPH REFERENCE",
				"SUPPLIED IDOMA DIALECT REFERENCE",
				"SUPPLIED IDOMA SPEECH-WORK EXAMPLES",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := idomaReferenceContext(tt.question)

			for _, expected := range tt.want {
				if !strings.Contains(got, expected) {
					t.Errorf("context for %q should contain %q; got:\n%s",
						tt.question, expected, got)
				}
			}

			for _, unexpected := range tt.dontWant {
				if strings.Contains(got, unexpected) {
					t.Errorf("context for %q should not contain %q; got:\n%s",
						tt.question, unexpected, got)
				}
			}
		})
	}
}
