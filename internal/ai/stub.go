package ai

import (
	"context"

	"github.com/sanjeev0120test/oncallgraph/internal/model"
)

// Summarizer produces a natural-language incident summary.
type Summarizer interface {
	Summarize(ctx context.Context, res model.AskResult) (string, error)
}

// Embedder embeds text for RAG retrieval. Reserved for future wiring.
type Embedder interface {
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
}

// StubSummarizer is a deterministic, offline Summarizer used in tests.
type StubSummarizer struct {
	Text string
	Err  error
}

// Summarize implements Summarizer.
func (s StubSummarizer) Summarize(_ context.Context, res model.AskResult) (string, error) {
	if s.Err != nil {
		return "", s.Err
	}
	if s.Text != "" {
		return s.Text, nil
	}
	return LocalSummary(res), nil
}

// StubEmbedder returns zero vectors; used to keep AI tests network-free.
type StubEmbedder struct{}

// EmbedDocuments implements Embedder.
func (StubEmbedder) EmbedDocuments(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = []float32{0}
	}
	return out, nil
}
