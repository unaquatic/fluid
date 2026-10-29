package main

import (
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
)

//go:embed design_gallery.html
var galleryHTML string

var galleryTemplate = template.Must(template.New("gallery").Parse(galleryHTML))

// gallery is a page that draws golden files in color: for each snapshot, its
// cells in their foreground, background, and attributes, the colors it uses
// with their legend names, and the raw file.
type gallery struct {
	// Bg is the color of transparent cells.
	Bg template.CSS
	// Count is the number of files.
	Count int
	// Groups holds the files, one group per directory.
	Groups []galleryGroup
}

// galleryGroup is the golden files of one directory.
type galleryGroup struct {
	Dir   string
	Files []galleryFile
}

// galleryFile is one golden file.
type galleryFile struct {
	// ID is the file's anchor in the page.
	ID   string
	Name string
	// Meta summarizes the file: its size, or its step count for a sequence.
	Meta string
	// Steps holds the file's snapshots; a frames file has one, unnamed.
	Steps []galleryStep
	// Raw is the file's contents.
	Raw string
}

// galleryStep is one snapshot drawn as rows of styled runs.
type galleryStep struct {
	Name   string
	Size   string
	Rows   [][]styledRun
	Legend []legendColor
}

// styledRun is a stretch of a row whose cells share one style.
type styledRun struct {
	Style template.CSS
	Text  string
}

// legendColor describes one color a snapshot uses.
type legendColor struct {
	// Kind is "fg" or "bg".
	Kind string
	// Hex is the color as #rrggbb, or "-" when it is transparent.
	Hex string
	// Name is the color's legend name, if any.
	Name   string
	Swatch template.CSS
}

// runHTML runs the html subcommand with args, the arguments after its name:
// flags, then the directories whose golden files the page draws, the current
// directory by default. It writes the page to the file that -o names.
func runHTML(args []string) error {
	flags := flag.NewFlagSet("fluid design html", flag.ContinueOnError)
	out := flags.String("o", "goldens.html", "Write the page to `file`")
	bgHex := flags.String("bg", formatHex(style.DarculaSolid().Bg), "Show transparent cells on `color`, as #rrggbb")
	flags.Usage = func() {
		fmt.Fprint(flags.Output(), "Usage:\n  fluid design html [-o file] [-bg color] [dir...]\n\nFlags:\n")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	bg, err := parseHex(*bgHex)
	if err != nil || bg.IsTransparent() {
		return fmt.Errorf("-bg %q is not #rrggbb", *bgHex)
	}
	dirs := flags.Args()
	if len(dirs) == 0 {
		dirs = []string{"."}
	}
	g, err := loadGallery(bg, dirs...)
	if err != nil {
		return err
	}
	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	if err := errors.Join(g.writeHTML(f), f.Close()); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "fluid: wrote %d files to %s\n", g.Count, *out)
	return nil
}

// loadGallery returns a gallery of the golden frames and sequence files under
// each of dirs, as library.paths finds them, whose transparent cells show bg.
func loadGallery(bg style.Color, dirs ...string) (*gallery, error) {
	g := &gallery{Bg: template.CSS(formatHex(bg))}
	for _, dir := range dirs {
		if err := g.addDir(dir); err != nil {
			return nil, err
		}
	}
	return g, nil
}

// addDir adds the golden files under dir to g, in lexical order.
func (g *gallery) addDir(dir string) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	for p, err := range (library{root: root}).paths() {
		if err != nil {
			return err
		}
		data, err := root.ReadFile(p)
		if err != nil {
			return err
		}
		full := path.Join(filepath.ToSlash(dir), p)
		file, err := newGalleryFile(full, string(data))
		if err != nil {
			return err
		}
		g.add(path.Dir(full), file)
	}
	return nil
}

// add appends file to the group for dir, starting a group for a new dir, and
// gives the file the next ID.
func (g *gallery) add(dir string, file galleryFile) {
	g.Count++
	file.ID = fmt.Sprintf("f%d", g.Count)
	if n := len(g.Groups); n > 0 && g.Groups[n-1].Dir == dir {
		g.Groups[n-1].Files = append(g.Groups[n-1].Files, file)
		return
	}
	g.Groups = append(g.Groups, galleryGroup{Dir: dir, Files: []galleryFile{file}})
}

// writeHTML writes g to w as an HTML page.
func (g *gallery) writeHTML(w io.Writer) error {
	return galleryTemplate.Execute(w, g)
}

// newGalleryFile returns the golden file at p, whose contents are doc. Its ID
// is left for gallery.add to give.
func newGalleryFile(p, doc string) (galleryFile, error) {
	snaps, err := parseDoc(doc)
	if err != nil {
		return galleryFile{}, fmt.Errorf("%s: %w", p, err)
	}
	file := galleryFile{Name: path.Base(p), Raw: doc}
	for _, s := range snaps {
		file.Steps = append(file.Steps, newGalleryStep(s))
	}
	if len(file.Steps) == 1 && file.Steps[0].Name == "" {
		file.Meta = file.Steps[0].Size
	} else {
		file.Meta = fmt.Sprintf("sequence · %d steps", len(file.Steps))
	}
	return file, nil
}

// newGalleryStep returns s drawn as rows of styled runs, with the colors it
// uses in the order they first appear.
func newGalleryStep(s testutil.Snapshot) galleryStep {
	buf := s.Buf
	step := galleryStep{Name: s.Name, Size: fmt.Sprintf("%d×%d", buf.Width(), buf.Height())}
	var fgs, bgs []style.Color
	for y := range buf.Height() {
		var row []styledRun
		for x := range buf.Width() {
			c := buf.Cell(x, y)
			if c.IsContinuation() {
				continue
			}
			if !c.IsEmpty() && !slices.Contains(fgs, c.Fg) {
				fgs = append(fgs, c.Fg)
			}
			if !c.Bg.IsTransparent() && !slices.Contains(bgs, c.Bg) {
				bgs = append(bgs, c.Bg)
			}
			row = appendRun(row, c)
		}
		step.Rows = append(step.Rows, row)
	}
	step.Legend = slices.Concat(legend("fg", fgs, s.Names), legend("bg", bgs, s.Names))
	return step
}

// appendRun adds c to row, extending the last run when the styles match.
func appendRun(row []styledRun, c buffer.Cell) []styledRun {
	text := string(c.Char)
	if c.Char == 0 {
		text = " "
	}
	css := cellCSS(c)
	if n := len(row); n > 0 && row[n-1].Style == css {
		row[n-1].Text += text
		return row
	}
	return append(row, styledRun{Style: css, Text: text})
}

// cellCSS returns the CSS that draws c. A cell that shows no character has
// no foreground.
func cellCSS(c buffer.Cell) template.CSS {
	var rules []string
	if hex := formatHex(c.Fg); hex != "" && !c.IsEmpty() {
		rules = append(rules, "color:"+hex)
	}
	if hex := formatHex(c.Bg); hex != "" {
		rules = append(rules, "background:"+hex)
	}
	for _, r := range []struct {
		attr style.Attrs
		css  string
	}{
		{style.AttrBold, "font-weight:700"},
		{style.AttrItalic, "font-style:italic"},
		{style.AttrDim, "opacity:.55"},
	} {
		if c.Attributes&r.attr != 0 {
			rules = append(rules, r.css)
		}
	}
	var deco []string
	if c.Attributes&style.AttrUnderline != 0 {
		deco = append(deco, "underline")
	}
	if c.Attributes&style.AttrStrikethrough != 0 {
		deco = append(deco, "line-through")
	}
	if len(deco) > 0 {
		rules = append(rules, "text-decoration:"+strings.Join(deco, " "))
	}
	return template.CSS(strings.Join(rules, ";"))
}

// legend lists colors of kind with their names.
func legend(kind string, colors []style.Color, names map[style.Color]string) []legendColor {
	var out []legendColor
	for _, c := range colors {
		lc := legendColor{Kind: kind, Hex: "-", Name: names[c]}
		if hex := formatHex(c); hex != "" {
			lc.Hex = hex
			lc.Swatch = template.CSS("background:" + hex)
		}
		out = append(out, lc)
	}
	return out
}
