package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/sanjeev0120test/opsgraph/internal/output"
	"github.com/sanjeev0120test/opsgraph/internal/report"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	var src sourceFlags
	var since time.Duration
	var outPath string
	var format string
	var meta bool
	cmd := &cobra.Command{
		Use:               "export <service>",
		Short:             "Export ask result to a file (json or markdown)",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeServiceArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireArg("service", args[0]); err != nil {
				return err
			}
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
			ls, cfg, err := src.loadCtx(cmd.Context(), since)
			if err != nil {
				return failSource(err)
			}
			defer ls.cleanup()
			if since == 0 {
				since = cfg.Since()
			}
			res, err := askService(ls, args[0], since)
			if err != nil {
				return failAsk(err)
			}
			if dir := filepath.Dir(outPath); dir != "." && dir != "" {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return fail(2, "%v", err)
				}
			}
			tmpPath := outPath + ".tmp"
			f, err := os.Create(tmpPath)
			if err != nil {
				return fail(2, "create %s: %v", tmpPath, err)
			}
			writeErr := func() error {
				if format == "json" {
					return output.JSON(f, res)
				}
				md := report.Markdown(res)
				if !strings.HasSuffix(md, "\n") {
					md += "\n"
				}
				_, err := f.WriteString(md)
				return err
			}()
			if writeErr != nil {
				_ = f.Close()
				_ = os.Remove(tmpPath)
				return fail(2, "%v", writeErr)
			}
			if err := f.Sync(); err != nil {
				_ = f.Close()
				_ = os.Remove(tmpPath)
				return fail(2, "sync %s: %v", tmpPath, err)
			}
			if err := f.Close(); err != nil {
				_ = os.Remove(tmpPath)
				return fail(2, "close %s: %v", tmpPath, err)
			}
			if err := os.Rename(tmpPath, outPath); err != nil {
				// Windows cannot rename over an existing file — stage via backup
				// so a crash cannot leave the destination deleted.
				bak := outPath + ".bak"
				_ = os.Remove(bak)
				if _, statErr := os.Stat(outPath); statErr == nil {
					if err := os.Rename(outPath, bak); err != nil {
						_ = os.Remove(tmpPath)
						return fail(2, "backup %s: %v", outPath, err)
					}
				}
				if err := os.Rename(tmpPath, outPath); err != nil {
					_ = os.Rename(bak, outPath) // best-effort restore
					_ = os.Remove(tmpPath)
					return fail(2, "rename %s: %v", outPath, err)
				}
				_ = os.Remove(bak)
			}
			info, err := os.Stat(outPath)
			if err != nil {
				return fail(2, "%v", err)
			}
			if meta {
				return output.JSON(cmd.OutOrStdout(), map[string]any{
					"path":    outPath,
					"bytes":   info.Size(),
					"service": res.Service.ID,
					"format":  format,
				})
			}
			fmt.Fprintln(cmd.OutOrStdout(), "wrote", outPath)
			return nil
		},
	}
	bindSourceFlags(cmd, &src)
	cmd.Flags().DurationVar(&since, "since", 0, "lookback window")
	cmd.Flags().StringVar(&outPath, "out", "", "output file path (default: <service>-incident.json|md)")
	cmd.Flags().StringVar(&format, "format", "markdown", "output format: json|markdown")
	_ = cmd.RegisterFlagCompletionFunc("format", completeFormatMarkdownJSON)
	cmd.Flags().BoolVar(&meta, "meta", false, "print JSON {path,bytes,service,format} instead of wrote line")
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
