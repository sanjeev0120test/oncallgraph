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
