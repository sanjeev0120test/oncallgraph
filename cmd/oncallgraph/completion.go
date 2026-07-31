package main

import (
	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate completion scripts for your shell.

Examples:
  oncallgraph completion bash > /etc/bash_completion.d/oncallgraph
  oncallgraph completion powershell | Out-String | Invoke-Expression
`,
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			w := cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(w)
			case "zsh":
				return root.GenZshCompletion(w)
			case "fish":
				return root.GenFishCompletion(w, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(w)
			default:
				return fail(2, "unsupported shell %q", args[0])
			}
		},
	}
	return cmd
}
