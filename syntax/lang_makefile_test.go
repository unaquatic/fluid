package syntax

import "testing"

func TestMakefile(t *testing.T) {
	testLanguage(t, "makefile", []string{"make", "mk", "mak", "Makefile", "GNUmakefile", "build.mk"}, makefileTests)
}

var makefileTests = []kindCase{
	{name: "assignments", src: "CC := gcc\nCFLAGS += -O2 -Wall\nPREFIX ?= /usr/local\nSRCS != ls *.c\nOBJS = $(SRCS:.c=.o)\nLIB ::= x", want: []Token{
		tok(Variable, "CC"), tok(Operator, ":="), tok(Text, " gcc"), tok(Variable, "CFLAGS"), tok(Operator, "+="),
		tok(Text, " -O2 -Wall"), tok(Variable, "PREFIX"), tok(Operator, "?="), tok(Variable, "SRCS"), tok(Operator, "!="),
		tok(Variable, "OBJS"), tok(Operator, "="), tok(Variable, "$(SRCS:.c=.o)"), tok(Variable, "LIB"), tok(Operator, "::="),
	}},
	{name: "rules and recipes", src: ".PHONY: all clean\nall: build test\nbuild: $(OBJS)\n\t$(CC) -o $@ $^ $(LDFLAGS)\n\t@echo \"built $@\"\nclean:\n\t-rm -f *.o", want: []Token{
		tok(Function, ".PHONY"), tok(Operator, ":"), tok(Text, " all clean"), tok(Function, "all"), tok(Operator, ":"),
		tok(Text, " build test"), tok(Function, "build"), tok(Operator, ":"), tok(Variable, "$(OBJS)"), tok(Variable, "$(CC)"),
		tok(Text, " -o "), tok(Variable, "$@"), tok(Variable, "$^"), tok(Variable, "$(LDFLAGS)"), tok(Operator, "@"),
		tok(BuiltinFunction, "echo"), tok(String, `"built `), tok(Variable, "$@"), tok(String, `"`), tok(Function, "clean"),
		tok(Operator, ":"), tok(Operator, "-"), tok(Text, "rm -f *.o"),
	}},
	{name: "directives", src: "include config.mk\n-include deps.d\nifeq ($(OS),Windows_NT)\n  EXE := .exe\nelse\n  EXE :=\nendif\nexport PATH\noverride CFLAGS += -g", want: []Token{
		tok(Keyword, "include"), tok(Text, " config.mk"), tok(Keyword, "-include"), tok(Keyword, "ifeq"),
		tok(Variable, "$(OS)"), tok(Variable, "EXE"), tok(Operator, ":="), tok(Keyword, "else"), tok(Variable, "EXE"),
		tok(Operator, ":="), tok(Keyword, "endif"), tok(Keyword, "export"), tok(Keyword, "override"),
	}},
	{name: "comments", src: "# comment\nall: # trailing\n\techo hi # shell comment", want: []Token{
		tok(Comment, "# comment"), tok(Function, "all"), tok(Comment, "# trailing"), tok(BuiltinFunction, "echo"), tok(Text, "hi"),
		tok(Comment, "# shell comment"),
	}},
	{name: "recipes continued across lines", src: "test:\n\tgo test \\\n\t  ./... && \\\n\t  echo ok\nnext: x", want: []Token{
		tok(Function, "test"), tok(Text, "go test"), tok(StringEscape, `\`), tok(Text, "./..."), tok(Operator, "&&"),
		tok(StringEscape, `\`), tok(BuiltinFunction, "echo"), tok(Function, "next"), tok(Operator, ":"),
	}},
	{name: "functions and shell variables", src: "FILES := $(wildcard src/*.go)\nrun:\n\tfor f in $(FILES); do echo $$f; done", want: []Token{
		tok(Variable, "FILES"), tok(Variable, "$(wildcard src/*.go)"), tok(Function, "run"), tok(Keyword, "for"),
		tok(Keyword, "in"), tok(Variable, "$(FILES)"), tok(Operator, ";"), tok(Keyword, "do"), tok(BuiltinFunction, "echo"),
		tok(Variable, "$$f"), tok(Keyword, "done"),
	}},
	{name: "nested references", src: "OUT = $(patsubst %.c,%.o,$(SRC)) ${DIR}/x", want: []Token{
		tok(Variable, "$(patsubst %.c,%.o,$(SRC))"), tok(Variable, "${DIR}"), tok(Text, "/x"),
	}},
	{name: "double-colon and pattern rules", src: "lib.a:: a.o\n%.o: %.c\n\t$(CC) -c $<", want: []Token{
		tok(Function, "lib.a"), tok(Operator, "::"), tok(Function, "%.o"), tok(Operator, ":"), tok(Variable, "$<"),
	}},
	{name: "unclosed recipe continuation", src: "all:\n\techo \\", open: true, want: []Token{
		tok(BuiltinFunction, "echo"), tok(StringEscape, `\`),
	}},
}
