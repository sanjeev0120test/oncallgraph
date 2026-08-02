package ai

import (
	"strings"
	"testing"
)

func TestEvidenceDocuments(t *testing.T) {
	res := sampleResult()
	docs := evidenceDocuments(res)
	if len(docs) < 2 {
		t.Fatalf("want evidence+runbook docs, got %d", len(docs))
	}
	ids := map[string]bool{}
	for _, d := range docs {
		if d.ID == "" || d.Content == "" {
			t.Fatalf("empty doc: %+v", d)
		}
		if ids[d.ID] {
			t.Fatalf("duplicate doc id %q", d.ID)
		}
		ids[d.ID] = true
	}
	if !ids["ev-change-1"] || !ids["ev-alert-1"] {
		t.Fatalf("missing evidence docs: %v", ids)
	}
}

func TestOllamaAPIBase(t *testing.T) {
	if got := ollamaAPIBase(""); got != "http://localhost:11434/api" {
		t.Fatalf("default=%q", got)
	}
	if got := ollamaAPIBase("http://127.0.0.1:11434/"); got != "http://127.0.0.1:11434/api" {
		t.Fatalf("trim=%q", got)
	}
}

func TestBuildPromptCitesEvidence(t *testing.T) {
	p := buildPrompt(sampleResult(), []string{"related snippet"})
	for _, want := range []string{
		"Service: checkout",
		"ev-change-1",
		"CheckoutErrorRateHigh",
		"Related context:",
		"related snippet",
		"Bullets:",
	} {
		if !strings.Contains(p, want) {
			t.Fatalf("prompt missing %q:\n%s", want, p)
		}
	}
}
