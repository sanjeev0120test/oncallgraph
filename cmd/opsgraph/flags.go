package main

import "github.com/spf13/cobra"

func bindSourceFlags(cmd *cobra.Command, src *sourceFlags) {
	cmd.Flags().StringVar(&src.fixture, "fixture", "", "path to a fixture pack directory")
	cmd.Flags().StringVar(&src.configPath, "config", "", "path to .opsgraph.yaml")
	cmd.Flags().StringVar(&src.dataDir, "data-dir", "", "persistent store directory (default: .opsgraph/data)")
}
