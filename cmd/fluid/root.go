package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

// demoOptions are the rendering flags every demo accepts.
type demoOptions struct {
	noMouse bool
}

func demoFlags(fs *flag.FlagSet, o *demoOptions) {
	fs.BoolVar(&o.noMouse, "no-mouse", false, "Disable mouse input")
}

// run runs the command with args, the arguments after its name: "demo"
// followed by the arguments of runDemo, or "design" followed by the arguments
// of runDesign. With no arguments or a help flag, it prints the usage.
func run(ctx context.Context, args []string) error {
	if len(args) == 0 || isHelp(args[0]) {
		writeUsage(os.Stdout)
		return nil
	}
	switch args[0] {
	case "demo":
		return runDemo(ctx, args[1:])
	case "design":
		return runDesign(ctx, args[1:])
	}
	return fmt.Errorf("unknown command %q", args[0])
}

// isHelp reports whether arg is a flag that asks for help.
func isHelp(arg string) bool {
	switch arg {
	case "-h", "-help", "--help":
		return true
	}
	return false
}
