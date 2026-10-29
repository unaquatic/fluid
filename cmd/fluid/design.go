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

// DesignSynopsis lists the forms of "fluid design", one indented line each.
const DesignSynopsis = `  fluid design [-port n] [-root dir] [-theme file]
  fluid design ansi file...
  fluid design html [-o file] [-bg color] [dir...]
`

// ServeOptions are the flags of the editor.
type ServeOptions struct {
	Port int
	// Root is the directory whose golden files the editor opens and saves.
	Root string
	// Theme is the file holding the theme whose roles the palette offers, as
	// JSON, or "" for style.DarculaSolid.
	Theme string
}

// RunDesign runs "fluid design" with args, the arguments after its name:
// "ansi" and the files to print, "html" and the arguments of RunHTML, or the
// flags of the editor, which it serves until ctx is done or an interrupt.
func RunDesign(ctx context.Context, args []string) error {
	if len(args) > 0 && args[0] == "html" {
		return RunHTML(args[1:])
	}
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
	fs := flag.NewFlagSet("fluid design", flag.ContinueOnError)
	fs.IntVar(&o.Port, "port", 4098, "Port to listen on")
	fs.StringVar(&o.Root, "root", ".", "Open and save the golden files under `dir`")
	fs.StringVar(&o.Theme, "theme", "", "Offer the roles of the style.Theme in JSON `file` (default style.DarculaSolid)")
	fs.Usage = func() {
		fmt.Fprint(fs.Output(), "Usage:\n"+DesignSynopsis+"\nFlags:\n")
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
	return Serve(ctx, o)
}

// Serve serves the editor as o says, on localhost, until ctx is done or an
// interrupt.
func Serve(ctx context.Context, o ServeOptions) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
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
