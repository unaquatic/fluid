// Package syntax splits source code into classified tokens for syntax
// highlighting.
//
// A [Language] lexes source one line at a time with its [Lexer]. The [State]
// one line ends in is the State the next line starts in, so a caller that
// keeps the State at the start of each line can resume lexing at any line,
// as when appended text changes only the last lines of a document.
//
// Lexing is lexical: a lexer classifies words, literals, and comments by
// their spelling and the tokens around them, without parsing. Text a lexer
// does not recognize is [Text].
package syntax

import (
	"iter"
	"strconv"
	"strings"
	"sync"
)

// Kind classifies a token by the role its text plays in source code.
type Kind uint8

const (
	// Text is text with no more specific role, such as whitespace or prose.
	Text Kind = iota
	// Keyword is a reserved word, such as if or return.
	Keyword
	// Type names a type that is not built in, such as a declared class.
	Type
	// Function names a function or method where it is declared or called.
	Function
	// BuiltinType names a type the language or its runtime predeclares,
	// such as int, str, or Array.
	BuiltinType
	// BuiltinFunction names a function the language or its runtime
	// predeclares, such as len, print, or fetch.
	BuiltinFunction
	// BuiltinVariable names a variable or object the language or its runtime
	// predeclares, such as console, Math, self, or __name__.
	BuiltinVariable
	// Variable is an identifier with no more specific role, or a variable
	// marked by a sigil, such as $HOME.
	Variable
	// Constant names a constant, such as an all-capitals identifier.
	Constant
	// Primitive is a literal keyword, such as true, false, or nil.
	Primitive
	// Property names a field, key, or property, such as a JSON object key.
	Property
	// Attribute names a markup attribute, such as href.
	Attribute
	// Tag names a markup element, such as div.
	Tag
	// Decorator is an annotation or decorator, such as @Override.
	Decorator
	// String is a string or character literal.
	String
	// StringEscape is an escape sequence or interpolation in a string, such
	// as \n or ${name}.
	StringEscape
	// Regex is a regular expression literal.
	Regex
	// Number is a numeric literal.
	Number
	// Comment is a comment.
	Comment
	// Preproc is a preprocessor directive or compiler pragma, such as
	// #include.
	Preproc
	// Operator is an operator, such as + or :=.
	Operator
	// Punctuation is a delimiter or separator, such as a bracket or a comma.
	Punctuation
	// Heading is a top-level heading, such as a Markdown h1 or a diff's
	// header.
	Heading
	// Subheading is a lower-level heading, such as a Markdown h2 or a diff's
	// hunk header.
	Subheading
	// Emph is emphasized prose.
	Emph
	// Strong is strongly emphasized prose.
	Strong
	// Inserted is text a diff adds.
	Inserted
	// Deleted is text a diff removes.
	Deleted

	// NumKinds is the number of kinds. Kinds run from 0 to NumKinds-1, so
	// an array of NumKinds elements can hold a value for each.
	NumKinds
)

// kindNames holds the String of each Kind.
var kindNames = [NumKinds]string{
	"Text", "Keyword", "Type", "Function", "BuiltinType", "BuiltinFunction",
	"BuiltinVariable", "Variable", "Constant", "Primitive", "Property",
	"Attribute", "Tag", "Decorator", "String", "StringEscape", "Regex", "Number",
	"Comment", "Preproc", "Operator", "Punctuation", "Heading", "Subheading",
	"Emph", "Strong", "Inserted", "Deleted",
}

// String returns the name of k, such as "Keyword".
func (k Kind) String() string {
	if k < NumKinds {
		return kindNames[k]
	}
	return "Kind(" + strconv.Itoa(int(k)) + ")"
}

// Token is a run of source text and its Kind.
type Token struct {
	// Kind classifies Text.
	Kind Kind
	// Text is the run of source text.
	Text string
}

// Lexer lexes source code one line at a time.
type Lexer interface {
	// AppendLine appends the tokens of line, which holds no newline, to dst
	// and returns the extended slice and the State the next line starts in.
	// Lexing starts in st, the State the previous line ended in, or the zero
	// State for the first line. The texts of the appended tokens concatenate
	// to line.
	AppendLine(dst []Token, line string, st State) ([]Token, State)
}

// Language is a language a Lexer lexes and the names Lookup finds it by.
type Language struct {
	// Lexer lexes the language.
	Lexer
	// Name is the canonical name of the language, in lower case, such as
	// "go".
	Name string
	// Aliases are other names of the language and its file extensions, in
	// lower case, such as "golang".
	Aliases []string
	// Filenames are the base names of files in the language that have no
	// extension to tell it by, such as "Makefile".
	Filenames []string
}

// Tokens returns an iterator over the tokens of src. A newline between
// two lines is a Text token of its own.
func (l *Language) Tokens(src string) iter.Seq[Token] {
	return func(yield func(Token) bool) {
		var (
			st   State
			toks []Token
		)
		first := true
		for line := range strings.SplitSeq(src, "\n") {
			if !first && !yield(Token{Kind: Text, Text: "\n"}) {
				return
			}
			first = false
			toks, st = l.AppendLine(toks[:0], line, st)
			for _, t := range toks {
				if !yield(t) {
					return
				}
			}
		}
	}
}

// registry holds the languages Lookup finds and an index of their names. It
// is built on first use, so that a program that never lexes pays nothing
// for the keyword tables.
var registry = sync.OnceValues(func() ([]*Language, map[string]*Language) {
	langs := builtinLanguages()
	index := make(map[string]*Language, len(langs)*4)
	for _, l := range langs {
		index[l.Name] = l
		for _, a := range l.Aliases {
			index[a] = l
		}
		for _, f := range l.Filenames {
			index[strings.ToLower(f)] = l
		}
	}
	return langs, index
})

// Lookup returns the language name names, or nil if none does. The name is
// compared in lower case with each language's Name, Aliases, and Filenames;
// failing that, a file name or path, such as "cmd/main.go", is looked up by
// its base name and then by its extension.
func Lookup(name string) *Language {
	_, index := registry()
	name = strings.ToLower(strings.TrimSpace(name))
	if l := index[name]; l != nil {
		return l
	}
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
		if l := index[name]; l != nil {
			return l
		}
	}
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return index[name[i+1:]]
	}
	return nil
}

// Languages returns an iterator over the languages Lookup finds.
func Languages() iter.Seq[*Language] {
	return func(yield func(*Language) bool) {
		langs, _ := registry()
		for _, l := range langs {
			if !yield(l) {
				return
			}
		}
	}
}
