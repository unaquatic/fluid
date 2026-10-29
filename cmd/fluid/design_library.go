package main

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"iter"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
)

// Golden file suffixes.
const (
	framesSuffix   = ".frames.golden"
	sequenceSuffix = ".seq.golden"
)

// errBadPath reports a path that names no golden frames file inside the
// library.
var errBadPath = errors.New("path must be relative, stay inside the root, and end in " + framesSuffix)

// library is the golden files under one directory, which the editor opens
// and saves designs to. It reaches no file outside the directory.
type library struct {
	root *os.Root
}

// paths yields the slash-separated path, relative to the root, of every
// golden frames or sequence file under it, in lexical order. Directories
// whose names begin with a dot are skipped. A walk error ends the sequence.
func (l library) paths() iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		err := fs.WalkDir(l.root.FS(), ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if p != "." && strings.HasPrefix(d.Name(), ".") {
					return fs.SkipDir
				}
				return nil
			}
			if isGolden(p) && !yield(p, nil) {
				return fs.SkipAll
			}
			return nil
		})
		if err != nil {
			yield("", err)
		}
	}
}

// isGolden reports whether p names a golden frames or sequence file.
func isGolden(p string) bool {
	return strings.HasSuffix(p, framesSuffix) || strings.HasSuffix(p, sequenceSuffix)
}

// load reads the snapshots of the golden file at p.
func (l library) load(p string) ([]testutil.Snapshot, error) {
	if !fs.ValidPath(p) || !isGolden(p) {
		return nil, fmt.Errorf("%s: not a golden file in the library", p)
	}
	data, err := l.root.ReadFile(p)
	if err != nil {
		return nil, err
	}
	snaps, err := parseDoc(string(data))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", p, err)
	}
	return snaps, nil
}

// save writes doc to the golden frames file at p, making its directories.
// Unless overwrite is set, it fails with an error matching fs.ErrExist when
// the file is there already.
func (l library) save(p, doc string, overwrite bool) error {
	if !fs.ValidPath(p) || !strings.HasSuffix(p, framesSuffix) || path.Base(p) == framesSuffix {
		return errBadPath
	}
	if err := l.root.MkdirAll(path.Dir(p), 0o755); err != nil {
		return err
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if overwrite {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	f, err := l.root.OpenFile(p, flags, 0o644)
	if err != nil {
		return err
	}
	_, err = f.WriteString(doc)
	return errors.Join(err, f.Close())
}

// fileEntry describes a golden file that the editor can open.
type fileEntry struct {
	Path string `json:"path"`
	// Size is the width×height of a frames file's snapshot.
	Size string `json:"size,omitempty"`
	// Steps names the steps of a sequence file, in order.
	Steps []string `json:"steps,omitempty"`
	// Error says why the file does not parse.
	Error string `json:"error,omitempty"`
}

// usedColor is a color that golden files use, with the number of snapshots
// that use it for characters and for backgrounds.
type usedColor struct {
	Hex string `json:"hex"`
	// Name is a name the color has in a legend.
	Name string `json:"name,omitempty"`
	Fg   int    `json:"fg"`
	Bg   int    `json:"bg"`
}

// catalog returns an entry for every golden file in the library and the
// colors their snapshots use, the most used first.
func (l library) catalog() ([]fileEntry, []usedColor, error) {
	var files []fileEntry
	used := make(map[style.Color]*usedColor)
	for p, err := range l.paths() {
		if err != nil {
			return nil, nil, err
		}
		entry := fileEntry{Path: p}
		snaps, err := l.load(p)
		if err != nil {
			entry.Error = err.Error()
		}
		for _, s := range snaps {
			if strings.HasSuffix(p, sequenceSuffix) {
				entry.Steps = append(entry.Steps, s.Name)
			} else {
				entry.Size = fmt.Sprintf("%d×%d", s.Buf.Width(), s.Buf.Height())
			}
			countColors(used, s)
		}
		files = append(files, entry)
	}
	colors := make([]usedColor, 0, len(used))
	for _, u := range used {
		colors = append(colors, *u)
	}
	slices.SortFunc(colors, func(a, b usedColor) int {
		return cmp.Or(cmp.Compare(b.Fg+b.Bg, a.Fg+a.Bg), cmp.Compare(a.Hex, b.Hex))
	})
	return files, colors, nil
}

// countColors adds the colors that s uses once each to used, with their
// legend names.
func countColors(used map[style.Color]*usedColor, s testutil.Snapshot) {
	fgs := make(map[style.Color]bool)
	bgs := make(map[style.Color]bool)
	buf := s.Buf
	for y := range buf.Height() {
		for x := range buf.Width() {
			c := buf.Cell(x, y)
			if !c.IsEmpty() && !c.Fg.IsTransparent() {
				fgs[c.Fg] = true
			}
			if !c.Bg.IsTransparent() {
				bgs[c.Bg] = true
			}
		}
	}
	entry := func(c style.Color) *usedColor {
		u := used[c]
		if u == nil {
			u = &usedColor{Hex: formatHex(c)}
			used[c] = u
		}
		u.Name = cmp.Or(u.Name, s.Names[c])
		return u
	}
	for c := range fgs {
		entry(c).Fg++
	}
	for c := range bgs {
		entry(c).Bg++
	}
}

// swatch is a color the editor offers, with its name.
type swatch struct {
	Name string `json:"name"`
	Hex  string `json:"hex"`
}

// swatchGroup is a titled set of swatches.
type swatchGroup struct {
	Title  string   `json:"title"`
	Colors []swatch `json:"colors"`
}

// themePalette returns the colors of th by role, each resolved as widgets
// resolve it, in groups. Roles that resolve to a transparent color are left
// out.
func themePalette(th *style.Theme) []swatchGroup {
	type role struct {
		name  string
		color style.Color
	}
	groups := []struct {
		title string
		roles []role
	}{
		{"Surfaces", []role{
			{"Bg", th.Bg},
			{"Surface", th.Surface},
			{"SurfaceAlt", th.SurfaceAlt},
			{"SurfaceRaised", th.ResolveSurfaceRaised()},
			{"SelectionBg", th.ResolveSelectionBg()},
			{"SelectionInactiveBg", th.ResolveSelectionInactiveBg()},
			{"CodeBg", th.CodeBg},
		}},
		{"Semantic surfaces", []role{
			{"SuccessSurface", th.SuccessSurface},
			{"WarningSurface", th.WarningSurface},
			{"DangerSurface", th.DangerSurface},
			{"InfoSurface", th.InfoSurface},
			{"DiffAddSurface", th.ResolveDiffAddSurface()},
			{"DiffRemoveSurface", th.ResolveDiffRemoveSurface()},
		}},
		{"Text", []role{
			{"TextStrong", th.TextStrong},
			{"TextBase", th.ResolveTextBase()},
			{"TextWeak", th.TextWeak},
			{"TextMuted", th.TextMuted},
			{"TextLink", th.ResolveTextLink()},
			{"CodeFg", th.CodeFg},
			{"SelectionFg", th.ResolveSelectionFg()},
		}},
		{"Accent", []role{
			{"Accent", th.Accent},
			{"AccentStrong", th.AccentStrong},
			{"Interactive", th.ResolveInteractive()},
			{"InteractiveStrong", th.ResolveInteractiveStrong()},
			{"OnAccent", th.ResolveOnAccent()},
		}},
		{"Semantic", []role{
			{"Success", th.Success},
			{"Warning", th.Warning},
			{"Danger", th.Danger},
			{"Info", th.Info},
			{"DiffAddFg", th.ResolveDiffAddFg()},
			{"DiffRemoveFg", th.ResolveDiffRemoveFg()},
		}},
		{"Borders", []role{
			{"Border", th.Border},
			{"BorderStrong", th.BorderStrong},
			{"BorderWeak", th.ResolveBorderWeak()},
			{"BorderFocus", th.ResolveBorderFocus()},
		}},
	}
	var out []swatchGroup
	for _, g := range groups {
		sg := swatchGroup{Title: g.title}
		for _, r := range g.roles {
			if hex := formatHex(r.color); hex != "" {
				sg.Colors = append(sg.Colors, swatch{Name: r.name, Hex: hex})
			}
		}
		if len(sg.Colors) > 0 {
			out = append(out, sg)
		}
	}
	return out
}
