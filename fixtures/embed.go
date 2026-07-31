// Package fixtures embeds incident packs so commands like `demo` work from any
// working directory. The embed directive lives here (inside fixtures/) because
// go:embed cannot reference parent directories.
package fixtures

import (
	"embed"
	"io/fs"
)

//go:embed incident_checkout
var embedded embed.FS

// CheckoutFS returns the embedded incident_checkout pack as a filesystem rooted
// at the pack directory (so files are addressed as "services.yaml", etc.).
func CheckoutFS() (fs.FS, error) {
	return fs.Sub(embedded, "incident_checkout")
}
