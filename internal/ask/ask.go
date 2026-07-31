// Package ask assembles the deterministic incident-context answer for a service.
package ask

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/opsgraph/opsgraph/internal/model"
	"github.com/opsgraph/opsgraph/internal/runbook"
	"github.com/opsgraph/opsgraph/internal/store"
)

// Options controls how an answer is assembled.
type Options struct {
	Since       time.Duration
	Now         time.Time
	WithRunbook bool
}

// ErrServiceNotFound is returned when the query matches no service.
var ErrServiceNotFound = errors.New("service not found")

// Ask builds the full AskResult for a service query. It is fully deterministic:
// given the same store contents and Now, the output is byte-stable.
func Ask(s *store.Store, query string, opts Options) (model.AskResult, error) {
	now := opts.Now.UTC()
	window := now.Add(-opts.Since)

	svc, err := s.GetServiceByNameOrAlias(query)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return model.AskResult{}, fmt.Errorf("%w: %q", ErrServiceNotFound, query)
		}
		return model.AskResult{}, err
	}

	res := model.AskResult{
		Service:         *svc,
		Window:          humanDuration(opts.Since),
		GeneratedAt:     now,
		Changes:         []model.Change{},
		Alerts:          []model.Alert{},
		Upstream:        []model.Service{},
		Downstream:      []model.Service{},
		Timeline:        []model.TimelineEvent{},
		Recommendations: []string{},
		Evidence:        []model.Evidence{},
	}

	if svc.OwnerID != "" {
		if owner, err := s.GetOwner(svc.OwnerID); err == nil {
			res.Owner = owner
		} else if !errors.Is(err, store.ErrNotFound) {
			return model.AskResult{}, err
		}
	}

	if res.Changes, err = s.ListChanges(svc.ID, window); err != nil {
		return model.AskResult{}, err
	}
	if res.Changes == nil {
		res.Changes = []model.Change{}
	}
	if res.Alerts, err = s.ListAlerts(svc.ID, window); err != nil {
		return model.AskResult{}, err
	}
	if res.Alerts == nil {
		res.Alerts = []model.Alert{}
	}

	up, down, err := blastRadius(s, svc.ID)
	if err != nil {
		return model.AskResult{}, err
	}
	res.Upstream, res.Downstream = up, down

	if opts.WithRunbook {
		vr, err := runbook.NewVerifier(s, now).VerifyService(svc.ID)
		if err != nil {
			return model.AskResult{}, err
		}
		if vr.Status != model.StatusMissing {
			res.RunbookResult = &vr
		}
	}

	res.Timeline = buildTimeline(res.Changes, res.Alerts)
	res.Recommendations = recommend(res)

	if res.Evidence, err = collectEvidence(s, res); err != nil {
		return model.AskResult{}, err
	}
	return res, nil
}

// blastRadius returns upstream (services this one depends on) and downstream
// (services that depend on this one), each sorted by id with missing targets
// synthesized as unknown-health.
func blastRadius(s *store.Store, serviceID string) (up, down []model.Service, err error) {
	deps, err := s.ListDependencies(serviceID)
	if err != nil {
		return nil, nil, err
	}
	var upIDs, downIDs []string
	for _, d := range deps {
		if d.FromServiceID == serviceID {
			upIDs = append(upIDs, d.ToServiceID)
		}
		if d.ToServiceID == serviceID {
			downIDs = append(downIDs, d.FromServiceID)
		}
	}
	up, err = resolveServices(s, upIDs)
	if err != nil {
		return nil, nil, err
	}
	down, err = resolveServices(s, downIDs)
	if err != nil {
		return nil, nil, err
	}
	return up, down, nil
}

func resolveServices(s *store.Store, ids []string) ([]model.Service, error) {
	seen := map[string]bool{}
	out := []model.Service{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		svc, err := s.GetService(id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				out = append(out, model.Service{ID: id, Name: id, Health: model.HealthUnknown, Sources: []string{"dependency"}})
				continue
			}
			return nil, err
		}
		out = append(out, *svc)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func buildTimeline(changes []model.Change, alerts []model.Alert) []model.TimelineEvent {
	events := make([]model.TimelineEvent, 0, len(changes)+len(alerts))
	for _, c := range changes {
		events = append(events, model.TimelineEvent{
			At: c.At, Kind: "change", Summary: c.Summary, ServiceID: c.ServiceID, EvidenceID: c.EvidenceID,
		})
	}
	for _, a := range alerts {
		events = append(events, model.TimelineEvent{
			At: a.At, Kind: "alert", Summary: a.Name + " (" + a.Status + ")", ServiceID: a.ServiceID,
			EvidenceID: a.EvidenceID, Severity: a.Severity,
		})
	}
	sort.SliceStable(events, func(i, j int) bool {
		if !events[i].At.Equal(events[j].At) {
			return events[i].At.Before(events[j].At)
		}
		if events[i].Kind != events[j].Kind {
			return events[i].Kind < events[j].Kind
		}
		return events[i].EvidenceID < events[j].EvidenceID
	})
	return events
}

func collectEvidence(s *store.Store, res model.AskResult) ([]model.Evidence, error) {
	seen := map[string]bool{}
	var ids []string
	add := func(id string) {
		if id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, c := range res.Changes {
		add(c.EvidenceID)
	}
	for _, a := range res.Alerts {
		add(a.EvidenceID)
	}
	if res.RunbookResult != nil {
		for _, st := range res.RunbookResult.Steps {
			add(st.EvidenceID)
		}
	}
	ev, err := s.ListEvidence(ids)
	if err != nil {
		return nil, err
	}
	if ev == nil {
		ev = []model.Evidence{}
	}
	return ev, nil
}

func humanDuration(d time.Duration) string {
	// Prefer minutes for readability up to 2h (so 60m stays "60m").
	if d%time.Minute == 0 && d < 2*time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d%time.Hour == 0 {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return d.String()
}
