package runbook_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sanjeev0120test/opsgraph/internal/model"
	"github.com/sanjeev0120test/opsgraph/internal/runbook"
	"github.com/sanjeev0120test/opsgraph/internal/store"
)

func TestKnownCheckNamesFrozen(t *testing.T) {
	got := runbook.KnownCheckNames()
	want := []string{
		"alert_firing",
		"deploy_age_gt",
		"deploy_age_lt",
		"k8s_deployment_exists",
		"manual",
		"service_healthy",
		"service_unhealthy",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("runbook check catalog drifted\n got: %v\nwant: %v", got, want)
	}
}

func TestKnownChecksRecognizedByVerifier(t *testing.T) {
	s, cleanup, err := store.OpenTemp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	v := runbook.NewVerifier(s, time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC))

	for _, name := range runbook.KnownCheckNames() {
		check := name
		if name != "manual" {
			check = name + ":arg"
		}
		res, err := v.Verify(model.Runbook{
			ServiceID: "svc",
			Steps:     []model.RunbookStep{{Number: 1, Text: "t", Check: check}},
		})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(res.Steps) != 1 {
			t.Fatalf("%s: want 1 step", name)
		}
		if strings.Contains(res.Steps[0].Message, "unknown check") {
			t.Fatalf("%s treated as unknown: %q", name, res.Steps[0].Message)
		}
	}
}
