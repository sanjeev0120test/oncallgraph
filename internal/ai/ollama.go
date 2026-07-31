package ai

import (
	"context"

	"github.com/opsgraph/opsgraph/internal/config"
	"github.com/opsgraph/opsgraph/internal/model"
)

// ollamaSummary attempts a local LLM summary via Ollama. Returns ok=false to
// signal the caller to fall back to LocalSummary. The full Ollama/RAG path is
// wired in the AI layer; this keeps the summary offline-safe by default.
func ollamaSummary(_ context.Context, cfg *config.Config, _ model.AskResult) (string, bool) {
	if cfg == nil || !cfg.AI.Enabled {
		return "", false
	}
	return "", false
}
