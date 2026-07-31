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

	root.AddCommand(newVersionCmd())
	root.AddCommand(newAskCmd())
	root.AddCommand(newVerifyRunbookCmd())
	root.AddCommand(newIngestCmd())
	root.AddCommand(newDemoCmd())
	root.AddCommand(newTestCmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newServicesCmd())
	root.AddCommand(newOwnersCmd())
	root.AddCommand(newGraphCmd())
	root.AddCommand(newEvidenceCmd())
	root.AddCommand(newExplainCmd())
	root.AddCommand(newReportCmd())
	root.AddCommand(newScoreCmd())
	root.AddCommand(newWhoCmd())
	root.AddCommand(newCompareCmd())
	root.AddCommand(newTimelineCmd())
	root.AddCommand(newPathCmd())
	root.AddCommand(newBlastCmd())
	root.AddCommand(newWatchCmd())
	root.AddCommand(newHealthCmd())
	root.AddCommand(newTopCmd())
	root.AddCommand(newResolveCmd())
	root.AddCommand(newChangesCmd())
	root.AddCommand(newAlertsCmd())
	root.AddCommand(newImpactCmd())
	root.AddCommand(newFingerprintCmd())
	root.AddCommand(newWhyCmd())
	root.AddCommand(newHandoffCmd())
	root.AddCommand(newExportCmd())
	root.AddCommand(newDoctorCmd())
	root.AddCommand(newValidateFixtureCmd())
	root.AddCommand(newCompletionCmd())
	return root
}
