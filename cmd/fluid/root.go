package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

// DemoOptions are the rendering flags every demo accepts.
type DemoOptions struct {
	NoMouse bool
}

func demoFlags(fs *flag.FlagSet, o *DemoOptions) {
	fs.BoolVar(&o.NoMouse, "no-mouse", false, "Disable mouse input")
}

// Run runs the command with args, the arguments after its name: "demo"
// followed by the arguments of RunDemo. With no arguments or a help flag, it
// prints the usage.
func Run(ctx context.Context, args []string) error {
	if len(args) == 0 || IsHelp(args[0]) {
		WriteUsage(os.Stdout)
		return nil
	}
	if args[0] != "demo" {
		return fmt.Errorf("unknown command %q", args[0])
	}
	return RunDemo(ctx, args[1:])
}

// IsHelp reports whether arg is a flag that asks for help.
func IsHelp(arg string) bool {
	switch arg {
	case "-h", "-help", "--help":
		return true
	}
	return false
}
