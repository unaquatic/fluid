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

// Gallery is a page that draws golden files in color: for each snapshot, its
// cells in their foreground, background, and attributes, the colors it uses
// with their legend names, and the raw file.
type Gallery struct {
	// Bg is the color of transparent cells.
	Bg template.CSS
	// Count is the number of files.
	Count int
	// Groups holds the files, one group per directory.
	Groups []GalleryGroup
}

// GalleryGroup is the golden files of one directory.
type GalleryGroup struct {
	Dir   string
	Files []GalleryFile
}

// GalleryFile is one golden file.
type GalleryFile struct {
	// ID is the file's anchor in the page.
	ID   string
	Name string
	// Meta summarizes the file: its size, or its step count for a sequence.
	Meta string
	// Steps holds the file's snapshots; a frames file has one, unnamed.
	Steps []GalleryStep
	// Raw is the file's contents.
	Raw string
}

// GalleryStep is one snapshot drawn as rows of styled runs.
type GalleryStep struct {
	Name   string
	Size   string
	Rows   [][]StyledRun
	Legend []LegendColor
}

// StyledRun is a stretch of a row whose cells share one style.
type StyledRun struct {
	Style template.CSS
	Text  string
}

// LegendColor describes one color a snapshot uses.
type LegendColor struct {
	// Kind is "fg" or "bg".
	Kind string
	// Hex is the color as #rrggbb, or "-" when it is transparent.
	Hex string
	// Name is the color's legend name, if any.
	Name   string
	Swatch template.CSS
}

// RunHTML runs the html subcommand with args, the arguments after its name:
// flags, then the directories whose golden files the page draws, the current
// directory by default. It writes the page to the file that -o names.
func RunHTML(args []string) error {
	flags := flag.NewFlagSet("fluid design html", flag.ContinueOnError)
	out := flags.String("o", "goldens.html", "Write the page to `file`")
	bgHex := flags.String("bg", Hex(style.DarculaSolid().Bg), "Show transparent cells on `color`, as #rrggbb")
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
	bg, err := ParseHex(*bgHex)
	if err != nil || bg.IsTransparent() {
		return fmt.Errorf("-bg %q is not #rrggbb", *bgHex)
	}
	dirs := flags.Args()
	if len(dirs) == 0 {
		dirs = []string{"."}
	}
	g, err := LoadGallery(bg, dirs...)
	if err != nil {
		return err
	}
	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	if err := errors.Join(g.WriteHTML(f), f.Close()); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "fluid: wrote %d files to %s\n", g.Count, *out)
	return nil
}

// LoadGallery returns a gallery of the golden frames and sequence files under
// each of dirs, as Library.Paths finds them, whose transparent cells show bg.
func LoadGallery(bg style.Color, dirs ...string) (*Gallery, error) {
	g := &Gallery{Bg: template.CSS(Hex(bg))}
	for _, dir := range dirs {
		if err := g.AddDir(dir); err != nil {
			return nil, err
		}
	}
	return g, nil
}

// AddDir adds the golden files under dir to g, in lexical order.
func (g *Gallery) AddDir(dir string) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	for p, err := range (Library{Root: root}).Paths() {
		if err != nil {
			return err
		}
		data, err := root.ReadFile(p)
		if err != nil {
			return err
		}
		full := path.Join(filepath.ToSlash(dir), p)
		file, err := NewGalleryFile(full, string(data))
		if err != nil {
			return err
		}
		g.Add(path.Dir(full), file)
	}
	return nil
}

// Add appends file to the group for dir, starting a group for a new dir, and
// gives the file the next ID.
func (g *Gallery) Add(dir string, file GalleryFile) {
	g.Count++
	file.ID = fmt.Sprintf("f%d", g.Count)
	if n := len(g.Groups); n > 0 && g.Groups[n-1].Dir == dir {
		g.Groups[n-1].Files = append(g.Groups[n-1].Files, file)
		return
	}
	g.Groups = append(g.Groups, GalleryGroup{Dir: dir, Files: []GalleryFile{file}})
}

// WriteHTML writes g to w as an HTML page.
func (g *Gallery) WriteHTML(w io.Writer) error {
	return galleryTemplate.Execute(w, g)
}

// NewGalleryFile returns the golden file at p, whose contents are doc. Its ID
// is left for Gallery.Add to give.
func NewGalleryFile(p, doc string) (GalleryFile, error) {
	snaps, err := ParseDoc(doc)
	if err != nil {
		return GalleryFile{}, fmt.Errorf("%s: %w", p, err)
	}
	file := GalleryFile{Name: path.Base(p), Raw: doc}
	for _, s := range snaps {
		file.Steps = append(file.Steps, NewGalleryStep(s))
	}
	if len(file.Steps) == 1 && file.Steps[0].Name == "" {
		file.Meta = file.Steps[0].Size
	} else {
		file.Meta = fmt.Sprintf("sequence · %d steps", len(file.Steps))
	}
	return file, nil
}

// NewGalleryStep returns s drawn as rows of styled runs, with the colors it
// uses in the order they first appear.
func NewGalleryStep(s testutil.Snapshot) GalleryStep {
	buf := s.Buf
	step := GalleryStep{Name: s.Name, Size: fmt.Sprintf("%d×%d", buf.Width(), buf.Height())}
	var fgs, bgs []style.Color
	for y := range buf.Height() {
		var row []StyledRun
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
func appendRun(row []StyledRun, c buffer.Cell) []StyledRun {
	text := string(c.Char)
	if c.Char == 0 {
		text = " "
	}
	css := CellCSS(c)
	if n := len(row); n > 0 && row[n-1].Style == css {
		row[n-1].Text += text
		return row
	}
	return append(row, StyledRun{Style: css, Text: text})
}

// CellCSS returns the CSS that draws c. A cell that shows no character has
// no foreground.
func CellCSS(c buffer.Cell) template.CSS {
	var rules []string
	if hex := Hex(c.Fg); hex != "" && !c.IsEmpty() {
		rules = append(rules, "color:"+hex)
	}
	if hex := Hex(c.Bg); hex != "" {
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
func legend(kind string, colors []style.Color, names map[style.Color]string) []LegendColor {
	var out []LegendColor
	for _, c := range colors {
		lc := LegendColor{Kind: kind, Hex: "-", Name: names[c]}
		if hex := Hex(c); hex != "" {
			lc.Hex = hex
			lc.Swatch = template.CSS("background:" + hex)
		}
		out = append(out, lc)
	}
	return out
}
