package style

import (
	"testing"
)

func TestNewStyledText(t *testing.T) {
	st := NewStyledText(Plain("hello"), TextBold(" world"))
	if len(st.Chunks) != 2 {
		t.Errorf("should have 2 chunks, got %d", len(st.Chunks))
	}
}

func TestPlainText(t *testing.T) {
	st := NewStyledText(Plain("hello"), TextBold(" world"))
	text := st.PlainText()
	if text != "hello world" {
		t.Errorf("got %q, want %q", text, "hello world")
	}
}

func TestStyledTextLen(t *testing.T) {
	st := NewStyledText(Plain("hello"), TextBold(" world"))
	if st.Len() != 11 {
		t.Errorf("got %d, want 11", st.Len())
	}
}

func TestStyledTextLen_Unicode(t *testing.T) {
	st := NewStyledText(Plain("hello "), Plain("世界"))
	if st.Len() != 8 {
		t.Errorf("got %d, want 8 (6 ascii + 2 CJK runes)", st.Len())
	}
}

func TestStyledTextAppend(t *testing.T) {
	a := NewStyledText(Plain("hello"))
	b := NewStyledText(Plain(" world"))
	result := a.Append(b)
	if result.PlainText() != "hello world" {
		t.Errorf("got %q, want %q", result.PlainText(), "hello world")
	}
	if len(result.Chunks) != 2 {
		t.Errorf("should have 2 chunks, got %d", len(result.Chunks))
	}
}

func TestStyledTextString(t *testing.T) {
	st := NewStyledText(Plain("test"))
	if st.String() != "test" {
		t.Errorf("got %q, want %q", st.String(), "test")
	}
}

func TestStyledTextEmpty(t *testing.T) {
	st := NewStyledText()
	if st.PlainText() != "" {
		t.Errorf("empty styled text should have empty plain text")
	}
	if st.Len() != 0 {
		t.Errorf("empty styled text should have len 0")
	}
}

func TestPlainChunk(t *testing.T) {
	c := Plain("hello")
	if c.Text != "hello" {
		t.Errorf("text should be 'hello', got %q", c.Text)
	}
	if c.Fg != nil || c.Bg != nil {
		t.Error("plain chunk should have nil Fg and Bg")
	}
	if c.Attributes != AttrNone {
		t.Errorf("plain chunk should have no attributes, got %d", c.Attributes)
	}
}

func TestBoldChunk(t *testing.T) {
	c := TextBold("bold")
	if c.Attributes != AttrBold {
		t.Errorf("bold chunk should have AttrBold, got %d", c.Attributes)
	}
}

func TestItalicChunk(t *testing.T) {
	c := TextItalic("italic")
	if c.Attributes != AttrItalic {
		t.Errorf("italic chunk should have AttrItalic, got %d", c.Attributes)
	}
}

func TestUnderlineChunk(t *testing.T) {
	c := TextUnderline("underline")
	if c.Attributes != AttrUnderline {
		t.Errorf("underline chunk should have AttrUnderline, got %d", c.Attributes)
	}
}

func TestDimChunk(t *testing.T) {
	c := TextDim("dim")
	if c.Attributes != AttrDim {
		t.Errorf("dim chunk should have AttrDim, got %d", c.Attributes)
	}
}

func TestStrikethroughChunk(t *testing.T) {
	c := TextStrikethrough("strike")
	if c.Attributes != AttrStrikethrough {
		t.Errorf("strikethrough chunk should have AttrStrikethrough, got %d", c.Attributes)
	}
}

func TestInverseChunk(t *testing.T) {
	c := TextInverse("inverse")
	if c.Attributes != AttrInverse {
		t.Errorf("inverse chunk should have AttrInverse, got %d", c.Attributes)
	}
}

func TestFgFunction(t *testing.T) {
	redText := Fg(ColorString("#ff0000"))
	chunk := redText("hello")
	if chunk.Text != "hello" {
		t.Errorf("text should be 'hello', got %q", chunk.Text)
	}
	if chunk.Fg == nil {
		t.Fatal("Fg should not be nil")
	}
	if !chunk.Fg.Equals(ColorRed) {
		t.Errorf("fg should be ColorRed, got %s", chunk.Fg)
	}
}

func TestFgFunction_RGBA(t *testing.T) {
	greenText := Fg(ColorGreen)
	chunk := greenText("world")
	if chunk.Fg == nil {
		t.Fatal("Fg should not be nil")
	}
	if !chunk.Fg.Equals(ColorGreen) {
		t.Errorf("fg should be ColorGreen, got %s", chunk.Fg)
	}
}

func TestBgFunction(t *testing.T) {
	blueBg := Bg(ColorString("#0000ff"))
	chunk := blueBg("hello")
	if chunk.Bg == nil {
		t.Fatal("Bg should not be nil")
	}
	if !chunk.Bg.Equals(ColorBlue) {
		t.Errorf("bg should be ColorBlue, got %s", chunk.Bg)
	}
}

func TestLinkFunction(t *testing.T) {
	linkFn := Link("https://example.com")
	chunk := linkFn("click me")
	if chunk.Text != "click me" {
		t.Errorf("text should be 'click me', got %q", chunk.Text)
	}
	if chunk.Link != "https://example.com" {
		t.Errorf("link should be 'https://example.com', got %q", chunk.Link)
	}
	// Link applies the default underline decoration.
	if !HasAttribute(chunk.Attributes, AttrUnderline) {
		t.Error("link chunk should have underline attribute")
	}
}

func TestStyledChunk(t *testing.T) {
	s := StyleFrom(ColorRed, ColorBlue, AttrBold)
	chunk := Styled("styled", s)
	if chunk.Text != "styled" {
		t.Errorf("text should be 'styled', got %q", chunk.Text)
	}
	if chunk.Fg == nil || !chunk.Fg.Equals(ColorRed) {
		t.Error("fg should be ColorRed")
	}
	if chunk.Bg == nil || !chunk.Bg.Equals(ColorBlue) {
		t.Error("bg should be ColorBlue")
	}
	if chunk.Attributes != AttrBold {
		t.Errorf("attrs should be Bold, got %d", chunk.Attributes)
	}
}

func TestFgBlack(t *testing.T) {
	c := FgBlack("text")
	if c.Fg == nil || !c.Fg.Equals(ColorBlack) {
		t.Error("should have ColorBlack fg")
	}
}

func TestFgRed(t *testing.T) {
	c := FgRed("text")
	if c.Fg == nil || !c.Fg.Equals(ColorRed) {
		t.Error("should have ColorRed fg")
	}
}

func TestFgGreen(t *testing.T) {
	c := FgGreen("text")
	if c.Fg == nil || !c.Fg.Equals(ColorGreen) {
		t.Error("should have ColorGreen fg")
	}
}

func TestFgYellow(t *testing.T) {
	c := FgYellow("text")
	if c.Fg == nil || !c.Fg.Equals(ColorYellow) {
		t.Error("should have ColorYellow fg")
	}
}

func TestFgBlue(t *testing.T) {
	c := FgBlue("text")
	if c.Fg == nil || !c.Fg.Equals(ColorBlue) {
		t.Error("should have ColorBlue fg")
	}
}

func TestFgMagenta(t *testing.T) {
	c := FgMagenta("text")
	if c.Fg == nil || !c.Fg.Equals(ColorMagenta) {
		t.Error("should have Magenta fg")
	}
}

func TestFgCyan(t *testing.T) {
	c := FgCyan("text")
	if c.Fg == nil || !c.Fg.Equals(ColorCyan) {
		t.Error("should have Cyan fg")
	}
}

func TestFgWhite(t *testing.T) {
	c := FgWhite("text")
	if c.Fg == nil || !c.Fg.Equals(ColorWhite) {
		t.Error("should have ColorWhite fg")
	}
}

func TestFgBrightRed(t *testing.T) {
	c := FgBrightRed("text")
	if c.Fg == nil || !c.Fg.Equals(ColorBrightRed) {
		t.Error("should have BrightRed fg")
	}
}

func TestFgBrightGreen(t *testing.T) {
	c := FgBrightGreen("text")
	if c.Fg == nil || !c.Fg.Equals(ColorBrightGreen) {
		t.Error("should have BrightGreen fg")
	}
}

func TestBgBlack(t *testing.T) {
	c := BgBlack("text")
	if c.Bg == nil || !c.Bg.Equals(ColorBlack) {
		t.Error("should have ColorBlack bg")
	}
}

func TestBgRed(t *testing.T) {
	c := BgRed("text")
	if c.Bg == nil || !c.Bg.Equals(ColorRed) {
		t.Error("should have ColorRed bg")
	}
}

func TestBgGreen(t *testing.T) {
	c := BgGreen("text")
	if c.Bg == nil || !c.Bg.Equals(ColorGreen) {
		t.Error("should have ColorGreen bg")
	}
}

func TestBgYellow(t *testing.T) {
	c := BgYellow("text")
	if c.Bg == nil || !c.Bg.Equals(ColorYellow) {
		t.Error("should have ColorYellow bg")
	}
}

func TestBgBlue(t *testing.T) {
	c := BgBlue("text")
	if c.Bg == nil || !c.Bg.Equals(ColorBlue) {
		t.Error("should have ColorBlue bg")
	}
}

func TestBgMagenta(t *testing.T) {
	c := BgMagenta("text")
	if c.Bg == nil || !c.Bg.Equals(ColorMagenta) {
		t.Error("should have Magenta bg")
	}
}

func TestBgCyan(t *testing.T) {
	c := BgCyan("text")
	if c.Bg == nil || !c.Bg.Equals(ColorCyan) {
		t.Error("should have Cyan bg")
	}
}

func TestBgWhite(t *testing.T) {
	c := BgWhite("text")
	if c.Bg == nil || !c.Bg.Equals(ColorWhite) {
		t.Error("should have ColorWhite bg")
	}
}

func TestComposedStyledText(t *testing.T) {
	st := NewStyledText(
		FgRed("Error: "),
		TextBold("something went wrong"),
		Plain(" - see "),
		Link("https://docs.example.com")("docs"),
	)

	plainText := st.PlainText()
	want := "Error: something went wrong - see docs"
	if plainText != want {
		t.Errorf("got %q, want %q", plainText, want)
	}

	if len(st.Chunks) != 4 {
		t.Errorf("should have 4 chunks, got %d", len(st.Chunks))
	}

	// The first chunk keeps its red foreground.
	if st.Chunks[0].Fg == nil || !st.Chunks[0].Fg.Equals(ColorRed) {
		t.Error("first chunk should have ColorRed fg")
	}

	// The second chunk keeps its bold decoration.
	if st.Chunks[1].Attributes != AttrBold {
		t.Error("second chunk should be bold")
	}

	// The fourth chunk keeps its URL.
	if st.Chunks[3].Link != "https://docs.example.com" {
		t.Error("fourth chunk should have link URL")
	}
}
