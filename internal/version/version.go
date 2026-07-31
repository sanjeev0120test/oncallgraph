// Package version holds build metadata, injected via -ldflags at build time.
package version

// These are overridden at build time with:
//
//	-ldflags "-X github.com/opsgraph/opsgraph/internal/version.Version=..."
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String returns a human-readable version string.
func String() string {
	return Version + " (commit " + Commit + ", built " + Date + ")"
}
