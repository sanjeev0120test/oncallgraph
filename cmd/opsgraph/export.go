package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/sanjeev0120test/opsgraph/internal/ask"
	"github.com/sanjeev0120test/opsgraph/internal/output"
	"github.com/sanjeev0120test/opsgraph/internal/report"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	var src sourceFlags
	var since time.Duration
	var outPath string
	var format string
	cmd := &cobra.Command{
		Use:   "export <service>",
		Short: "Export ask result to a file (json or markdown)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "json" && format != "markdown" && format != "md" {
				return fail(2, "invalid --format %q (want json|markdown)", format)
			}
			if outPath == "" {
				ext := "json"
				if format != "json" {
					ext = "md"
				}
				outPath = sanitizeFileStem(args[0]) + "-incident." + ext
			}
			ls, cfg, err := src.load(since)
			if err != nil {
				return failSource(err)
			}
			defer ls.cleanup()
			if since == 0 {
				since = cfg.Since()
			}
			res, err := askService(ls, args[0], since)
			if err != nil {
				if errors.Is(err, ask.ErrServiceNotFound) {
					return fail(1, "%v", err)
				}
				return fail(2, "%v", err)
			}
			if dir := filepath.Dir(outPath); dir != "." && dir != "" {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return fail(2, "%v", err)
				}
			}
			f, err := os.Create(outPath)
			if err != nil {
				return fail(2, "create %s: %v", outPath, err)
			}
			defer f.Close()
			if format == "json" {
				if err := output.JSON(f, res); err != nil {
					return fail(2, "%v", err)
				}
			} else if _, err := f.WriteString(report.Markdown(res)); err != nil {
				return fail(2, "%v", err)
			}
			cmd.Printf("wrote %s\n", outPath)
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window")
	cmd.Flags().StringVar(&outPath, "out", "", "output file path (default: <service>-incident.json|md)")
	cmd.Flags().StringVar(&format, "format", "markdown", "output format: json|markdown")
	return cmd
}

func sanitizeFileStem(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "service"
	}
	return out
}
