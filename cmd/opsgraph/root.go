package main

import (
	"github.com/opsgraph/opsgraph/internal/version"
	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "opsgraph",
		Short: "Evidence-backed incident context for on-call engineers",
		Long: "opsgraph gathers what changed, what's affected, who owns it, and whether the\n" +
			"runbook is still valid - from local git, a Kubernetes snapshot, and fixtures.\n" +
			"It is free, offline-first, and needs no accounts or secrets.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.String(),
	}

	root.AddCommand(newVersionCmd())
	return root
}
