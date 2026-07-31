// Package ai provides an optional, fully local AI layer. It is inert unless the
// user passes --ai. When Ollama is unavailable it degrades gracefully to a
// deterministic, offline, extractive summary so the tool is always useful and
// always free.
package ai

import (
	"context"

	"github.com/opsgraph/opsgraph/internal/config"
	"github.com/opsgraph/opsgraph/internal/model"
)

// Summarize returns a short natural-language incident summary. It tries a local
// Ollama model first (see ollama.go); on any error or when disabled it returns a
// deterministic offline summary. It never fails and never needs the network.
func Summarize(ctx context.Context, cfg *config.Config, res model.AskResult) string {
	if s, ok := ollamaSummary(ctx, cfg, res); ok {
		return s
	}
	return LocalSummary(res)
}
