// Command design serves a page for drawing terminal screens cell by cell:
// typing text, painting foreground and background colors and attributes,
// drawing boxes, and moving regions around. The page opens the golden
// frames and sequence files under a directory, and saves designs there as
// golden frames files, the format that testutil.AssertGoldenFrames checks
// and agents read, written by testutil.GoldenFrames itself. It also exports
// a design as a PNG image or as text with ANSI escape sequences.
//
// The palette offers the roles of a theme, style.DarculaSolid unless -theme
// names a style.Theme as JSON, and the colors the golden files already use.
//
// Usage:
//
//	design [-port n] [-root dir] [-theme file]
//	design ansi file...
//
// For example, to paint a golden file of this repository and print it with
// ANSI escape sequences:
//
//	go run ./cmd/design
//	open 'http://localhost:4098/?open=widget/testdata/box_border_with_title.frames.golden'
//	go run ./cmd/design ansi widget/testdata/box_border_with_title.frames.golden
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if err := Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "design:", err)
		os.Exit(1)
	}
}

// ServeOptions are the flags of the editor.
type ServeOptions struct {
	Port int
	// Root is the directory whose golden files the editor opens and saves.
	Root string
	// Theme is the file holding the theme whose roles the palette offers, as
	// JSON, or "" for style.DarculaSolid.
	Theme string
}

// Run runs the command with args, the arguments after its name: "ansi" and
// the files to print, or the flags of the editor, which it serves.
func Run(args []string) error {
	if len(args) > 0 && args[0] == "ansi" {
		files := args[1:]
		if len(files) == 0 {
			return errors.New("ansi expects at least 1 file")
		}
		for _, p := range files {
			if err := PrintANSI(p); err != nil {
				return err
			}
		}
		return nil
	}
	var o ServeOptions
	fs := flag.NewFlagSet("design", flag.ContinueOnError)
	fs.IntVar(&o.Port, "port", 4098, "Port to listen on")
	fs.StringVar(&o.Root, "root", ".", "Open and save the golden files under `dir`")
	fs.StringVar(&o.Theme, "theme", "", "Offer the roles of the style.Theme in JSON `file` (default style.DarculaSolid)")
	fs.Usage = func() {
		fmt.Fprint(fs.Output(), "Usage:\n  design [-port n] [-root dir] [-theme file]\n  design ansi file...\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	return Serve(o)
}

// Serve serves the editor as o says, on localhost, until an interrupt.
func Serve(o ServeOptions) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	th, err := LoadTheme(o.Theme)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(o.Root)
	if err != nil {
		return err
	}
	defer root.Close()
	return ListenAndServe(ctx, &Server{Library: Library{Root: root}, Theme: th}, o.Port)
}

// PrintANSI writes the snapshots of the golden file at p to standard output
// as WriteANSI does, each step of a sequence after a line naming it.
func PrintANSI(p string) error {
	data, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	snaps, err := ParseDoc(string(data))
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}
	for _, s := range snaps {
		if s.Name != "" {
			fmt.Printf("=== step: %s ===\n", s.Name)
		}
		if err := WriteANSI(os.Stdout, s.Buf); err != nil {
			return err
		}
	}
	return nil
}
