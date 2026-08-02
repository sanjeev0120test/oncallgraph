package ai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/config"
	"github.com/sanjeev0120test/opsgraph/internal/model"
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
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8<<10))
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// ollamaSummary generates an incident summary with a local Ollama model, using
// RAG over the incident's evidence for grounding. It returns ok=false on any
// error so callers fall back to the deterministic offline summary.
func ollamaSummary(ctx context.Context, cfg *config.Config, res model.AskResult) (string, bool) {
	if cfg == nil || !cfg.AI.Enabled {
		return "", false
	}

	budget := cfg.AITimeout()
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()

	// Cap RAG so embedding cannot starve generation under a shared deadline.
	ragTimeout := budget / 3
	if ragTimeout < 2*time.Second {
		ragTimeout = budget / 2
	}
	ragCtx, ragCancel := context.WithTimeout(ctx, ragTimeout)
	ragContext := retrieveContext(ragCtx, cfg, res)
	ragCancel()

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
