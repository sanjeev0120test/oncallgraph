package main

import "github.com/spf13/cobra"

func bindSourceFlags(cmd *cobra.Command, src *sourceFlags) {
	cmd.Flags().StringVar(&src.fixture, "fixture", "", "path to a fixture pack directory (or OPSGRAPH_FIXTURE)")
	cmd.Flags().StringVar(&src.configPath, "config", "", "path to .opsgraph.yaml (or OPSGRAPH_CONFIG)")
	cmd.Flags().StringVar(&src.dataDir, "data-dir", "", "persistent store directory (or OPSGRAPH_DATA_DIR; default: .opsgraph/data)")
}

// bindFormatCompletion registers table|json completion for --format when present.
func bindFormatCompletion(cmd *cobra.Command) {
	if cmd.Flags().Lookup("format") == nil {
		return
	}
	_ = cmd.RegisterFlagCompletionFunc("format", completeFormatTableJSON)
}
