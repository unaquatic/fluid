package widget

import (
	"strings"
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
)

func makeSelectItems() []SelectItem {
	return []SelectItem{
		{Name: "Option 1", Description: "First option", Value: 1},
		{Name: "Option 2", Description: "Second option", Value: 2},
		{Name: "Option 3", Description: "Third option", Value: 3},
	}
}

func setupSelect() *Select {
	s := NewSelect(makeSelectItems())
	testutil.SetupWidget(s, 30, 10)
	return s
}

func TestNewSelect_Defaults(t *testing.T) {
	s := NewSelect(makeSelectItems())
	if s.SelectedIndex() != 0 {
		t.Errorf("expected selectedIndex 0, got %d", s.SelectedIndex())
	}
	if len(s.Items()) != 3 {
		t.Errorf("expected 3 items, got %d", len(s.Items()))
	}
	item := s.SelectedItem()
	if item == nil || item.Name != "Option 1" {
		t.Error("expected first item selected")
	}
}

func TestSelect_MoveDown(t *testing.T) {
	s := setupSelect()
	s.Focus()

	s.HandleKeyPress(&input.KeyEvent{Name: input.KeyDown, Type: input.KeyPress})
	if s.SelectedIndex() != 1 {
		t.Errorf("expected selectedIndex 1, got %d", s.SelectedIndex())
	}

	s.HandleKeyPress(&input.KeyEvent{Name: "j", Type: input.KeyPress})
	if s.SelectedIndex() != 2 {
		t.Errorf("expected selectedIndex 2, got %d", s.SelectedIndex())
	}
}

func TestSelect_MoveUp(t *testing.T) {
	s := setupSelect()
	s.SetSelectedIndex(2)

	s.HandleKeyPress(&input.KeyEvent{Name: input.KeyUp, Type: input.KeyPress})
	if s.SelectedIndex() != 1 {
		t.Errorf("expected selectedIndex 1, got %d", s.SelectedIndex())
	}

	s.HandleKeyPress(&input.KeyEvent{Name: "k", Type: input.KeyPress})
	if s.SelectedIndex() != 0 {
		t.Errorf("expected selectedIndex 0, got %d", s.SelectedIndex())
	}
}

func TestSelect_WrapSelection(t *testing.T) {
	s := setupSelect()
	// Up wraps from the first enabled option to the last.
	s.HandleKeyPress(&input.KeyEvent{Name: input.KeyUp, Type: input.KeyPress})
	if s.SelectedIndex() != 2 {
		t.Errorf("expected wrap to 2, got %d", s.SelectedIndex())
	}

	// Down wraps from the last enabled option to the first.
	s.HandleKeyPress(&input.KeyEvent{Name: input.KeyDown, Type: input.KeyPress})
	if s.SelectedIndex() != 0 {
		t.Errorf("expected wrap to 0, got %d", s.SelectedIndex())
	}
}

func TestSelect_HomeEnd(t *testing.T) {
	s := setupSelect()
	s.SetSelectedIndex(1)

	s.HandleKeyPress(&input.KeyEvent{Name: input.KeyHome, Type: input.KeyPress})
	if s.SelectedIndex() != 0 {
		t.Errorf("expected selectedIndex 0 (Home), got %d", s.SelectedIndex())
	}

	s.HandleKeyPress(&input.KeyEvent{Name: input.KeyEnd, Type: input.KeyPress})
	if s.SelectedIndex() != 2 {
		t.Errorf("expected selectedIndex 2 (End), got %d", s.SelectedIndex())
	}
}

func TestSelect_OnItemSelected(t *testing.T) {
	var selectedIdx int
	var selectedItem SelectItem
	s := NewSelect(makeSelectItems(), SelectWithOnItemSelected(func(idx int, item SelectItem) {
		selectedIdx = idx
		selectedItem = item
	}))
	testutil.SetupWidget(s, 30, 10)

	s.HandleKeyPress(&input.KeyEvent{Name: input.KeyReturn, Type: input.KeyPress})
	if selectedIdx != 0 || selectedItem.Name != "Option 1" {
		t.Error("expected first option to be selected via Enter")
	}
}

func TestSelect_SetOptions(t *testing.T) {
	s := setupSelect()
	s.SetSelectedIndex(2)

	newOpts := []SelectItem{
		{Name: "New Option 1", Value: "a"},
	}
	s.SetItems(newOpts)
	if s.SelectedIndex() != 0 {
		t.Errorf("expected selectedIndex clamped to 0, got %d", s.SelectedIndex())
	}
}

func TestSelect_Render(t *testing.T) {
	s := setupSelect()
	s.Focus()

	buf := buffer.New(30, 10)
	s.RenderSelf(buf, 0)

	// The first option must produce visible text.
	text := buf.String()
	if len(text) == 0 {
		t.Error("expected non-empty render output")
	}
}

// TestSelect_HintColor checks that a hint color colors the hint after each
// name on every row, the selected one included, and that without one the
// selected row's hint takes the row's text color.
func TestSelect_HintColor(t *testing.T) {
	hintFg := style.ColorFromInts(112, 112, 112, 255)
	textFg := style.ColorFromInts(180, 180, 180, 255)
	selectedFg := style.ColorWhite
	items := []SelectItem{{Name: "alpha", Hint: "[default]"}, {Name: "beta", Hint: "[new]"}}
	// The hint starts after the name and a space.
	hintAt := func(buf *buffer.Buffer, row int, name string) style.Color {
		return buf.Cell(len(name)+1, row).Fg
	}
	tests := []struct {
		name             string
		opts             []SelectOption
		selected, normal style.Color
	}{
		{name: "hint color", opts: []SelectOption{SelectWithHintColor(hintFg)}, selected: hintFg, normal: hintFg},
		{name: "description color", selected: selectedFg, normal: textFg},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSelect(items, append([]SelectOption{
				SelectWithTextColor(textFg),
				SelectWithDescriptionColor(textFg),
				SelectWithFocusedTextColor(selectedFg),
				SelectWithSelectedTextColor(selectedFg),
			}, tt.opts...)...)
			s.Focus()
			s.SetBounds(0, 0, 20, 2)
			buf := buffer.New(20, 2)
			s.RenderSelf(buf, 0)
			if got := hintAt(buf, 0, "alpha"); got != tt.selected {
				t.Errorf("selected row's hint fg = %v, want %v", got, tt.selected)
			}
			if got := hintAt(buf, 1, "beta"); got != tt.normal {
				t.Errorf("other row's hint fg = %v, want %v", got, tt.normal)
			}
		})
	}
}

func TestSelect_ArrowColumn(t *testing.T) {
	items := []SelectItem{
		{Name: "alpha", Description: "first"},
		{Name: "beta", Hint: "hint"},
	}
	tests := []struct {
		name      string
		opts      []SelectOption
		nameWidth int
		want      []string
	}{
		{
			name: "arrow covers the name without a column",
			want: []string{"→lpha first", "beta hint"},
		},
		{
			name: "names after the column",
			opts: []SelectOption{SelectWithArrowColumn(true)},
			want: []string{"→ alpha first", "  beta hint"},
		},
		{
			name:      "name width set later",
			opts:      []SelectOption{SelectWithArrowColumn(true)},
			nameWidth: 8,
			want:      []string{"→ alpha   first", "  beta hint"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSelect(items, append([]SelectOption{
				SelectWithSelectionArrow(true),
				SelectWithInlineDescription(true),
				SelectWithNameWidth(6),
			}, tt.opts...)...)
			if tt.nameWidth > 0 {
				s.SetNameWidth(tt.nameWidth)
			}
			testutil.SetupWidget(s, 20, 2)
			buf := buffer.New(20, 2)
			s.RenderSelf(buf, 0)

			for i, line := range strings.Split(buf.String(), "\n") {
				if got := strings.TrimRight(line, " "); got != tt.want[i] {
					t.Errorf("row %d = %q, want %q", i, got, tt.want[i])
				}
			}
		})
	}
}

func TestSelect_Options(t *testing.T) {
	s := NewSelect(makeSelectItems(),
		SelectWithID("my-select"),
		SelectWithBackgroundColor(style.ColorBlack),
		SelectWithTextColor(style.ColorWhite),
		SelectWithFocusedBackgroundColor(style.ColorBlue),
		SelectWithShowScrollIndicator(false),
		SelectWithWrapSelection(false),
		SelectWithShowDescription(false),
		SelectWithWidth(layout.Points(20)),
		SelectWithHeight(layout.Points(5)),
	)

	if s.ID() != "my-select" {
		t.Errorf("expected id 'my-select', got %q", s.ID())
	}
	if s.showScrollIndicator {
		t.Error("expected showScrollIndicator false")
	}
	if s.wrapSelection {
		t.Error("expected wrapSelection false")
	}
}

func TestSelect_SelectedBackground(t *testing.T) {
	items := []SelectItem{
		{Name: "enter code manually", Value: "manual"},
		{Name: "cancel", Value: "cancel"},
	}
	selectedBg := style.ColorFromInts(51, 153, 255, 255)
	s := NewSelect(items,
		SelectWithBackgroundColor(style.ColorFromInts(30, 30, 30, 255)),
		SelectWithTextColor(style.ColorFromInts(180, 180, 180, 255)),
		SelectWithSelectedBackgroundColor(selectedBg),
		SelectWithSelectedTextColor(style.ColorWhite),
		SelectWithWrapSelection(true),
		SelectWithShowScrollIndicator(false),
		SelectWithShowDescription(false),
		SelectWithItemPaddingX(1),
	)
	testutil.SetupWidget(s, 30, 2)

	buf := buffer.New(30, 2)
	s.RenderSelf(buf, 0)

	text := buf.String()
	t.Logf("rendered text:\n%s", text)

	// The selected first row receives the configured blue background.
	cell0 := buf.Cell(1, 0)
	t.Logf("cell at (1,0): char=%c (%U), bg=%v", cell0.Char, cell0.Char, cell0.Bg)
	if cell0.Bg != selectedBg {
		t.Errorf("expected selected bg %v at (1,0), got %v", selectedBg, cell0.Bg)
	}

	// The unselected second row retains its normal background.
	cell1 := buf.Cell(1, 1)
	t.Logf("cell at (1,1): char=%c (%U), bg=%v", cell1.Char, cell1.Char, cell1.Bg)
	if cell1.Bg == selectedBg {
		t.Error("second row should not have selected background")
	}
}
