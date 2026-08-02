package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestFilterPrefix(t *testing.T) {
	got := filterPrefix([]string{"table", "json", "markdown"}, "j")
	if len(got) != 1 || got[0] != "json" {
		t.Fatalf("got %v", got)
	}
	got = filterPrefix([]string{"healthy", "degraded"}, "")
	if len(got) != 2 {
		t.Fatalf("empty prefix should keep all: %v", got)
	}
}

func TestCompleteFormatHelpers(t *testing.T) {
	cmd := &cobra.Command{}
	vals, _ := completeFormatTableJSON(cmd, nil, "t")
	if len(vals) != 1 || vals[0] != "table" {
		t.Fatalf("table json: %v", vals)
	}
	vals, _ = completeFormatGraph(cmd, nil, "m")
	if len(vals) != 1 || vals[0] != "mermaid" {
		t.Fatalf("graph: %v", vals)
	}
	vals, _ = completeFormatMarkdownJSON(cmd, nil, "md")
	if len(vals) != 0 {
		t.Fatalf("markdown prefix md should miss: %v", vals)
	}
	vals, _ = completeFormatMarkdownJSON(cmd, nil, "mark")
	if len(vals) != 1 || vals[0] != "markdown" {
		t.Fatalf("markdown: %v", vals)
	}
	vals, _ = completeHealthValues(cmd, nil, "un")
	if len(vals) != 2 { // unknown, unhealthy
		t.Fatalf("health: %v", vals)
	}
}

func TestCompleteServiceArgFromFixture(t *testing.T) {
	fx := fixtureDir(t)
	root := newRootCmd()
	ask, _, err := root.Find([]string{"ask"})
	if err != nil {
		t.Fatal(err)
	}
	if err := ask.Flags().Set("fixture", fx); err != nil {
		t.Fatal(err)
	}
	vals, dir := completeServiceArg(ask, nil, "check")
	if dir != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive=%v", dir)
	}
	found := false
	for _, v := range vals {
		if v == "checkout" || v == "checkout-api" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected checkout suggestions, got %v", vals)
	}
}
