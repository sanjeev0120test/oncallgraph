package output_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sanjeev0120test/oncallgraph/internal/model"
	"github.com/sanjeev0120test/oncallgraph/internal/output"
)

func TestJSONDoesNotEscapeHTML(t *testing.T) {
	var buf bytes.Buffer
	v := model.AskResult{
		Service: model.Service{ID: "x", Name: "x"},
		Owner:   &model.Owner{ID: "o", Name: "Team", Email: "a@b.com"},
		Window:  "60m",
		Recommendations: []string{
			"Notify owner Team <a@b.com>.",
		},
	}
	if err := output.JSON(&buf, v); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if strings.Contains(got, `\u003c`) || strings.Contains(got, `\u003e`) {
		t.Fatalf("HTML should not be escaped: %s", got)
	}
	if !strings.Contains(got, "<a@b.com>") {
		t.Fatalf("expected literal angle brackets: %s", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Fatal("JSON output must end with newline")
	}
}
