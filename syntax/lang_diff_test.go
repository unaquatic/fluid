package syntax

import "testing"

func TestDiff(t *testing.T) {
	testLanguage(t, "diff", []string{"patch", "udiff", "fix.patch", "changes.diff"}, diffTests)
}

var diffTests = []kindCase{
	{name: "unified", src: "diff --git a/f.go b/f.go\nindex 83db48f..bf269f4 100644\n--- a/f.go\n+++ b/f.go\n@@ -1,3 +1,4 @@ func main() {\n context\n-removed\n+added\n\\ No newline at end of file", want: []Token{
		tok(Heading, "diff --git a/f.go b/f.go"), tok(Heading, "index 83db48f..bf269f4 100644"),
		tok(Deleted, "--- a/f.go"), tok(Inserted, "+++ b/f.go"), tok(Subheading, "@@ -1,3 +1,4 @@ func main() {"),
		tok(Text, " context"), tok(Deleted, "-removed"), tok(Inserted, "+added"), tok(Text, `\ No newline at end of file`),
	}},
	{name: "normal", src: "5,7c5,6\n< old\n---\n> new\n3a4\n> added\n8d7\n< gone", want: []Token{
		tok(Subheading, "5,7c5,6"), tok(Deleted, "< old"), tok(Strong, "---"), tok(Inserted, "> new"),
		tok(Subheading, "3a4"), tok(Inserted, "> added"), tok(Subheading, "8d7"), tok(Deleted, "< gone"),
	}},
	{name: "context", src: "*** a.txt\n--- b.txt\n***************\n! changed\n  same", want: []Token{
		tok(Text, "*** a.txt"), tok(Deleted, "--- b.txt"), tok(Text, "***************"), tok(Strong, "! changed"),
		tok(Text, "  same"),
	}},
	{name: "headers", src: "Index: file.c\n===================\ndiff -u a b", want: []Token{
		tok(Heading, "Index: file.c"), tok(Heading, "==================="), tok(Heading, "diff -u a b"),
	}},
	{name: "other lines", src: "plain text\n12 is not a command\nFrom: someone", want: []Token{
		tok(Text, "plain text"), tok(Text, "12 is not a command"), tok(Text, "From: someone"),
	}},
}
