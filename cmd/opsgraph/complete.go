package main

import (
	"os"
	"strings"

	"github.com/sanjeev0120test/opsgraph/internal/config"
	"github.com/spf13/cobra"
)

// completeServiceArg suggests service IDs and aliases from the active source.
func completeServiceArg(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	max := 1
	if cmd.Name() == "path" || cmd.Name() == "compare" {
		max = 2
	}
	if len(args) >= max {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	fixture, _ := cmd.Flags().GetString("fixture")
	configPath, _ := cmd.Flags().GetString("config")
	dataDir, _ := cmd.Flags().GetString("data-dir")
	if fixture == "" {
		fixture = strings.TrimSpace(os.Getenv("OPSGRAPH_FIXTURE"))
	}
	cfg, err := config.Load(configPathOrEnv(configPath))
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ls, err := loadAskStore(cmd.Context(), fixture, configPathOrEnv(configPath), dataDir, cfg, cfg.Since())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	defer ls.cleanup()
	svcs, err := ls.store.ListServices()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	prefix := strings.ToLower(toComplete)
	out := make([]string, 0, len(svcs)*2)
	seen := map[string]struct{}{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if prefix != "" && !strings.HasPrefix(strings.ToLower(s), prefix) {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, s := range svcs {
		add(s.ID)
		for _, a := range s.Aliases {
			add(a)
		}
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}
