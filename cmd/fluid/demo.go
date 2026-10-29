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

// Demo is an interactive demo that "fluid demo" runs by name.
type Demo struct {
	Name  string
	Short string
	Run   func(ctx context.Context, o DemoOptions) error
}

// Demos are the interactive component demos, in the order the usage lists
// them.
var Demos = []Demo{
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

// LookupDemo returns the demo called name, and false if there is none.
func LookupDemo(name string) (Demo, bool) {
	i := slices.IndexFunc(Demos, func(d Demo) bool { return d.Name == name })
	if i < 0 {
		return Demo{}, false
	}
	return Demos[i], true
}

// RunDemo runs the demo that args name first, with the flags that follow its
// name. With no name or a help flag, it prints the usage.
func RunDemo(ctx context.Context, args []string) error {
	if len(args) == 0 || IsHelp(args[0]) {
		WriteUsage(os.Stdout)
		return nil
	}
	d, ok := LookupDemo(args[0])
	if !ok {
		return fmt.Errorf("unknown demo %q", args[0])
	}
	var o DemoOptions
	fs := flag.NewFlagSet("fluid demo "+d.Name, flag.ContinueOnError)
	demoFlags(fs, &o)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage:\n  fluid demo %s [flags]\n\n%s.\n\nFlags:\n", d.Name, d.Short)
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
	return d.Run(ctx, o)
}

// WriteUsage writes the usage of the command to w: its synopsis, the demos
// and the flags every demo accepts.
func WriteUsage(w io.Writer) {
	fmt.Fprint(w, "Usage:\n  fluid demo name [flags]\n"+DesignSynopsis+"\nDemos:\n")
	width := 0
	for _, d := range Demos {
		width = max(width, len(d.Name))
	}
	for _, d := range Demos {
		fmt.Fprintf(w, "  %-*s  %s\n", width, d.Name, d.Short)
	}
	fmt.Fprint(w, "\nDemo flags:\n")
	fs := flag.NewFlagSet("fluid demo", flag.ContinueOnError)
	demoFlags(fs, new(DemoOptions))
	fs.SetOutput(w)
	fs.PrintDefaults()
}
