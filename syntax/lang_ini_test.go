package syntax

import "testing"

func TestINI(t *testing.T) {
	testLanguage(t, "ini", []string{"cfg", "conf", "env", "dotenv", "properties", "editorconfig", ".env", ".gitconfig", "setup.cfg", "app.ini"}, iniTests)
}

var iniTests = []kindCase{
	{name: "sections and keys", src: "[core]\neditor = vim\nautocrlf=false\n[remote \"origin\"]\nurl = git@github.com:a/b.git", want: []Token{
		tok(Keyword, "[core]"), tok(Property, "editor"), tok(Operator, "="), tok(String, "vim"), tok(Property, "autocrlf"),
		tok(Operator, "="), tok(String, "false"), tok(Keyword, `[remote "origin"]`), tok(Property, "url"),
		tok(String, "git@github.com:a/b.git"),
	}},
	{name: "comments", src: "; semicolon\n# hash\n  ; indented\nkey = value", want: []Token{
		tok(Comment, "; semicolon"), tok(Comment, "# hash"), tok(Comment, "; indented"), tok(Property, "key"),
	}},
	{name: "colon separators", src: "key: value\nname : spaced value", want: []Token{
		tok(Property, "key"), tok(Operator, ":"), tok(String, "value"), tok(Property, "name"), tok(Operator, ":"),
		tok(String, "spaced value"),
	}},
	{name: "dotenv", src: "export API_KEY=secret\nDEBUG=1\nEMPTY=", want: []Token{
		tok(Keyword, "export"), tok(Property, "API_KEY"), tok(Operator, "="), tok(String, "secret"), tok(Property, "DEBUG"),
		tok(String, "1"), tok(Property, "EMPTY"), tok(Operator, "="),
	}},
	{name: "lines without separators", src: "[section]\nstandalone", want: []Token{
		tok(Keyword, "[section]"), tok(Text, "standalone"),
	}},
}
