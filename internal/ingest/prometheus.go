package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
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
	if !strings.EqualFold(strings.TrimSpace(resp.Status), "success") {
		return fmt.Errorf("prometheus api status %q (want success)", resp.Status)
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
	// Mark a successful scrape so live-prefer can trust empty quiet results.
	if err := s.SetMeta("connector:prometheus", "ok"); err != nil {
		return err
	}
	return nil
}

type promAlertsResponse struct {
	Status string `json:"status"`
	Data   struct {
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
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "opsgraph/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
		return nil, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	const maxBody = 1 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxBody {
		return nil, fmt.Errorf("GET %s: response exceeds %d bytes", url, maxBody)
	}
	return body, nil
}

func upsertRemoteAlert(s *store.Store, labels, annotations map[string]string, state string, at time.Time, source string, fallbackNow time.Time) (id string, ok bool, err error) {
	name := labels["alertname"]
	if name == "" {
		return "", false, nil
	}
	rawSvc := resolveServiceLabel(labels)
	if rawSvc == "" {
		return "", false, nil
	}
	// Map label aliases (checkout-api) onto canonical service ids when known.
	svcID := rawSvc
	if svc, err := s.GetServiceByNameOrAlias(rawSvc); err == nil {
		svcID = svc.ID
	} else if errors.Is(err, store.ErrAmbiguous) {
		fmt.Fprintf(os.Stderr, "warning: %s alert %q service label %q is ambiguous; skipping\n", source, name, rawSvc)
		return "", false, nil
	} else if errors.Is(err, store.ErrNotFound) {
		// Synthesize a stub so orphan alerts remain ask/resolve-able.
		if err := s.UpsertService(model.Service{
			ID: rawSvc, Name: rawSvc, Health: model.HealthUnknown, Sources: []string{source},
		}); err != nil {
			return "", false, err
		}
		svcID = rawSvc
	} else if err != nil {
		return "", false, err
	}
	var status string
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "firing", "active":
		status = "firing"
	case "resolved", "inactive":
		// Prometheus uses "inactive" for cleared alerts.
		status = "resolved"
	case "suppressed":
		// Silenced ≠ paging; keep visible but do not drive R3/score as active.
		status = "suppressed"
	case "pending", "unprocessed":
		status = "pending"
	default:
		// Unknown states must not invent paging signals.
		fmt.Fprintf(os.Stderr, "warning: %s alert %q has unknown state %q; skipping\n", source, name, state)
		return "", false, nil
	}
	if at.IsZero() {
		at = fallbackNow
		if at.IsZero() {
			at = time.Now().UTC()
		}
	}
	// Fingerprint remaining labels so multiple series (instance, pod, …) do not collapse.
	id = source + "-" + slug(name) + "-" + slug(svcID) + "-" + labelFingerprint(labels)
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
	// Prefer explicit service labels. Demote generic app/k8s name labels — they
	// often name the scrape target, not the owning on-call service.
	if svc := firstNonEmpty(
		labels["service"],
		labels["service_name"],
		labels["exported_service"],
		labels["label_service"],
	); svc != "" {
		return svc
	}
	return ""
}

func labelFingerprint(labels map[string]string) string {
	if len(labels) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		switch k {
		case "alertname",
			"service", "service_name", "exported_service", "label_service",
			"app", "app_kubernetes_io_name", "label_app_kubernetes_io_name",
			"severity":
			// Already in the id prefix (name/service) or intentionally mutable.
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(labels[k])
		b.WriteByte('|')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:4])
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
