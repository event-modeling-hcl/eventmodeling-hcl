// playground-seed writes a JavaScript assignment containing an .em.hcl
// document. It is used by build-wasm so the editor always starts with the
// shipped minimal example.
package main

import (
	"fmt"
	"os"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/playground"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: playground-seed <model.em.hcl>")
		os.Exit(2)
	}
	source, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	script, err := playground.SeedScript(source)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_, _ = os.Stdout.Write(script)
}
