package widget

import (
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
)

func makeSelectOptions() []SelectOption {
	return []SelectOption{
		{Name: "Option 1", Description: "First option", Value: 1},
		{Name: "Option 2", Description: "Second option", Value: 2},
		{Name: "Option 3", Description: "Third option", Value: 3},
	}
}

func setupSelect() *Select {
	s := NewSelect(makeSelectOptions())
	testutil.SetupWidget(s, 30, 10)
	return s
}

func TestNewSelect_Defaults(t *testing.T) {
	s := NewSelect(makeSelectOptions())
	if s.SelectedIndex() != 0 {
		t.Errorf("expected selectedIndex 0, got %d", s.SelectedIndex())
	}
	if len(s.Options()) != 3 {
		t.Errorf("expected 3 options, got %d", len(s.Options()))
	}
	opt := s.SelectedOption()
	if opt == nil || opt.Name != "Option 1" {
		t.Error("expected first option selected")
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
	// Wrap from top to bottom
	s.HandleKeyPress(&input.KeyEvent{Name: input.KeyUp, Type: input.KeyPress})
	if s.SelectedIndex() != 2 {
		t.Errorf("expected wrap to 2, got %d", s.SelectedIndex())
	}

	// Wrap from bottom to top
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
	var selectedOpt SelectOption
	s := NewSelect(makeSelectOptions(), SelectWithOnItemSelected(func(idx int, opt SelectOption) {
		selectedIdx = idx
		selectedOpt = opt
	}))
	testutil.SetupWidget(s, 30, 10)

	s.HandleKeyPress(&input.KeyEvent{Name: input.KeyReturn, Type: input.KeyPress})
	if selectedIdx != 0 || selectedOpt.Name != "Option 1" {
		t.Error("expected first option to be selected via Enter")
	}
}

func TestSelect_SetOptions(t *testing.T) {
	s := setupSelect()
	s.SetSelectedIndex(2)

	newOpts := []SelectOption{
		{Name: "New Option 1", Value: "a"},
	}
	s.SetOptions(newOpts)
	if s.SelectedIndex() != 0 {
		t.Errorf("expected selectedIndex clamped to 0, got %d", s.SelectedIndex())
	}
}

func TestSelect_Render(t *testing.T) {
	s := setupSelect()
	s.Focus()

	buf := buffer.New(30, 10)
	s.RenderSelf(buf, 0)

	// Verify something was rendered
	text := buf.PlainText(true)
	if len(text) == 0 {
		t.Error("expected non-empty render output")
	}
}

func TestSelect_Options(t *testing.T) {
	s := NewSelect(makeSelectOptions(),
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
	opts := []SelectOption{
		{Name: "enter code manually", Value: "manual"},
		{Name: "cancel", Value: "cancel"},
	}
	selectedBg := style.ColorFromInts(51, 153, 255, 255)
	s := NewSelect(opts,
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

	text := buf.PlainText(true)
	t.Logf("rendered text:\n%s", text)

	// First row (selected) should have the blue background.
	cell0 := buf.Cell(1, 0)
	t.Logf("cell at (1,0): char=%c (%U), bg=%v", cell0.Char, cell0.Char, cell0.Bg)
	if cell0.Bg != selectedBg {
		t.Errorf("expected selected bg %v at (1,0), got %v", selectedBg, cell0.Bg)
	}

	// Second row (unselected) should NOT have the blue background.
	cell1 := buf.Cell(1, 1)
	t.Logf("cell at (1,1): char=%c (%U), bg=%v", cell1.Char, cell1.Char, cell1.Bg)
	if cell1.Bg == selectedBg {
		t.Error("second row should not have selected background")
	}
}
