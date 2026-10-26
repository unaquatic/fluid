// Goldenview writes an HTML page that draws golden frames files in color.
//
// Usage:
//
//	goldenview [-o file] [-bg color] [dir ...]
//
// Goldenview walks each dir, the current directory by default, and draws every
// *.frames.golden and *.seq.golden file it finds in a testdata directory. The
// page groups the files by directory and shows, for each snapshot, its cells
// in their foreground, background, and attributes, the colors it uses with
// their legend names, and the raw file. Cells with a transparent background
// show the -bg color. Directories whose names begin with a dot are skipped.
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
)

//go:embed page.html
var pageHTML string

var pageTemplate = template.Must(template.New("page").Parse(pageHTML))

// Page is the data the page template draws.
type Page struct {
	// Bg is the color of transparent cells.
	Bg template.CSS
	// Count is the number of files.
	Count int
	// Groups holds the files, one group per testdata directory.
	Groups []Group
}

// Group is the golden files of one directory.
type Group struct {
	Dir   string
	Files []File
}

// File is one golden file.
type File struct {
	// ID is the file's anchor in the page.
	ID   string
	Name string
	// Meta summarizes the file: its size, or its step count for a sequence.
	Meta string
	// Steps holds the file's snapshots; a frames file has one, unnamed.
	Steps []Step
	// Raw is the file's contents.
	Raw string
}

// Step is one snapshot drawn as rows of styled runs.
type Step struct {
	Name   string
	Size   string
	Rows   [][]Run
	Legend []Swatch
}

// Run is a stretch of a row whose cells share one style.
type Run struct {
	Style template.CSS
	Text  string
}

// Swatch describes one color a snapshot uses.
type Swatch struct {
	// Kind is "fg" or "bg".
	Kind string
	Hex  string
	// Name is the color's legend name, if any.
	Name   string
	Swatch template.CSS
}

func main() {
	// A flag set of its own keeps out the -update flag that testutil
	// registers on flag.CommandLine for tests.
	flags := flag.NewFlagSet("goldenview", flag.ExitOnError)
	out := flags.String("o", "goldens.html", "write the page to `file`")
	bg := flags.String("bg", colorHex(style.DarculaSolid().Bg), "show transparent cells on `color`")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "usage: goldenview [-o file] [-bg color] [dir ...]\n")
		flags.PrintDefaults()
	}
	flags.Parse(os.Args[1:])
	dirs := flags.Args()
	if len(dirs) == 0 {
		dirs = []string{"."}
	}
	if err := run(*out, style.ColorFromHex(*bg), dirs); err != nil {
		fmt.Fprintln(os.Stderr, "goldenview:", err)
		os.Exit(1)
	}
}

// run writes the page for the golden files under dirs to out.
func run(out string, bg style.Color, dirs []string) error {
	page := Page{Bg: template.CSS(colorHex(bg))}
	for _, dir := range dirs {
		paths, err := goldenPaths(dir)
		if err != nil {
			return err
		}
		for _, path := range paths {
			file, err := readFile(path, page.Count+1)
			if err != nil {
				return err
			}
			page.add(filepath.Dir(path), file)
		}
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	if err := pageTemplate.Execute(f, page); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "goldenview: wrote %d files to %s\n", page.Count, out)
	return nil
}

// add appends file to the group for dir, starting a group for a new dir.
func (p *Page) add(dir string, file File) {
	p.Count++
	if n := len(p.Groups); n > 0 && p.Groups[n-1].Dir == dir {
		p.Groups[n-1].Files = append(p.Groups[n-1].Files, file)
		return
	}
	p.Groups = append(p.Groups, Group{Dir: dir, Files: []File{file}})
}

// goldenPaths returns the golden files in testdata directories under root,
// in lexical order.
func goldenPaths(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Base(filepath.Dir(path)) == "testdata" && isGolden(d.Name()) {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

// isGolden reports whether name is a frames or sequence golden file.
func isGolden(name string) bool {
	return strings.HasSuffix(name, ".frames.golden") || strings.HasSuffix(name, ".seq.golden")
}

// readFile reads and parses the golden file at path, the n-th on the page.
func readFile(path string, n int) (File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return File{}, err
	}
	file := File{ID: fmt.Sprintf("f%d", n), Name: filepath.Base(path), Raw: string(data)}
	snaps, err := parse(path, file.Raw)
	if err != nil {
		return File{}, fmt.Errorf("%s: %w", path, err)
	}
	for _, s := range snaps {
		file.Steps = append(file.Steps, stepView(s))
	}
	if strings.HasSuffix(path, ".seq.golden") {
		file.Meta = fmt.Sprintf("sequence · %d steps", len(snaps))
	} else {
		file.Meta = file.Steps[0].Size
	}
	return file, nil
}

// parse reads doc as a sequence or a single frames snapshot, by path's suffix.
func parse(path, doc string) ([]testutil.Snapshot, error) {
	if strings.HasSuffix(path, ".seq.golden") {
		return testutil.ParseSequence(doc)
	}
	s, err := testutil.ParseFrames(doc)
	if err != nil {
		return nil, err
	}
	return []testutil.Snapshot{s}, nil
}

// stepView draws s as rows of styled runs and lists the colors it uses.
func stepView(s testutil.Snapshot) Step {
	buf := s.Buf
	step := Step{Name: s.Name, Size: fmt.Sprintf("%d×%d", buf.Width(), buf.Height())}
	var fgs, bgs []style.Color
	for y := range buf.Height() {
		var row []Run
		for x := range buf.Width() {
			c := buf.Cell(x, y)
			if c.IsContinuation() {
				continue
			}
			if shows(c) && !slices.Contains(fgs, c.Fg) {
				fgs = append(fgs, c.Fg)
			}
			if !c.Bg.IsTransparent() && !slices.Contains(bgs, c.Bg) {
				bgs = append(bgs, c.Bg)
			}
			row = appendCell(row, c)
		}
		step.Rows = append(step.Rows, row)
	}
	step.Legend = append(swatches("fg", fgs, s.Names), swatches("bg", bgs, s.Names)...)
	return step
}

// appendCell adds c to row, extending the last run when the styles match.
func appendCell(row []Run, c buffer.Cell) []Run {
	text := string(c.Char)
	if c.Char == 0 {
		text = " "
	}
	css := cellStyle(c)
	if n := len(row); n > 0 && row[n-1].Style == css {
		row[n-1].Text += text
		return row
	}
	return append(row, Run{Style: css, Text: text})
}

// shows reports whether c shows a character, and so has a visible foreground.
func shows(c buffer.Cell) bool {
	return c.Char != 0 && c.Char != ' '
}

// cellStyle returns the CSS that draws c.
func cellStyle(c buffer.Cell) template.CSS {
	var rules []string
	if shows(c) && !c.Fg.IsTransparent() {
		rules = append(rules, "color:"+colorHex(c.Fg))
	}
	if !c.Bg.IsTransparent() {
		rules = append(rules, "background:"+colorHex(c.Bg))
	}
	attrs := c.Attributes
	for _, r := range []struct {
		attr style.Attrs
		css  string
	}{
		{style.AttrBold, "font-weight:700"},
		{style.AttrItalic, "font-style:italic"},
		{style.AttrDim, "opacity:.55"},
	} {
		if attrs&r.attr != 0 {
			rules = append(rules, r.css)
		}
	}
	var deco []string
	if attrs&style.AttrUnderline != 0 {
		deco = append(deco, "underline")
	}
	if attrs&style.AttrStrikethrough != 0 {
		deco = append(deco, "line-through")
	}
	if len(deco) > 0 {
		rules = append(rules, "text-decoration:"+strings.Join(deco, " "))
	}
	return template.CSS(strings.Join(rules, ";"))
}

// swatches lists colors of kind with their names.
func swatches(kind string, colors []style.Color, names map[style.Color]string) []Swatch {
	var out []Swatch
	for _, c := range colors {
		sw := Swatch{Kind: kind, Hex: "-", Name: names[c]}
		if !c.IsTransparent() {
			sw.Hex = colorHex(c)
			sw.Swatch = template.CSS("background:" + sw.Hex)
		}
		out = append(out, sw)
	}
	return out
}

// colorHex returns c as #rrggbb, ignoring alpha.
func colorHex(c style.Color) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}
