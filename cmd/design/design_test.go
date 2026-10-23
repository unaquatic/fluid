package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/testutil"
)

// TestDesignRoundTripsGoldens checks that every golden file in the
// repository reads into a design that, after a trip through JSON, writes the
// same document again, so that the editor saves exactly what tests assert.
func TestDesignRoundTripsGoldens(t *testing.T) {
	root, err := os.OpenRoot("../..")
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	lib := Library{Root: root}
	n := 0
	for p, err := range lib.Paths() {
		if err != nil {
			t.Fatal(err)
		}
		t.Run(p, func(t *testing.T) {
			want, err := root.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			snaps, err := lib.Load(p)
			if err != nil {
				t.Fatal(err)
			}
			var got strings.Builder
			for _, s := range snaps {
				if s.Name != "" {
					got.WriteString("=== step: " + s.Name + " ===\n")
				}
				doc, err := viaJSON(t, NewDesign(s.Buf, s.Names)).Frames()
				if err != nil {
					t.Fatal(err)
				}
				got.WriteString(doc)
			}
			if got.String() != string(want) {
				t.Errorf("frames differ from the file\n--- got ---\n%s--- want ---\n%s", got.String(), want)
			}
		})
		n++
	}
	if n == 0 {
		t.Fatal("found no golden files")
	}
}

// viaJSON returns d after encoding it as JSON and decoding it again.
func viaJSON(t *testing.T, d Design) Design {
	t.Helper()
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var out Design
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// row returns a one-row design of cells.
func row(cells ...Cell) Design {
	return Design{Width: len(cells), Height: 1, Cells: [][]Cell{cells}}
}

func TestDesignFrames(t *testing.T) {
	d := row(
		Cell{Ch: "o", Fg: "#ff0000", Bg: "#101010", Attrs: "B"},
		Cell{Ch: "k", Fg: "#00ff00", Bg: "#101010", Attrs: "BU"},
		Cell{Ch: " ", Fg: "#0000ff"},
		Cell{Ch: "中", Fg: "#ff0000"},
		Cell{Ch: ""},
		// A right half with no wide character to its left reads as a space.
		Cell{Ch: "", Bg: "#202020"},
	)
	d.Names = map[string]string{"#101010": "Surface", "#00ff00": "Accent", "#999999": "Unused"}
	got, err := d.Frames()
	if err != nil {
		t.Fatal(err)
	}
	// The text frame holds a rune per cell, so the right half of 中 is a space.
	want := `=== text ===
|ok 中  |
=== fg ===
|AB A  |
=== bg ===
|░░   ▒|
=== attrs ===
|B1    |
=== legend ===
A #ff0000
B #00ff00 Accent
░ #101010 Surface
▒ #202020
1 BU
`
	if got != want {
		t.Fatalf("frames =\n%s\nwant\n%s", got, want)
	}
	s, err := testutil.ParseFrames(got)
	if err != nil {
		t.Fatal(err)
	}
	back := NewDesign(s.Buf, s.Names)
	if c := back.Cells[0][3]; c.Ch != "中" || back.Cells[0][4].Ch != "" || back.Cells[0][5].Ch != " " {
		t.Errorf("read back cells 3–5 as %+v, %+v, %+v; want the wide character, its right half, and a space",
			c, back.Cells[0][4], back.Cells[0][5])
	}
	if back.Cells[0][2].Fg != "" {
		t.Errorf("blank cell has foreground %q, want none", back.Cells[0][2].Fg)
	}
}

func TestDesignErrors(t *testing.T) {
	bgs := make([]Cell, 5)
	for i := range bgs {
		bgs[i] = Cell{Ch: " ", Bg: "#00000" + string(rune('1'+i))}
	}
	tests := []struct {
		name string
		d    Design
		want string
	}{
		{"empty", Design{}, "size 0×0"},
		{"too big", Design{Width: MaxSide + 1, Height: 1}, "size"},
		{"short rows", Design{Width: 2, Height: 1, Cells: [][]Cell{{{Ch: "a"}}}}, "row 0 is 1 cells wide, not 2"},
		{"missing rows", Design{Width: 1, Height: 2, Cells: [][]Cell{{{Ch: "a"}}}}, "1 rows, not 2"},
		{"bad color", row(Cell{Ch: "a", Fg: "red"}), `color "red" is not #rrggbb`},
		{"short color", row(Cell{Ch: "a", Bg: "#fff"}), `color "#fff" is not #rrggbb`},
		{"two runes", row(Cell{Ch: "ab", Fg: "#ffffff"}), "not one code point"},
		{"newline", row(Cell{Ch: "\n", Fg: "#ffffff"}), "takes up no column"},
		{"combining mark", row(Cell{Ch: "\u0301", Fg: "#ffffff"}), "takes up no column"},
		{"attribute", row(Cell{Ch: "a", Fg: "#ffffff", Attrs: "X"}), `unknown attribute 'X'`},
		{"name color", Design{Width: 1, Height: 1, Cells: [][]Cell{{{Ch: " "}}}, Names: map[string]string{"x": "X"}}, "is not #rrggbb"},
		{"name newline", Design{Width: 1, Height: 1, Cells: [][]Cell{{{Ch: " "}}}, Names: map[string]string{"#ffffff": "a\nb"}}, "control character"},
		{"five backgrounds", row(bgs...), "more than 4 distinct values"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.d.Frames()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Frames() error = %v, want one holding %q", err, tt.want)
			}
		})
	}
}

func TestDesignANSI(t *testing.T) {
	d := Design{Width: 6, Height: 2, Cells: [][]Cell{
		{
			{Ch: "a", Fg: "#ff0000"},
			{Ch: " "},
			{Ch: "b", Fg: "#ff0000", Attrs: "B"},
			{Ch: " ", Bg: "#000080"},
			{Ch: " "},
			{Ch: " "},
		},
		{{Ch: " "}, {Ch: " "}, {Ch: " "}, {Ch: " "}, {Ch: " "}, {Ch: " "}},
	}}
	got, err := d.ANSI()
	if err != nil {
		t.Fatal(err)
	}
	want := "\x1b[0;38;2;255;0;0ma \x1b[0;1;38;2;255;0;0mb\x1b[0;38;2;255;0;0;48;2;0;0;128m \x1b[0m\n\n"
	if got != want {
		t.Errorf("ANSI() = %q\nwant      %q", got, want)
	}
}
