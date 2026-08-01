package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/store"
)

// IngestAlertmanager fetches /api/v2/alerts and upserts alerts.
// Alerts previously seen from this source but absent in the scrape are marked
// resolved. Disabled unless explicitly enabled; tested with httptest.
func IngestAlertmanager(ctx context.Context, s *store.Store, baseURL string, client *http.Client) error {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
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
	seen := map[string]bool{}
	dropped := 0
	for _, a := range alerts {
		state := a.Status.State
		if state == "" {
			state = "active"
		}
		id, ok, err := upsertRemoteAlert(s, a.Labels, a.Annotations, state, a.StartsAt, "alertmanager")
		if err != nil {
			return err
		}
		if ok {
			seen[id] = true
		} else {
			dropped++
		}
	}
	if dropped > 0 {
		fmt.Fprintf(os.Stderr, "warning: dropped %d alertmanager alert(s) without a resolvable service label\n", dropped)
	}
	if _, err := s.ResolveActiveAlertsNotIn("alertmanager", seen); err != nil {
		return err
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
