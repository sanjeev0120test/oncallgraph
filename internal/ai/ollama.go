package ai

import (
	"context"
	"strings"

	"github.com/opsgraph/opsgraph/internal/config"
	"github.com/opsgraph/opsgraph/internal/model"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

// ollamaSummary generates an incident summary with a local Ollama model, using
// RAG over the incident's evidence for grounding. It returns ok=false on any
// error (including "disabled" or "daemon unreachable") so callers fall back to
// the deterministic offline summary. It never touches the network unless AI is
// explicitly enabled.
func ollamaSummary(ctx context.Context, cfg *config.Config, res model.AskResult) (string, bool) {
	if cfg == nil || !cfg.AI.Enabled {
		return "", false
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.AITimeout())
	defer cancel()

	// Best-effort retrieval; nil context is fine.
	ragContext := retrieveContext(ctx, cfg, res)
	prompt := buildPrompt(res, ragContext)

	llm, err := ollama.New(
		ollama.WithModel(cfg.AI.Model),
		ollama.WithServerURL(cfg.AI.OllamaURL),
	)
	if err != nil {
		return "", false
	}
	out, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		return "", false
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return "", false
	}
	return out, true
}
