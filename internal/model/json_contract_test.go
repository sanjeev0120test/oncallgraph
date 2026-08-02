package model

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

// Machine JSON field-set freeze: renaming/dropping exported tags breaks
// enterprise consumers and must fail CI.
func TestAskResultJSONKeysFrozen(t *testing.T) {
	got := jsonFieldKeys(AskResult{})
	want := []string{
		"ai_summary",
		"alerts",
		"changes",
		"correlations",
		"downstream",
		"evidence",
		"generated_at",
		"owner",
		"recommendations",
		"runbook",
		"service",
		"timeline",
		"upstream",
		"window",
	}
	assertKeys(t, "AskResult", got, want)
}

func TestVerifyResultJSONKeysFrozen(t *testing.T) {
	got := jsonFieldKeys(VerifyResult{})
	want := []string{
		"path",
		"service_id",
		"status",
		"steps",
	}
	assertKeys(t, "VerifyResult", got, want)
}

func TestStepVerifyResultJSONKeysFrozen(t *testing.T) {
	got := jsonFieldKeys(StepVerifyResult{})
	want := []string{
		"check",
		"evidence_id",
		"message",
		"number",
		"status",
		"text",
	}
	assertKeys(t, "StepVerifyResult", got, want)
}

func TestNestedMachineJSONKeysFrozen(t *testing.T) {
	cases := []struct {
		label string
		got   []string
		want  []string
	}{
		{"Service", jsonFieldKeys(Service{}), []string{"aliases", "health", "id", "labels", "name", "owner_id", "sources"}},
		{"Owner", jsonFieldKeys(Owner{}), []string{"email", "id", "name", "team"}},
		{"Change", jsonFieldKeys(Change{}), []string{"at", "author", "evidence_id", "id", "revision", "service_id", "source", "summary", "type"}},
		{"Alert", jsonFieldKeys(Alert{}), []string{"at", "evidence_id", "id", "name", "service_id", "severity", "source", "status", "summary"}},
		{"Evidence", jsonFieldKeys(Evidence{}), []string{"at", "id", "kind", "raw_ref", "service_id", "source", "summary"}},
		{"TimelineEvent", jsonFieldKeys(TimelineEvent{}), []string{"at", "evidence_id", "kind", "service_id", "severity", "summary"}},
		{"Correlation", jsonFieldKeys(Correlation{}), []string{"alert_evidence_id", "alert_id", "change_evidence_id", "change_id", "gap", "kind", "summary"}},
		{"Dependency", jsonFieldKeys(Dependency{}), []string{"from_service_id", "source", "to_service_id", "type"}},
	}
	for _, tc := range cases {
		assertKeys(t, tc.label, tc.got, tc.want)
	}
}

func jsonFieldKeys(v any) []string {
	typ := reflect.TypeOf(v)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	var keys []string
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		if name == "" || name == "-" {
			continue
		}
		keys = append(keys, name)
	}
	sort.Strings(keys)
	return keys
}

func assertKeys(t *testing.T, label string, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s JSON keys drifted\n got: %v\nwant: %v", label, got, want)
	}
}
