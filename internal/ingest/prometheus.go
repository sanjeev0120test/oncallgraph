package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sanjeev0120test/oncallgraph/internal/model"
	"github.com/sanjeev0120test/oncallgraph/internal/store"
)

// IngestPrometheus fetches /api/v1/alerts and upserts firing/pending alerts.
// Disabled unless explicitly enabled; tested with httptest (no live Prometheus).
func IngestPrometheus(ctx context.Context, s *store.Store, baseURL string, client *http.Client) error {
	if client == nil {
		client = http.DefaultClient
	}
	url := strings.TrimRight(baseURL, "/") + "/api/v1/alerts"
	body, err := getJSON(ctx, client, url)
	if err != nil {
		return err
	}
	var resp promAlertsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("prometheus decode: %w", err)
	}
	for _, a := range resp.Data.Alerts {
		if err := upsertRemoteAlert(s, a.labels(), a.annotations(), a.State, a.ActiveAt, "prometheus"); err != nil {
			return err
		}
	}
	return nil
}

type promAlertsResponse struct {
	Data struct {
		Alerts []promAlert `json:"alerts"`
	} `json:"data"`
}

type promAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	State       string            `json:"state"`
	ActiveAt    time.Time         `json:"activeAt"`
}

func (a promAlert) labels() map[string]string      { return a.Labels }
func (a promAlert) annotations() map[string]string { return a.Annotations }

func getJSON(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	return body, nil
}

func upsertRemoteAlert(s *store.Store, labels, annotations map[string]string, state string, at time.Time, source string) error {
	name := labels["alertname"]
	if name == "" {
		return nil
	}
	svcID := firstNonEmpty(labels["service"], labels["app"], labels["job"])
	if svcID == "" {
		return nil
	}
	status := "firing"
	switch strings.ToLower(state) {
	case "firing", "active":
		status = "firing"
	case "resolved", "suppressed":
		status = "resolved"
	case "pending":
		status = "pending"
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	id := source + "-" + slug(name) + "-" + slug(svcID)
	evID := "ev-" + id
	summary := firstNonEmpty(annotations["summary"], annotations["description"], name)
	if err := s.UpsertAlert(model.Alert{
		ID: id, ServiceID: svcID, At: at.UTC(), Severity: firstNonEmpty(labels["severity"], "warning"),
		Name: name, Status: status, Summary: summary, Source: source, EvidenceID: evID,
	}); err != nil {
		return err
	}
	return s.UpsertEvidence(model.Evidence{
		ID: evID, Source: source, At: at.UTC(), Kind: "alert", Summary: summary, RawRef: name, ServiceID: svcID,
	})
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func slug(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
