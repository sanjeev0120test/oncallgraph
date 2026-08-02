//go:build ignore

// check_govuln_binary runs govulncheck -mode=binary -json and fails on any
// third-party finding OSV ID not listed in scripts/govulncheck-allowlist.txt.
//
// Stdlib findings are ignored here: source-mode govulncheck (required in CI)
// already covers reachable stdlib issues, and the Go version is pinned via go.mod.
//
// Usage: go run ./scripts/check_govuln_binary.go <binary>
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type findingEvent struct {
	Finding *struct {
		OSV   string `json:"osv"`
		Trace []struct {
			Module string `json:"module"`
		} `json:"trace"`
	} `json:"finding"`
}

func main() {
	if len(os.Args) != 2 {
		fatal("usage: check_govuln_binary <binary>")
	}
	bin := os.Args[1]
	if _, err := os.Stat(bin); err != nil {
		fatal("binary: %v", err)
	}
	allow, err := loadAllowlist("scripts/govulncheck-allowlist.txt")
	if err != nil {
		fatal("%v", err)
	}

	cmd := exec.Command("go", "tool", "govulncheck", "-mode=binary", "-json", bin)
	out, runErr := cmd.CombinedOutput()
	thirdParty, stdlibN, err := parseFindings(out)
	if err != nil {
		fatal("parse govulncheck json: %v\n%s", err, truncate(out, 2000))
	}
	if len(thirdParty) == 0 && stdlibN == 0 && runErr != nil {
		fatal("govulncheck failed: %v\n%s", runErr, truncate(out, 2000))
	}

	var unexpected []string
	for _, id := range thirdParty {
		if !allow[id] {
			unexpected = append(unexpected, id)
		}
	}
	if len(unexpected) > 0 {
		fmt.Fprintln(os.Stderr, "check_govuln_binary: unexpected third-party binary findings:")
		for _, id := range unexpected {
			fmt.Fprintln(os.Stderr, " ", id)
		}
		fmt.Fprintln(os.Stderr, "Add an explicit entry to scripts/govulncheck-allowlist.txt only after review.")
		os.Exit(1)
	}
	fmt.Printf("check_govuln_binary: ok (third-party=%d allowlisted, stdlib_ignored=%d)\n", len(thirdParty), stdlibN)
}

func loadAllowlist(path string) (map[string]bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]bool{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line != "" {
			out[line] = true
		}
	}
	return out, sc.Err()
}

func parseFindings(raw []byte) (thirdParty []string, stdlibN int, err error) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	seenThird := map[string]bool{}
	seenStd := map[string]bool{}
	for dec.More() {
		var ev findingEvent
		if err := dec.Decode(&ev); err != nil {
			return nil, 0, err
		}
		if ev.Finding == nil || ev.Finding.OSV == "" {
			continue
		}
		id := ev.Finding.OSV
		if isStdlibOnly(ev.Finding.Trace) {
			if !seenStd[id] {
				seenStd[id] = true
				stdlibN++
			}
			continue
		}
		if !seenThird[id] {
			seenThird[id] = true
			thirdParty = append(thirdParty, id)
		}
	}
	return thirdParty, stdlibN, nil
}

func isStdlibOnly(trace []struct {
	Module string `json:"module"`
}) bool {
	if len(trace) == 0 {
		return false
	}
	for _, t := range trace {
		if t.Module != "" && t.Module != "stdlib" {
			return false
		}
	}
	return true
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "\n...truncated..."
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "check_govuln_binary: "+format+"\n", args...)
	os.Exit(1)
}
