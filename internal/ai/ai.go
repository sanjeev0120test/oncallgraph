// Package ai provides an optional, fully local AI layer. It is inert unless the
// user passes --ai. When Ollama is unavailable it degrades gracefully to a
// deterministic, offline, extractive summary so the tool is always useful and
// always free.
package ai

import (
	"context"

	"github.com/sanjeev0120test/opsgraph/internal/config"
	"github.com/sanjeev0120test/opsgraph/internal/model"
)

// UnavailableMessage is returned (as the AI summary prefix) when Ollama is down.
const UnavailableMessage = "AI unavailable, deterministic result only"

// Summarize returns a short natural-language incident summary. It tries a local
// Ollama model first (see ollama.go); on any error or when disabled it returns a
// deterministic offline summary. It never fails and never needs the network
// unless AI is explicitly enabled.
func Summarize(ctx context.Context, cfg *config.Config, res model.AskResult) string {
	if cfg == nil || !cfg.AI.Enabled {
		return LocalSummary(res)
	}
	if !ollamaReachable(ctx, cfg) {
		return UnavailableMessage + "\n" + LocalSummary(res)
	}
	if s, ok := ollamaSummary(ctx, cfg, res); ok {
		return s
	}
	return UnavailableMessage + "\n" + LocalSummary(res)
}

// SummarizeWith uses an injected Summarizer (tests / future backends).
func SummarizeWith(ctx context.Context, s Summarizer, res model.AskResult) string {
	out, err := s.Summarize(ctx, res)
	if err != nil || out == "" {
		return LocalSummary(res)
	}
	return out
}
