package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/store"
)

// IngestPrometheus fetches /api/v1/alerts and upserts firing/pending alerts.
// Alerts previously seen from this source but absent in the scrape are marked
// resolved so the store does not keep zombie firings. Disabled unless
// explicitly enabled; tested with httptest (no live Prometheus).
// now is used when an alert lacks StartsAt/activeAt (zero means wall clock).
func IngestPrometheus(ctx context.Context, s *store.Store, baseURL string, client *http.Client, now time.Time) error {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	if now.IsZero() {
		now = time.Now().UTC()
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
	seen := map[string]bool{}
	dropped := 0
	for _, a := range resp.Data.Alerts {
		id, ok, err := upsertRemoteAlert(s, a.labels(), a.annotations(), a.State, a.ActiveAt, "prometheus", now)
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
		fmt.Fprintf(os.Stderr, "warning: dropped %d prometheus alert(s) without a resolvable service label\n", dropped)
	}
	// Always resolve against successfully upserted IDs so unlabeled drops cannot
	// freeze zombie cleanup, and a clean empty scrape still clears firings.
	if _, err := s.ResolveActiveAlertsNotIn("prometheus", seen); err != nil {
		return err
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

func upsertRemoteAlert(s *store.Store, labels, annotations map[string]string, state string, at time.Time, source string, fallbackNow time.Time) (id string, ok bool, err error) {
	name := labels["alertname"]
	if name == "" {
		return "", false, nil
	}
	svcID := resolveServiceLabel(labels)
	if svcID == "" {
		return "", false, nil
	}
	status := "firing"
	switch strings.ToLower(state) {
	case "firing", "active":
		status = "firing"
	case "resolved", "inactive":
		// Prometheus uses "inactive" for cleared alerts.
		status = "resolved"
	case "suppressed":
		// Silenced in Alertmanager ≠ cleared; keep paging signal visible.
		status = "firing"
	case "pending", "unprocessed":
		status = "pending"
	}
	if at.IsZero() {
		at = fallbackNow
		if at.IsZero() {
			at = time.Now().UTC()
		}
	}
	id = source + "-" + slug(name) + "-" + slug(svcID)
	evID := "ev-" + id
	summary := firstNonEmpty(annotations["summary"], annotations["description"], name)
	if err := s.UpsertAlert(model.Alert{
		ID: id, ServiceID: svcID, At: at.UTC(), Severity: firstNonEmpty(labels["severity"], "warning"),
		Name: name, Status: status, Summary: summary, Source: source, EvidenceID: evID,
	}); err != nil {
		return "", false, err
	}
	if err := s.UpsertEvidence(model.Evidence{
		ID: evID, Source: source, At: at.UTC(), Kind: "alert", Summary: summary, RawRef: name, ServiceID: svcID,
	}); err != nil {
		return "", false, err
	}
	return id, true, nil
}

func resolveServiceLabel(labels map[string]string) string {
	// Intentionally omit job — it is usually a scrape target name, not a service.
	return firstNonEmpty(
		labels["service"],
		labels["service_name"],
		labels["exported_service"],
		labels["label_service"],
		labels["app"],
		labels["app_kubernetes_io_name"],
		labels["label_app_kubernetes_io_name"],
	)
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
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "unknown"
	}
	return out
}
