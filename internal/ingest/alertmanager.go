package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/opsgraph/opsgraph/internal/store"
)

// IngestAlertmanager fetches /api/v2/alerts and upserts alerts.
// Disabled unless explicitly enabled; tested with httptest (no live Alertmanager).
func IngestAlertmanager(ctx context.Context, s *store.Store, baseURL string, client *http.Client) error {
	if client == nil {
		client = http.DefaultClient
	}
	url := strings.TrimRight(baseURL, "/") + "/api/v2/alerts"
	body, err := getJSON(ctx, client, url)
	if err != nil {
		return err
	}
	var alerts []amAlert
	if err := json.Unmarshal(body, &alerts); err != nil {
		return fmt.Errorf("alertmanager decode: %w", err)
	}
	for _, a := range alerts {
		state := a.Status.State
		if state == "" {
			state = "active"
		}
		if err := upsertRemoteAlert(s, a.Labels, a.Annotations, state, a.StartsAt, "alertmanager"); err != nil {
			return err
		}
	}
	return nil
}

type amAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt"`
	Status      struct {
		State string `json:"state"`
	} `json:"status"`
}
