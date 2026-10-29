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

// designSynopsis lists the forms of "fluid design", one indented line each.
const designSynopsis = `  fluid design [-port n] [-root dir] [-theme file]
  fluid design ansi file...
  fluid design html [-o file] [-bg color] [dir...]
`

// serveOptions are the flags of the editor.
type serveOptions struct {
	port int
	// root is the directory whose golden files the editor opens and saves.
	root string
	// theme is the file holding the theme whose roles the palette offers, as
	// JSON, or "" for style.DarculaSolid.
	theme string
}

// runDesign runs "fluid design" with args, the arguments after its name:
// "ansi" and the files to print, "html" and the arguments of runHTML, or the
// flags of the editor, which it serves until ctx is done or an interrupt.
func runDesign(ctx context.Context, args []string) error {
	if len(args) > 0 && args[0] == "html" {
		return runHTML(args[1:])
	}
	if len(args) > 0 && args[0] == "ansi" {
		files := args[1:]
		if len(files) == 0 {
			return errors.New("ansi expects at least 1 file")
		}
		for _, p := range files {
			if err := printANSI(p); err != nil {
				return err
			}
		}
		return nil
	}
	var o serveOptions
	fs := flag.NewFlagSet("fluid design", flag.ContinueOnError)
	fs.IntVar(&o.port, "port", 4098, "Port to listen on")
	fs.StringVar(&o.root, "root", ".", "Open and save the golden files under `dir`")
	fs.StringVar(&o.theme, "theme", "", "Offer the roles of the style.Theme in JSON `file` (default style.DarculaSolid)")
	fs.Usage = func() {
		fmt.Fprint(fs.Output(), "Usage:\n"+designSynopsis+"\nFlags:\n")
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
	return serve(ctx, o)
}

// serve serves the editor as o says, on localhost, until ctx is done or an
// interrupt.
func serve(ctx context.Context, o serveOptions) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	th, err := loadTheme(o.theme)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(o.root)
	if err != nil {
		return err
	}
	defer root.Close()
	return listenAndServe(ctx, &server{library: library{root: root}, theme: th}, o.port)
}

// printANSI writes the snapshots of the golden file at p to standard output
// as writeANSI does, each step of a sequence after a line naming it.
func printANSI(p string) error {
	data, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	snaps, err := parseDoc(string(data))
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}
	for _, s := range snaps {
		if s.Name != "" {
			fmt.Printf("=== step: %s ===\n", s.Name)
		}
		if err := writeANSI(os.Stdout, s.Buf); err != nil {
			return err
		}
	}
	return nil
}
