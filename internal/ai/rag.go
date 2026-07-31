package ai

import (
	"context"
	"strings"

	"github.com/opsgraph/opsgraph/internal/config"
	"github.com/opsgraph/opsgraph/internal/model"
	chromem "github.com/philippgille/chromem-go"
)

// retrieveContext builds an in-memory vector index of the incident's evidence
// (embedded via a local Ollama model) and returns the most relevant snippets
// for the incident question. It is best-effort: any error yields nil so the
// caller degrades gracefully.
func retrieveContext(ctx context.Context, cfg *config.Config, res model.AskResult) []string {
	docs := evidenceDocuments(res)
	if len(docs) < 2 {
		return nil // nothing meaningful to retrieve
	}

	embed := chromem.NewEmbeddingFuncOllama(cfg.AI.EmbedModel, ollamaAPIBase(cfg.AI.OllamaURL))
	db := chromem.NewDB()
	col, err := db.CreateCollection("incident", nil, embed)
	if err != nil {
		return nil
	}
	if err := col.AddDocuments(ctx, docs, 1); err != nil {
		return nil
	}

	n := 3
	if n > len(docs) {
		n = len(docs)
	}
	query := "root cause and blast radius for " + res.Service.ID + " incident"
	results, err := col.Query(ctx, query, n, nil, nil)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(results))
	for _, r := range results {
		out = append(out, r.Content)
	}
	return out
}

func evidenceDocuments(res model.AskResult) []chromem.Document {
	var docs []chromem.Document
	for _, e := range res.Evidence {
		content := e.Kind + ": " + e.Summary
		docs = append(docs, chromem.Document{ID: e.ID, Content: content})
	}
	return docs
}

func ollamaAPIBase(url string) string {
	url = strings.TrimRight(url, "/")
	if url == "" {
		return "http://localhost:11434/api"
	}
	return url + "/api"
}
