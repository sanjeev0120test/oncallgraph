//go:build ignore

// coverfloor checks go tool cover -func output against package floors.
// Usage:
//
//	go tool cover -func=cover.out | go run ./scripts/coverfloor.go total=62 cmd/opsgraph=60 internal/ai=35
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	floors := map[string]float64{}
	for _, arg := range os.Args[1:] {
		pkg, wantStr, ok := strings.Cut(arg, "=")
		if !ok {
			fatal("want pkg=percent, got %q", arg)
		}
		want, err := strconv.ParseFloat(strings.TrimSuffix(wantStr, "%"), 64)
		if err != nil {
			fatal("bad percent %q", wantStr)
		}
		floors[pkg] = want
	}
	if len(floors) == 0 {
		fatal("no floors provided")
	}

	type row struct {
		pkg string
		pct float64
	}
	var totals []row
	pkgSum := map[string]float64{}
	pkgN := map[string]int{}

	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "total:") {
			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			pctStr := strings.TrimSuffix(fields[len(fields)-1], "%")
			pct, err := strconv.ParseFloat(pctStr, 64)
			if err != nil {
				fatal("parse total: %v", err)
			}
			totals = append(totals, row{pkg: "total", pct: pct})
			continue
		}
		// path:func statements pct%
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pctStr := strings.TrimSuffix(fields[len(fields)-1], "%")
		pct, err := strconv.ParseFloat(pctStr, 64)
		if err != nil {
			continue
		}
		path := fields[0]
		// github.com/.../cmd/opsgraph/file.go:N:
		mod := "github.com/sanjeev0120test/opsgraph/"
		if !strings.HasPrefix(path, mod) {
			continue
		}
		rest := strings.TrimPrefix(path, mod)
		filePart, _, _ := strings.Cut(rest, ":")
		pkg := filePart
		if i := strings.LastIndex(filePart, "/"); i >= 0 {
			pkg = filePart[:i]
		}
		if strings.HasSuffix(pkg, ".go") {
			pkg = "."
		}
		pkgSum[pkg] += pct
		pkgN[pkg]++
	}
	if err := sc.Err(); err != nil {
		fatal("read: %v", err)
	}

	failed := false
	for pkg, want := range floors {
		var got float64
		var ok bool
		if pkg == "total" {
			if len(totals) == 0 {
				fatal("no total: line in cover -func output")
			}
			got = totals[0].pct
			ok = true
		} else {
			n := pkgN[pkg]
			if n == 0 {
				fmt.Fprintf(os.Stderr, "coverfloor: missing package %q in cover output\n", pkg)
				failed = true
				continue
			}
			got = pkgSum[pkg] / float64(n)
			ok = true
		}
		if !ok {
			continue
		}
		fmt.Printf("coverfloor %s: %.1f%% (floor %.1f%%)\n", pkg, got, want)
		if got+0.05 < want { // tiny float slack
			fmt.Fprintf(os.Stderr, "coverfloor: %s below floor: got %.1f%% want >= %.1f%%\n", pkg, got, want)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "coverfloor: "+format+"\n", args...)
	os.Exit(1)
}
