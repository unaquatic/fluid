package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
)

// demo is an interactive demo that "fluid demo" runs by name.
type demo struct {
	name  string
	short string
	run   func(ctx context.Context, o demoOptions) error
}

// demos are the interactive component demos, in the order the usage lists
// them.
var demos = []demo{
	demoExtmarksCmd,
	demoMeasureCmd,
	demoLayoutCmd,
	demoScrollboxCmd,
	demoSelectCmd,
	demoCodeCmd,
	demoMarkdownCmd,
	demoDashboardCmd,
	demoNestedScrollCmd,
	demoInlineCmd,
	demoInlineInputCmd,
	demoTabCmd,
	demoColorsCmd,
}

// lookupDemo returns the demo called name, and false if there is none.
func lookupDemo(name string) (demo, bool) {
	i := slices.IndexFunc(demos, func(d demo) bool { return d.name == name })
	if i < 0 {
		return demo{}, false
	}
	return demos[i], true
}

// runDemo runs the demo that args name first, with the flags that follow its
// name. With no name or a help flag, it prints the usage.
func runDemo(ctx context.Context, args []string) error {
	if len(args) == 0 || isHelp(args[0]) {
		writeUsage(os.Stdout)
		return nil
	}
	d, ok := lookupDemo(args[0])
	if !ok {
		return fmt.Errorf("unknown demo %q", args[0])
	}
	var o demoOptions
	fs := flag.NewFlagSet("fluid demo "+d.name, flag.ContinueOnError)
	demoFlags(fs, &o)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage:\n  fluid demo %s [flags]\n\n%s.\n\nFlags:\n", d.name, d.short)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	return d.run(ctx, o)
}

// writeUsage writes the usage of the command to w: its synopsis, the demos
// and the flags every demo accepts.
func writeUsage(w io.Writer) {
	fmt.Fprint(w, "Usage:\n  fluid demo name [flags]\n"+designSynopsis+"\nDemos:\n")
	width := 0
	for _, d := range demos {
		width = max(width, len(d.name))
	}
	for _, d := range demos {
		fmt.Fprintf(w, "  %-*s  %s\n", width, d.name, d.short)
	}
	fmt.Fprint(w, "\nDemo flags:\n")
	fs := flag.NewFlagSet("fluid demo", flag.ContinueOnError)
	demoFlags(fs, new(demoOptions))
	fs.SetOutput(w)
	fs.PrintDefaults()
}
