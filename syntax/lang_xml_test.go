package syntax

import "testing"

func TestXML(t *testing.T) {
	testLanguage(t, "xml", []string{"svg", "xsd", "xsl", "plist", "csproj", "pom.xml", "icon.svg"}, xmlTests)
}

var xmlTests = []kindCase{
	{name: "documents", src: "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<root xmlns:x=\"urn:x\">\n  <x:item id=\"1\">text</x:item>\n  <empty/>\n</root>", want: []Token{
		tok(Preproc, `<?xml version="1.0" encoding="UTF-8"?>`), tok(Tag, "root"), tok(Attribute, "xmlns:x"),
		tok(String, `"urn:x"`), tok(Tag, "x:item"), tok(Attribute, "id"), tok(String, `"1"`), tok(Text, "text"),
		tok(Tag, "x:item"), tok(Tag, "empty"), tok(Punctuation, "/>"), tok(Tag, "root"),
	}},
	{name: "cdata and comments", src: "<a><![CDATA[ <not> & tags ]]></a>\n<!-- c -->", want: []Token{
		tok(Preproc, "<![CDATA[ <not> & tags ]]>"), tok(Tag, "a"), tok(Comment, "<!-- c -->"),
	}},
	{name: "cdata across lines", src: "<![CDATA[\nline\n]]><b/>", want: []Token{
		tok(Preproc, "<![CDATA["), tok(Preproc, "line"), tok(Preproc, "]]>"), tok(Tag, "b"),
	}},
	{name: "declarations", src: "<!DOCTYPE note SYSTEM \"Note.dtd\">\n<!ENTITY e \"x\">", want: []Token{
		tok(Preproc, `<!DOCTYPE note SYSTEM "Note.dtd">`), tok(Preproc, `<!ENTITY e "x">`),
	}},
	{name: "entities", src: "<a>&lt;&amp;&custom;&#10;</a>", want: []Token{
		tok(StringEscape, "&lt;&amp;&custom;&#10;"),
	}},
	{name: "processing instructions across lines", src: "<?target\n  data\n?><a/>", want: []Token{
		tok(Preproc, "<?target"), tok(Preproc, "  data"), tok(Preproc, "?>"), tok(Tag, "a"),
	}},
	{name: "no embedded scripts", src: "<script>var x</script>", want: []Token{
		tok(Tag, "script"), tok(Text, "var x"), tok(Tag, "script"),
	}},
}
