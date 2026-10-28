package extension_test

import (
	"slices"
	"testing"

	"github.com/unaquatic/fluid/internal/markdown"
	"github.com/unaquatic/fluid/internal/markdown/extension"
	"github.com/unaquatic/fluid/internal/markdown/md"
	"github.com/unaquatic/fluid/internal/markdown/parser"
)

// TestTypographer checks the punctuation that the typographer replaces, and
// that which it leaves as written.
func TestTypographer(t *testing.T) {
	m := md.New(md.WithExtensions(extension.Typographer))
	for _, tt := range []struct {
		name, src, want string
	}{
		{"single quotes", "This should 'be' replaced", "This should ‘be’ replaced"},
		{"double quotes", `This should "be" replaced`, "This should “be” replaced"},
		{"dashes, ellipsis, angle quotes", "**--** *---* a...<< b>>", "<strong>–</strong> <em>—</em> a…« b»"},
		{"many hyphens", "a----- b", "a—– b"},
		{"decades", "Some say '90s, others say 90's, but I can't say which is best.", "Some say ’90s, others say 90’s, but I can’t say which is best."},
		{"contractions", "Alice's, I'm ,Don't, You'd", "Alice’s, I’m ,Don’t, You’d"},
		{"longer contractions", "I've, I'll, You're", "I’ve, I’ll, You’re"},
		{"after a link", "[Cat][]'s Pajamas\n\n[Cat]: /c", `<a href="/c">Cat</a>’s Pajamas`},
		{"after punctuation", "Yahoo!'s", "Yahoo!’s"},
		{"feet and inches", `My height is 5'6"".`, "My height is 5'6&quot;&quot;."},
		{"inch mark", `"Monitor 21"" and "Monitor""`, "“Monitor 21&quot;” and “Monitor”&quot;"},
		{"before punctuation", `reported "issue 1 (IE-only)", "issue 2", 'issue3 (FF-only)', 'issue4'`,
			"reported “issue 1 (IE-only)”, “issue 2”, ‘issue3 (FF-only)’, ‘issue4’"},
		{"in emphasis", `***"At first, things were not clear."***`, "<em><strong>“At first, things were not clear.”</strong></em>"},
		{"plural possessive", "The Smiths' dog is named Rover.", "The Smiths’ dog is named Rover."},
		{"dropped letter", `"I'm not doin' that," Bill said.`, "“I’m not doin’ that,” Bill said."},
		{"quotes in quotes", `Janet said, "When everything is 'breaking news,' nothing is 'breaking news.'"`,
			"Janet said, “When everything is ‘breaking news,’ nothing is ‘breaking news.’”"},
		{"abbreviations", "internet --- 'net for short. Let's rock 'n roll!", "internet — ’net for short. Let’s rock ’n roll!"},
		{"around a link", `see "[Hugo](/t)").`, `see “<a href="/t">Hugo</a>”).`},
		{"in a link", `["moof" and the "dogcow."](/m)`, `<a href="/m">“moof” and the “dogcow.”</a>`},
		{"in alt text", "![Nice & day, **isn't** it?](/i.jpg)", `<img src="/i.jpg" alt="Nice &amp; day, isn’t it?">`},
		{"code, autolinks, and HTML", "`'code' --` <http://a--b> <b title=\"'x'\">",
			`<code>'code' --</code> <a href="http://a--b">http://a--b</a> <!-- raw HTML omitted -->`},
		{"escapes and entities", `\'x\' &quot;y&quot;`, "'x' &quot;y&quot;"},
		{"link title", `[a](/u "t--")`, `<a href="/u" title="t--">a</a>`},

		// Unlike goldmark, whose counts of open quotes never reset, each
		// block counts its own.
		{"quote in another block", "\"open\n\nclose\" now", "“open</p>\n<p>close&quot; now"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := convert(t, m, tt.src), "<p>"+tt.want+"</p>\n"; got != want {
				t.Errorf("got  %q\nwant %q", got, want)
			}
		})
	}
}

// TestTypographerBlocks checks that the typographer replaces punctuation in
// headings and table cells too.
func TestTypographerBlocks(t *testing.T) {
	m := md.New(md.WithExtensions(extension.Table, extension.Typographer))
	const want = "<h1>“Title”</h1>\n<table>\n<thead>\n<tr>\n<th>‘a’ – b</th>\n</tr>\n</thead>\n</table>\n"
	if got := convert(t, m, "# \"Title\"\n\n| 'a' -- b |\n| - |"); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

// TestTypographicSubstitutions checks that substitutions replace the
// defaults, that "" keeps punctuation as written, and that substitutions
// are text, which renders escaped.
func TestTypographicSubstitutions(t *testing.T) {
	m := md.New(md.WithExtensions(extension.NewTypographer(
		extension.WithTypographicSubstitutions(extension.TypographicSubstitutions{
			extension.LeftDoubleQuote:  "„",
			extension.RightDoubleQuote: "“",
			extension.EmDash:           "",
			extension.Ellipsis:         "<...>",
		}),
		extension.WithTypographicSubstitutions(extension.TypographicSubstitutions{extension.LeftAngleQuote: ""}),
	)))
	const want = "<p>„a“ b–-c d&lt;...&gt; &lt;&lt; e » ‘f’</p>\n"
	if got := convert(t, m, `"a" b---c d... << e >> 'f'`); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

// TestTypographerTree checks that replacements are text, which joins the
// text around it, with the extension of a parser on its own.
func TestTypographerTree(t *testing.T) {
	p := parser.NewParser(parser.WithExtensions(extension.Typographer))
	var got []string
	_ = markdown.Walk(p.Parse(`a--b "c"`), func(n markdown.Node, entering bool) (markdown.WalkStatus, error) {
		if t, ok := n.(*markdown.Text); ok && entering {
			got = append(got, t.Value)
		}
		return markdown.WalkContinue, nil
	})
	if want := []string{"a–b “c”"}; !slices.Equal(got, want) {
		t.Errorf("texts %q, want %q", got, want)
	}
}
