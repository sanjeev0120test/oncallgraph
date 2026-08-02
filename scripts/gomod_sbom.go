//go:build ignore

// gomod_sbom converts `go list -m -json all` (object stream) into one JSON array.
// Used by pack-release.sh so release packaging does not require python3.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func main() {
	dec := json.NewDecoder(os.Stdin)
	var objs []json.RawMessage
	for {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			if err == io.EOF {
				break
			}
			fmt.Fprintf(os.Stderr, "decode module json: %v\n", err)
			os.Exit(1)
		}
		objs = append(objs, raw)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(objs); err != nil {
		fmt.Fprintf(os.Stderr, "encode sbom: %v\n", err)
		os.Exit(1)
	}
}
