package ai

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/sanjeev0120test/oncallgraph/internal/config"
	"github.com/sanjeev0120test/oncallgraph/internal/model"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

// ollamaReachable probes GET {url}/api/tags with a short timeout.
func ollamaReachable(ctx context.Context, cfg *config.Config) bool {
	base := strings.TrimRight(cfg.AI.OllamaURL, "/")
	if base == "" {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/tags", nil)
	if err != nil {
		return false
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// ollamaSummary generates an incident summary with a local Ollama model, using
// RAG over the incident's evidence for grounding. It returns ok=false on any
// error so callers fall back to the deterministic offline summary.
func ollamaSummary(ctx context.Context, cfg *config.Config, res model.AskResult) (string, bool) {
	if cfg == nil || !cfg.AI.Enabled {
		return "", false
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.AITimeout())
	defer cancel()

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
	filtered, ok := filterCitedBullets(out, res)
	if !ok {
		// Model ignored citation rules; degrade rather than ship ungrounded text.
		return "", false
	}
	return filtered, true
}
