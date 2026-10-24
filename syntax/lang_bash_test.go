package syntax

import "testing"

func TestBash(t *testing.T) {
	testLanguage(t, "bash", []string{"sh", "shell", "zsh", "ksh", ".bashrc", ".zshrc", "PKGBUILD", "deploy.sh"}, bashTests)
}

var bashTests = []kindCase{
	{name: "commands and builtins", src: "echo hello; cd /tmp && ls -la | grep foo || exit 1", want: []Token{
		tok(Builtin, "echo"), tok(Text, "hello"), tok(Operator, ";"), tok(Builtin, "cd"), tok(Text, "/tmp"),
		tok(Operator, "&&"), tok(Text, "ls -la"), tok(Operator, "|"), tok(Text, "grep foo"), tok(Operator, "||"),
		tok(Builtin, "exit"), tok(Number, "1"),
	}},
	{name: "builtin names as arguments", src: "go test ./...\ngit commit -m done", want: []Token{
		tok(Text, "go test ./..."), tok(Text, "git commit -m done"),
	}},
	{name: "variables", src: "echo $HOME ${PATH:-/bin} $1 $@ $# $? $$ ${#arr[@]} ${x/a/b}", want: []Token{
		tok(Variable, "$HOME"), tok(Variable, "${PATH:-/bin}"), tok(Variable, "$1"), tok(Variable, "$@"),
		tok(Variable, "$#"), tok(Variable, "$?"), tok(Variable, "$$"), tok(Variable, "${#arr[@]}"), tok(Variable, "${x/a/b}"),
	}},
	{name: "assignments", src: "FOO=bar BAZ+=qux cmd arg\nexport PATH=/usr/bin:$PATH\nlocal x=1\nreadonly Y", want: []Token{
		tok(Variable, "FOO"), tok(Operator, "="), tok(Text, "bar"), tok(Variable, "BAZ"), tok(Operator, "+="),
		tok(Text, "qux"), tok(Text, "cmd arg"), tok(Builtin, "export"), tok(Variable, "$PATH"), tok(Builtin, "local"),
		tok(Builtin, "readonly"),
	}},
	{name: "quotes", src: "echo \"Hi $name, ${x} $(date +%s) \\\"q\\\"\" 'single $x' $'ansi\\n'", want: []Token{
		tok(String, `"Hi `), tok(Variable, "$name"), tok(String, ", "), tok(Variable, "${x}"), tok(String, " "),
		tok(StringEscape, "$(date +%s)"), tok(String, " "), tok(StringEscape, `\"`), tok(String, "q"),
		tok(StringEscape, `\"`), tok(String, `"`), tok(String, "'single $x'"), tok(String, "$'ansi"),
		tok(StringEscape, `\n`), tok(String, "'"),
	}},
	{name: "strings span lines", src: "msg=\"line one\nline two\"\necho \"$msg\"", want: []Token{
		tok(Variable, "msg"), tok(Operator, "="), tok(String, `"line one`), tok(String, `line two"`), tok(Builtin, "echo"),
		tok(Variable, "$msg"),
	}},
	{name: "conditionals", src: "if [[ -f x && $# -gt 0 ]]; then echo y; elif [ -d z ]; then :; else exit; fi", want: []Token{
		tok(Keyword, "if"), tok(Keyword, "[["), tok(Operator, "&&"), tok(Variable, "$#"), tok(Keyword, "]]"),
		tok(Operator, ";"), tok(Keyword, "then"), tok(Builtin, "echo"), tok(Keyword, "elif"), tok(Builtin, "["),
		tok(Keyword, "then"), tok(Keyword, "else"), tok(Builtin, "exit"), tok(Keyword, "fi"),
	}},
	{name: "loops", src: "for f in *.go; do echo $f; done\nwhile read -r line; do continue; done < file\nuntil false; do break; done\nselect opt in a b; do :; done", want: []Token{
		tok(Keyword, "for"), tok(Text, "f"), tok(Keyword, "in"), tok(Text, "*.go"), tok(Keyword, "do"), tok(Builtin, "echo"),
		tok(Keyword, "done"), tok(Keyword, "while"), tok(Builtin, "read"), tok(Keyword, "do"), tok(Builtin, "continue"),
		tok(Keyword, "done"), tok(Operator, "<"), tok(Keyword, "until"), tok(Builtin, "false"), tok(Builtin, "break"),
		tok(Keyword, "select"), tok(Keyword, "in"),
	}},
	{name: "case", src: "case $x in\n  a|b) echo ab ;;\n  *) ;;\nesac", want: []Token{
		tok(Keyword, "case"), tok(Variable, "$x"), tok(Keyword, "in"), tok(Operator, "|"), tok(Operator, ";;"),
		tok(Operator, ";;"), tok(Keyword, "esac"),
	}},
	{name: "functions", src: "function deploy {\n  echo go\n}\nbuild() { make; }", want: []Token{
		tok(Keyword, "function"), tok(Function, "deploy"), tok(Punctuation, "{"), tok(Builtin, "echo"),
		tok(Punctuation, "}"), tok(Function, "build"), tok(Punctuation, "()"), tok(Punctuation, "{"), tok(Text, "make"),
	}},
	{name: "heredocs", src: "cat <<EOF > out\nhello $USER\nEOF\ncat <<-'END'\n\tindented\n\tEND\ncat << \"Q\"\nq\nQ\necho done", want: []Token{
		tok(Text, "cat"), tok(Operator, "<<"), tok(String, "EOF"), tok(Operator, ">"), tok(String, "hello $USER"),
		tok(String, "EOF"), tok(Operator, "<<-"), tok(String, "'END'"), tok(String, "\tindented"), tok(String, "\tEND"),
		tok(Operator, "<<"), tok(String, `"Q"`), tok(String, "q"), tok(String, "Q"), tok(Builtin, "echo"),
	}},
	{name: "here-strings", src: "grep x <<< \"$s\"\necho next", want: []Token{
		tok(Operator, "<<<"), tok(Variable, "$s"), tok(Builtin, "echo"),
	}},
	{name: "arithmetic", src: "echo $((1 << n)); (( x <<= 2 ))\necho $[1<<2]\necho next", want: []Token{
		tok(Operator, "$(("), tok(Operator, "<<"), tok(Punctuation, "))"), tok(Punctuation, "(("), tok(Operator, "<<="),
		tok(Punctuation, "))"), tok(Builtin, "echo"), tok(Operator, "<<"), tok(Builtin, "echo"),
	}},
	{name: "comments", src: "# comment\necho a#b # real\n#!/bin/sh", want: []Token{
		tok(Comment, "# comment"), tok(Builtin, "echo"), tok(Text, "a#b"), tok(Comment, "# real"), tok(Comment, "#!/bin/sh"),
	}},
	{name: "redirections and jobs", src: "cmd 2>&1 >> log &> all 1>/dev/null &", want: []Token{
		tok(Number, "2"), tok(Operator, ">&"), tok(Number, "1"), tok(Operator, ">>"), tok(Operator, "&>"), tok(Number, "1"),
		tok(Operator, ">"), tok(Text, "/dev/null"), tok(Operator, "&"),
	}},
	{name: "command substitution", src: "x=$(git rev-parse HEAD); y=`date`; z=$(echo $(pwd))", want: []Token{
		tok(Variable, "x"), tok(Operator, "$("), tok(Text, "git"), tok(Punctuation, ")"), tok(Variable, "y"),
		tok(String, "`date`"), tok(Variable, "z"), tok(Operator, "$("), tok(Builtin, "echo"), tok(Operator, "$("),
		tok(Builtin, "pwd"),
	}},
	{name: "line continuation", src: "docker run \\\n  --rm \\\n  image\nset -e", want: []Token{
		tok(Text, "docker run"), tok(StringEscape, `\`), tok(Text, "--rm"), tok(StringEscape, `\`), tok(Text, "image"),
		tok(Builtin, "set"),
	}},
	{name: "continued command after an operator", src: "make && \\\n  echo ok", want: []Token{
		tok(Operator, "&&"), tok(StringEscape, `\`), tok(Builtin, "echo"),
	}},
	{name: "unclosed string", src: "echo \"open\nstill", open: true, want: []Token{
		tok(String, `"open`), tok(String, "still"),
	}},
}
