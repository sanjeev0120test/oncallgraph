package runbook

import "sort"

// KnownCheckNames is the frozen authoring catalog for `opsgraph:check=`.
// Renaming or dropping a name is a breaking change for every runbook in the wild.
func KnownCheckNames() []string {
	names := []string{
		"alert_firing",
		"deploy_age_gt",
		"deploy_age_lt",
		"k8s_deployment_exists",
		"manual",
		"service_healthy",
		"service_unhealthy",
	}
	sort.Strings(names)
	return names
}
