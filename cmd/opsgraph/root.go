package main

import (
	"github.com/sanjeev0120test/opsgraph/internal/version"
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
	root.AddGroup(&cobra.Group{ID: "core", Title: "Core Incident Commands"})
	root.AddGroup(&cobra.Group{ID: "fleet", Title: "Fleet & Topology"})
	root.AddGroup(&cobra.Group{ID: "signals", Title: "Signals & Evidence"})
	root.AddGroup(&cobra.Group{ID: "ops", Title: "Ops & Tooling"})

	add := func(group string, cmds ...*cobra.Command) {
		for _, c := range cmds {
			c.GroupID = group
			root.AddCommand(c)
		}
	}
	add("core",
		newAskCmd(), newWhyCmd(), newExplainCmd(), newHandoffCmd(),
		newScoreCmd(), newFingerprintCmd(), newVerifyRunbookCmd(),
		newReportCmd(), newExportCmd(), newWatchCmd(),
	)
	add("fleet",
		newServicesCmd(), newOwnersCmd(), newHealthCmd(), newTopCmd(),
		newBlastCmd(), newImpactCmd(), newPathCmd(), newGraphCmd(),
		newCompareCmd(), newResolveCmd(), newWhoCmd(),
	)
	add("signals",
		newChangesCmd(), newAlertsCmd(), newTimelineCmd(), newEvidenceCmd(),
	)
	add("ops",
		newDemoCmd(), newIngestCmd(), newStatusCmd(), newDoctorCmd(),
		newTestCmd(), newValidateFixtureCmd(), newCompletionCmd(), newVersionCmd(),
	)
	return root
}
