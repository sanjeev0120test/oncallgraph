// Package runbook parses Markdown runbooks (front matter + step checks) and
// verifies them against current incident state.
package runbook

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sanjeev0120test/opsgraph/internal/model"
	"gopkg.in/yaml.v3"
)

// FrontMatter is the YAML block at the top of a runbook.
type FrontMatter struct {
	Service string   `yaml:"service"`
	Owner   string   `yaml:"owner"`
	Aliases []string `yaml:"aliases"`
}

var (
	stepRe = regexp.MustCompile(`^\s*(\d+)\.\s+(.*\S)\s*$`)
	checkRe = regexp.MustCompile(`opsgraph:check=([^\s]+)\s*-->`)
)

// Parse parses a runbook's bytes. path is stored on the result for reference.
func Parse(data []byte, path string) (model.Runbook, FrontMatter, error) {
	body, fm, err := splitFrontMatter(data)
	if err != nil {
		return model.Runbook{}, FrontMatter{}, err
	}

	rb := model.Runbook{
		ServiceID: fm.Service,
		Path:      path,
		OwnerID:   fm.Owner,
	}
	if fm.Service != "" {
		rb.ID = "runbook-" + fm.Service
	}

	sc := bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	curr := -1 // index into rb.Steps of the step a check would bind to
	for sc.Scan() {
		line := sc.Text()

		if m := stepRe.FindStringSubmatch(line); m != nil {
			num, err := strconv.Atoi(m[1])
			if err != nil || num <= 0 {
				num = len(rb.Steps) + 1
			}
			rb.Steps = append(rb.Steps, model.RunbookStep{
				Number: num,
				Text:   strings.TrimSpace(m[2]),
			})
			curr = len(rb.Steps) - 1
			continue
		}
		if m := checkRe.FindStringSubmatch(line); m != nil && curr >= 0 {
			// Bind the check to the nearest preceding step (first check wins).
			if rb.Steps[curr].Check == "" {
				rb.Steps[curr].Check = strings.TrimSpace(m[1])
			}
		}
	}
	if err := sc.Err(); err != nil {
		return model.Runbook{}, FrontMatter{}, fmt.Errorf("scan runbook %q: %w", path, err)
	}
	return rb, fm, nil
}

// splitFrontMatter separates a leading `---` YAML block from the body.
func splitFrontMatter(data []byte) ([]byte, FrontMatter, error) {
	var fm FrontMatter
	s := string(data)
	s = strings.TrimLeft(s, "\ufeff") // strip BOM if present
	if !strings.HasPrefix(s, "---\n") && !strings.HasPrefix(s, "---\r\n") {
		return data, fm, nil
	}
	// Find the closing delimiter line.
	rest := s[strings.Index(s, "\n")+1:]
	end := regexp.MustCompile(`(?m)^---\s*$`).FindStringIndex(rest)
	if end == nil {
		return data, fm, nil // no closing fence; treat all as body
	}
	yamlBlock := rest[:end[0]]
	body := rest[end[1]:]
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return nil, FrontMatter{}, fmt.Errorf("parse runbook front matter: %w", err)
	}
	return []byte(body), fm, nil
}
