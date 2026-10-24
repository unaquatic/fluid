package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/unaquatic/fluid/input"
	"github.com/unaquatic/fluid/layout"
	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/terminal"
	"github.com/unaquatic/fluid/textbuf"
	"github.com/unaquatic/fluid/widget"
)

var demoExtmarksCmd = Demo{
	Name:  "extmarks",
	Short: "Virtual text ranges that the cursor skips over",
	Run:   runExtmarksDemo,
}

const extmarksInitialContent = `Welcome to the Extmarks Demo!

This editor showcases virtual extmarks: text ranges the cursor jumps over.

Try moving through the [VIRTUAL] markers below with arrow keys:
- The cursor skips over [VIRTUAL] ranges automatically.
- Backspace at the end of a [VIRTUAL] marker deletes the whole marker!
- Press Ctrl+L to add a new [MARKER] at the cursor position.

Example: [LINK:https://example.com] and [TAG:important] are virtual.

Regular text can be edited normally.`

func runExtmarksDemo(ctx context.Context, o DemoOptions) error {
	r, err := terminal.NewRenderer(terminal.Config{
		UseMouse:          !o.NoMouse,
		AltSendsEscPrefix: true,
	})
	if err != nil {
		return fmt.Errorf("renderer: %w", err)
	}

	root := r.Root()

	editor := widget.NewTextarea(
		widget.TextareaWithID("editor"),
		widget.TextareaWithText(extmarksInitialContent),
		widget.TextareaWithWrapMode(widget.WrapWord),
		widget.TextareaWithShowCursor(true),
		widget.TextareaWithFlexGrow(1),
	)
	helpText := widget.NewText(
		"Esc/Q: Exit | Arrows: Move | Shift+Opt+Arrow: Select word | Ctrl+L: Add marker | Ctrl+D/K: Delete/Clear",
		widget.TextWithHeight(layout.Points(1)),
	)
	statusText := widget.NewText("",
		widget.TextWithHeight(layout.Points(1)),
	)
	debugText := widget.NewText("Press any key to see raw input...",
		widget.TextWithHeight(layout.Points(2)),
	)

	container := widget.NewBox(
		widget.BoxWithID("container"),
		widget.BoxWithFlexGrow(1),
		widget.BoxWithPadding(layout.Points(1)),
		widget.BoxWithFlexDirection(layout.DirectionColumn),
		widget.Children(
			widget.NewBox(
				widget.BoxWithID("editor-box"),
				widget.BoxWithBorder(style.BorderAll()),
				widget.BoxWithBorderStyle(style.BorderSingle),
				widget.BoxWithTitle("Extmarks Demo"),
				widget.BoxWithPadding(layout.Points(1)),
				widget.BoxWithFlexGrow(1),
				widget.Children(editor),
			),
			helpText,
			statusText,
			debugText,
		),
	)
	root.Add(container)

	ec := editor.Extmarks()

	virtualFg := style.ColorFromHex("#4DB8FF")
	virtualBg := style.ColorFromHex("#1A3347")
	tagFg := style.ColorFromHex("#7EE787")
	tagBg := style.ColorFromHex("#1A3326")

	markerTypeID := ec.RegisterType("marker")
	linkTypeID := ec.RegisterType("link")
	tagTypeID := ec.RegisterType("tag")

	autoDetectExtmarks(ec, editor.Text(), r.LinkRegistry(), virtualFg, virtualBg, tagFg, tagBg, markerTypeID, linkTypeID, tagTypeID)

	r.EventBus().On(input.EventKeyboard, func(e input.TerminalEvent) {
		if e.Key == nil {
			return
		}
		k := e.Key

		if k.Type == input.KeyPress || k.Type == input.KeyRepeat {
			rawBytes := []byte(k.Raw)
			hexParts := make([]string, len(rawBytes))
			for i, b := range rawBytes {
				hexParts[i] = fmt.Sprintf("%02x", b)
			}
			debugInfo := fmt.Sprintf("Key: name=%q ctrl=%v shift=%v alt=%v meta=%v | Raw: [%s] src=%s",
				k.Name, k.Mods.Has(input.ModCtrl), k.Mods.Has(input.ModShift), k.Mods.Has(input.ModAlt), k.Mods.Has(input.ModMeta),
				strings.Join(hexParts, " "),
				k.Source)
			debugText.SetContent(debugInfo)
		}

		if k.Name == "escape" || (k.Name == "q" && !k.Mods.Has(input.ModCtrl) && !k.Mods.Has(input.ModAlt)) {
			r.Destroy()
			return
		}

		if k.Mods.Has(input.ModCtrl) && k.Name == "l" {
			k.PreventDefault()
			offset := editor.CursorOffset()
			markerText := "[MARKER]"
			editor.InsertText(markerText)
			ec.Create(textbuf.ExtmarkOptions{
				Start:   offset,
				End:     offset + textbuf.DisplayWidthOfText(markerText),
				Virtual: true,
				Fg:      virtualFg,
				Bg:      virtualBg,
				TypeID:  markerTypeID,
				Data:    "manual",
			})
			helpText.SetContent("Added virtual [MARKER] at cursor!")
			return
		}

		if k.Mods.Has(input.ModCtrl) && k.Name == "d" {
			k.PreventDefault()
			offset := editor.CursorOffset()
			ems := ec.AtOffset(offset)
			if len(ems) > 0 {
				ec.Delete(ems[0].ID)
				helpText.SetContent(fmt.Sprintf("Deleted extmark #%d", ems[0].ID))
			} else {
				helpText.SetContent("No extmark at cursor")
			}
			return
		}

		if k.Mods.Has(input.ModCtrl) && k.Name == "k" {
			k.PreventDefault()
			count := len(ec.All())
			ec.Clear()
			helpText.SetContent(fmt.Sprintf("Cleared %d extmark(s)", count))
			return
		}
	})

	editor.OnCursorChange = func(row, col int) {
		offset := editor.CursorOffset()
		ems := ec.AtOffset(offset)
		virtualCount := len(ec.Virtual())
		totalCount := len(ec.All())

		info := fmt.Sprintf("Ln %d, Col %d | Offset %d | Extmarks: %d (%d virtual)",
			row+1, col+1, offset, totalCount, virtualCount)
		if len(ems) > 0 {
			em := ems[0]
			typeName := ec.TypeName(em.TypeID)
			if typeName == "" {
				typeName = "default"
			}
			info += fmt.Sprintf(" | Inside: #%d [%d,%d) type=%s", em.ID, em.Start, em.End, typeName)
		}
		statusText.SetContent(info)
	}

	if err := r.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	r.FocusRenderable(editor)

	return r.Wait(ctx)
}

var bracketPattern = regexp.MustCompile(`\[(VIRTUAL|LINK:[^\]]+|TAG:[^\]]+|MARKER)\]`)

func autoDetectExtmarks(
	ec *textbuf.ExtmarksController,
	text string,
	lr *terminal.LinkRegistry,
	virtualFg, virtualBg, tagFg, tagBg style.Color,
	markerTypeID, linkTypeID, tagTypeID int,
) {
	matches := bracketPattern.FindAllStringIndex(text, -1)
	for _, loc := range matches {
		byteStart, byteEnd := loc[0], loc[1]
		content := text[byteStart:byteEnd]

		// Extmark offsets use display width rather than byte positions.
		start := byteOffsetToDisplayWidth(text, byteStart)
		end := start + textbuf.DisplayWidthOfText(content)

		fg, bg := virtualFg, virtualBg
		typeID := markerTypeID
		var attrs style.Attrs

		if len(content) > 5 && content[1:5] == "LINK" {
			typeID = linkTypeID
			url := content[6 : len(content)-1]
			linkID := lr.Alloc(url)
			attrs = style.AttrUnderline.WithLink(linkID)
		} else if len(content) > 4 && content[1:4] == "TAG" {
			typeID = tagTypeID
			fg, bg = tagFg, tagBg
		}

		ec.Create(textbuf.ExtmarkOptions{
			Start:      start,
			End:        end,
			Virtual:    true,
			Fg:         fg,
			Bg:         bg,
			Attributes: attrs,
			TypeID:     typeID,
			Data:       content,
		})
	}
}

// byteOffsetToDisplayWidth converts a byte offset to a grapheme display offset.
func byteOffsetToDisplayWidth(text string, byteOffset int) int {
	width := 0
	i := 0
	for i < len(text) && i < byteOffset {
		if text[i] == '\n' {
			width++
			i++
			continue
		}
		cluster, _, cw := textbuf.NextGraphemeCluster(text[i:])
		width += textbuf.ClusterWidth(cw)
		i += len(cluster)
	}
	return width
}
