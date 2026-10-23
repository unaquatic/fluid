// Command fluid runs interactive demos of fluid's widgets, layout engine and
// rendering modes in the terminal.
//
// Usage:
//
//	fluid demo name [-fps n] [-no-mouse] [-bg hex]
//
// For example:
//
//	go run ./cmd/fluid demo dashboard
package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	if err := Run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "fluid:", err)
		os.Exit(1)
	}
}
