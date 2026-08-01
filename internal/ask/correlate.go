package ask

import (
	"fmt"
	"sort"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
)

// correlateWindow is the max gap between a change and a later alert to count
// as a change→alert correlation. Wider than SuspectChangeWindow so an alert
// that fires ~45m after a deploy still surfaces as linked evidence.
const correlateWindow = 60 * time.Minute

// correlate returns deterministic change→alert links for active alerts that
// fire after a service change within correlateWindow. Prefer deploy/rollout
// over commit when both precede the same alert.
func correlate(res model.AskResult) []model.Correlation {
	if len(res.Changes) == 0 || len(res.Alerts) == 0 {
		return nil
	}
	var out []model.Correlation
	for _, a := range res.Alerts {
		if !model.AlertActive(a.Status) || a.At.IsZero() {
			continue
		}
		ch, ok := precedingChange(res.Changes, a.At)
		if !ok {
			continue
		}
		gap := a.At.Sub(ch.At)
		out = append(out, model.Correlation{
			Kind:           "change_then_alert",
			Summary:        fmt.Sprintf("%s %q preceded alert %s by %s", ch.Type, ch.Summary, a.Name, roundDur(gap)),
			ChangeID:       ch.ID,
			ChangeEvidence: ch.EvidenceID,
			AlertID:        a.ID,
			AlertEvidence:  a.EvidenceID,
			Gap:            roundDur(gap),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].AlertID != out[j].AlertID {
			return out[i].AlertID < out[j].AlertID
		}
		return out[i].ChangeID < out[j].ChangeID
	})
	return out
}

func precedingChange(changes []model.Change, alertAt time.Time) (model.Change, bool) {
	var best model.Change
	found := false
	bestRank := 99
	for _, c := range changes {
		if c.At.After(alertAt) || alertAt.Sub(c.At) > correlateWindow {
			continue
		}
		rank := changeRank(c.Type)
		if !found || rank < bestRank || (rank == bestRank && c.At.After(best.At)) ||
			(rank == bestRank && c.At.Equal(best.At) && c.ID < best.ID) {
			best = c
			bestRank = rank
			found = true
		}
	}
	return best, found
}

func changeRank(t string) int {
	switch t {
	case "deploy", "rollout":
		return 0
	case "commit":
		return 1
	default:
		return 2
	}
}

func roundDur(d time.Duration) string {
	if d < time.Minute {
		sec := int(d.Round(time.Second) / time.Second)
		if sec < 1 {
			sec = 1
		}
		return fmt.Sprintf("%ds", sec)
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Round(time.Minute)/time.Minute))
	}
	return fmt.Sprintf("%dh", int(d.Round(time.Hour)/time.Hour))
}
