package extension

import (
	"github.com/unaquatic/fluid/internal/markdown"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// taskListTransformer recognizes task list markers.
type taskListTransformer struct{}

// NewTaskListTransformer returns the transformer of paragraphs that
// recognizes task list markers: "[ ]", "[x]", or "[X]" at the start of the
// first paragraph of a list item, followed by whitespace or the paragraph's
// end. It removes the marker from the paragraph and sets the item's task. A
// paragraph that held only the marker is left empty, so that the item's
// span still covers it, until the transformer that [NewTaskListCleanup]
// returns removes it.
func NewTaskListTransformer() parser.ParagraphTransformer { return taskListTransformer{} }

// Transform implements [parser.ParagraphTransformer].
func (taskListTransformer) Transform(p *markdown.Paragraph, s *parser.BlockState) {
	item, ok := p.Parent().(*markdown.ListItem)
	if !ok || item.FirstChild() != markdown.Node(p) {
		return
	}
	lines := p.Lines()
	text := lines[0].Text
	if len(lines) > 1 {
		text += "\n"
	}
	task, _ := taskMarker(text)
	if task == markdown.NotTask {
		return
	}
	item.Task = task
	lines[0] = markdown.Line{Text: lines[0].Text[3:], Offset: lines[0].Offset + 3}
	for _, l := range lines {
		if !isBlank(l.Text) {
			p.SetLines(lines)
			return
		}
	}
	p.SetLines(nil)
}

// taskListCleanup removes the empty paragraphs of task list items.
type taskListCleanup struct{}

// NewTaskListCleanup returns the transformer of documents that removes the
// empty paragraphs that the transformer that [NewTaskListTransformer]
// returns leaves at the start of task list items.
func NewTaskListCleanup() parser.ASTTransformer { return taskListCleanup{} }

// Transform implements [parser.ASTTransformer].
func (taskListCleanup) Transform(doc *markdown.Document) {
	_ = markdown.Walk(doc, func(n markdown.Node, entering bool) (markdown.WalkStatus, error) {
		item, ok := n.(*markdown.ListItem)
		if !entering || !ok || item.Task == markdown.NotTask {
			return markdown.WalkContinue, nil
		}
		if p, ok := item.FirstChild().(*markdown.Paragraph); ok && p.Lines() == nil {
			item.RemoveChild(item, p)
		}
		return markdown.WalkContinue, nil
	})
}

// taskMarker returns the task list marker at the start of paragraph content
// s, if any, and the rest of s after it.
func taskMarker(s string) (markdown.Task, string) {
	if len(s) < 3 || s[0] != '[' || s[2] != ']' {
		return markdown.NotTask, s
	}
	t := markdown.Checked
	switch s[1] {
	case ' ', '\t':
		t = markdown.Unchecked
	case 'x', 'X':
	default:
		return markdown.NotTask, s
	}
	if len(s) > 3 && s[3] != ' ' && s[3] != '\t' && s[3] != '\n' {
		return markdown.NotTask, s
	}
	return t, s[3:]
}
