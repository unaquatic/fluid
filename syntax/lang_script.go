package syntax

import (
	"strings"
	"sync"
)

const (
	jsKeywords = `async await break case catch class const continue debugger
		default delete do else export extends finally for function if import in
		instanceof let new of return static super switch this throw try typeof
		var void while with yield from as`
	jsBuiltinVariables = `console window document globalThis process JSON Math Reflect
		Intl module exports`
	jsBuiltinFunctions = `require setTimeout setInterval clearTimeout clearInterval fetch
		parseInt parseFloat isNaN isFinite structuredClone queueMicrotask`
	jsBuiltinTypes = `Object Array String Number Boolean Symbol BigInt Promise
		Map Set WeakMap WeakSet Date RegExp Error TypeError RangeError
		SyntaxError Proxy Buffer`
	jsPrimitives = "true false null undefined NaN Infinity"
	tsKeywords   = `abstract asserts declare enum implements infer interface is
		keyof module namespace never private protected public readonly
		satisfies type unique override accessor`
	tsTypes = "string number boolean bigint symbol object void never unknown any"
)

// jsLexer returns a JavaScript lexer, with TypeScript's keywords and types
// if ts and JSX elements if jsx.
func jsLexer(ts, jsx bool) *CLike {
	words := wordKinds(Keyword, jsKeywords, BuiltinType, jsBuiltinTypes,
		BuiltinVariable, jsBuiltinVariables, BuiltinFunction, jsBuiltinFunctions,
		Primitive, jsPrimitives)
	if ts {
		// The variables come first so that TypeScript's keyword module, as in
		// declare module, overrides the CommonJS variable.
		words = wordKinds(BuiltinVariable, jsBuiltinVariables, Keyword, jsKeywords+" "+tsKeywords,
			BuiltinType, tsTypes+" "+jsBuiltinTypes, BuiltinFunction, jsBuiltinFunctions,
			Primitive, jsPrimitives)
	}
	return &CLike{
		Words: words,
		Declarators: map[string]Kind{
			"function": Function, "class": Type, "interface": Type, "type": Type, "enum": Type, "namespace": Type,
		},
		LineComments: []string{"//"},
		BlockComment: slashComments,
		Quotes: []Quote{
			{Open: `"`, Close: `"`, Escapes: true},
			{Open: `'`, Close: `'`, Escapes: true},
			{Open: "`", Close: "`", Escapes: true, Multiline: true, Interp: "${", InterpClose: "}"},
		},
		IdentChars:    "$",
		Decorators:    true,
		CapitalTypes:  true,
		CapsConstants: true,
		KeywordKeys:   true,
		Regex:         true,
		JSX:           jsx,
		Special:       jsSpecial,
	}
}

// jsSpecial lexes a shebang line and private class members, such as #x.
func jsSpecial(s *Scanner) bool {
	if s.Peek(0) != '#' {
		return false
	}
	if s.Pos == 0 && s.Peek(1) == '!' {
		s.EmitRest(Comment)
		return true
	}
	if IdentStart(s.Line, s.Pos+1, "$") {
		s.EmitTo(Property, IdentEnd(s.Line, s.Pos+1, "$"))
		return true
	}
	return false
}

// pythonLang returns Python.
func pythonLang() *Language {
	var quotes []Quote
	// Each group of prefixes takes the four delimiters, the triple ones
	// first.
	for _, g := range []struct {
		prefixes []string
		escapes  bool
		interp   bool
	}{
		{[]string{"r", "rb", "br", "ur"}, false, false},
		{[]string{"rf", "fr"}, false, true},
		{[]string{"f"}, true, true},
		{[]string{"b", "u"}, true, false},
		{nil, true, false},
	} {
		for _, d := range []string{`"""`, `'''`, `"`, `'`} {
			q := Quote{Open: d, Close: d, Prefixes: g.prefixes, Escapes: g.escapes, Multiline: len(d) == 3}
			if g.interp {
				q.Interp, q.InterpClose = "{", "}"
			}
			quotes = append(quotes, q)
		}
	}
	return &Language{
		Name:    "python",
		Aliases: []string{"py", "py3", "python3", "pyi", "pyw", "gyp", "bzl", "starlark"},
		Filenames: []string{
			"BUILD", "BUILD.bazel", "WORKSPACE", "SConstruct", "SConscript",
		},
		Lexer: &CLike{
			Words: wordKinds(
				Keyword, `and as assert async await break class continue def del elif
					else except finally for from global if import in is lambda
					nonlocal not or pass raise return try while with yield`,
				BuiltinFunction, `abs aiter all anext any ascii bin breakpoint callable chr
					compile delattr dir divmod eval exec format getattr globals hasattr
					hash help hex id input isinstance issubclass iter len locals max min
					next oct open ord pow print repr round setattr sorted sum vars
					__import__`,
				Constant, "NotImplemented Ellipsis",
				BuiltinType, `bool bytearray bytes complex dict float frozenset int
					list memoryview object range set slice str tuple type classmethod
					staticmethod property super enumerate zip map filter reversed
					BaseException Exception AssertionError AttributeError ImportError
					IndexError KeyError KeyboardInterrupt NotImplementedError OSError
					IOError FileNotFoundError RuntimeError StopIteration TypeError
					ValueError ZeroDivisionError`,
				BuiltinVariable, "self cls __name__ __file__ __doc__",
				Primitive, "True False None"),
			Declarators:   map[string]Kind{"def": Function, "class": Type},
			LineComments:  []string{"#"},
			Quotes:        quotes,
			Decorators:    true,
			CapsConstants: true,
		},
	}
}

// rubyLang returns Ruby.
func rubyLang() *Language {
	return &Language{
		Name:      "ruby",
		Aliases:   []string{"rb", "gemspec", "rake", "rbw", "ru", "podspec", "jbuilder"},
		Filenames: []string{"Gemfile", "Rakefile", "Podfile", "Vagrantfile", "Brewfile", "Guardfile"},
		Lexer: &CLike{
			Words: wordKinds(
				Keyword, `BEGIN END alias and begin break case class def defined? do
					else elsif end ensure for if in module next not or redo rescue
					retry return self super then undef unless until when while yield
					__FILE__ __LINE__ __method__ lambda proc`,
				BuiltinFunction, `puts print p pp require require_relative attr_accessor
					attr_reader attr_writer include extend prepend private public
					protected raise loop format sprintf gets`,
				Primitive, "true false nil"),
			Declarators:  map[string]Kind{"def": Function, "class": Type, "module": Type},
			LineComments: []string{"#"},
			Quotes: []Quote{
				{Open: `"`, Close: `"`, Escapes: true, Multiline: true, Interp: "#{", InterpClose: "}"},
				{Open: `'`, Close: `'`, Escapes: true, Multiline: true},
				{Open: "`", Close: "`", Escapes: true, Multiline: true, Interp: "#{", InterpClose: "}"},
			},
			IdentSuffixes: "?!",
			Sigils:        "@$",
			CapitalTypes:  true,
			CapsConstants: true,
			Regex:         true,
			Special:       rubySpecial,
		},
	}
}

// rubySpecial lexes Ruby's =begin comments, heredocs, symbols, and class
// variables.
func rubySpecial(s *Scanner) bool {
	rest := s.Rest()
	switch rest[0] {
	case '=':
		if s.Pos == 0 && strings.HasPrefix(rest, "=begin") {
			s.EmitRest(Comment)
			s.Heredoc(Comment, "=end", false, true)
			return true
		}
	case '<':
		return heredocStart(s, "<<", true, false)
	case ':':
		// A symbol, as in &:name, but not the second colon of :: or a
		// ternary's colon.
		if IdentStart(rest, 1, "") && (s.Pos == 0 || IsSpace(s.Line[s.Pos-1]) || strings.IndexByte("([{,=&", s.Line[s.Pos-1]) >= 0) {
			s.Emit(String, IdentEnd(rest, 1, ""))
			return true
		}
	case '@':
		if strings.HasPrefix(rest, "@@") && IdentStart(rest, 2, "") {
			s.Emit(Variable, IdentEnd(rest, 2, ""))
			return true
		}
	}
	return false
}

// heredocStart lexes the start of a heredoc, such as <<~EOS, <<-EOS, or
// <<'EOS', after the operator op. With dashes, as in Ruby, a tilde or dash
// lets the closing word be indented, and a bare word must be capitalized
// unless it does. With loose, as in PHP, the closing word may be indented
// and followed by more text, such as a semicolon. It reports whether a
// heredoc starts.
func heredocStart(s *Scanner, op string, dashes, loose bool) bool {
	rest := s.Rest()
	if !strings.HasPrefix(rest, op) {
		return false
	}
	i := len(op)
	indented := loose
	if dashes && i < len(rest) && (rest[i] == '~' || rest[i] == '-') {
		indented = true
		i++
	}
	var word string
	end := i
	if i < len(rest) && (rest[i] == '\'' || rest[i] == '"') {
		j := strings.IndexByte(rest[i+1:], rest[i])
		if j <= 0 {
			return false
		}
		word, end = rest[i+1:i+1+j], i+2+j
	} else {
		end = IdentEnd(rest, i, "")
		word = rest[i:end]
		if word == "" || (dashes && !indented && !IsCaps(word) && !IsCapitalized(word)) {
			return false
		}
	}
	s.Emit(Operator, i)
	s.Emit(String, end-i)
	s.Heredoc(String, word, indented, loose)
	return true
}

// phpLang returns PHP.
func phpLang() *Language {
	return &Language{
		Name:    "php",
		Aliases: []string{"php3", "php4", "php5", "php8", "phtml", "inc"},
		Lexer: &CLike{
			Words: wordKinds(
				Keyword, `abstract and array as break callable case catch class clone
					const continue declare default do echo else elseif empty
					enddeclare endfor endforeach endif endswitch endwhile enum eval
					exit extends final finally fn for foreach function global goto if
					implements include include_once instanceof insteadof interface
					isset list match namespace new or print private protected public
					readonly require require_once return static switch throw trait try
					unset use var while xor yield`,
				BuiltinType, "int float bool string void mixed never object iterable",
				Type, "self parent",
				Primitive, "true false null"),
			FoldCase: true,
			Declarators: map[string]Kind{
				"function": Function, "class": Type, "interface": Type, "trait": Type, "enum": Type,
			},
			LineComments: []string{"//", "#"},
			BlockComment: slashComments,
			Quotes: []Quote{
				{Open: `"`, Close: `"`, Escapes: true, Multiline: true, Sigil: '$'},
				{Open: `'`, Close: `'`, Escapes: true, Multiline: true},
			},
			Sigils:        "$",
			CapitalTypes:  true,
			CapsConstants: true,
			Special:       phpSpecial,
		},
	}
}

// phpSpecial lexes PHP's open and close tags, heredocs, and attributes.
func phpSpecial(s *Scanner) bool {
	rest := s.Rest()
	switch {
	case strings.HasPrefix(rest, "<?php"):
		s.Emit(Preproc, 5)
	case strings.HasPrefix(rest, "<?="):
		s.Emit(Preproc, 3)
	case strings.HasPrefix(rest, "?>"):
		s.Emit(Preproc, 2)
	case strings.HasPrefix(rest, "#["):
		s.Emit(Decorator, bracketEnd(rest, 1))
	default:
		return heredocStart(s, "<<<", false, true)
	}
	return true
}

// luaLang returns Lua.
func luaLang() *Language {
	return &Language{
		Name:    "lua",
		Aliases: []string{"luau", "wlua"},
		Lexer: &CLike{
			Words: wordKinds(
				Keyword, `and break do else elseif end for function goto if in local
					not or repeat return then until while`,
				BuiltinFunction, `assert collectgarbage dofile error getmetatable ipairs load
					loadfile next pairs pcall print rawequal rawget rawlen rawset
					require select setmetatable tonumber tostring type unpack xpcall`,
				BuiltinVariable, "self _G _ENV string table math io os coroutine debug utf8",
				Primitive, "true false nil"),
			Declarators:  map[string]Kind{"function": Function},
			LineComments: []string{"--"},
			Quotes:       cQuotes(),
			Special:      luaSpecial,
		},
	}
}

// luaSpecial lexes Lua's long brackets: comments such as --[[ ]] and
// strings such as [==[ ]==].
func luaSpecial(s *Scanner) bool {
	rest := s.Rest()
	k, i := String, 0
	if strings.HasPrefix(rest, "--[") {
		k, i = Comment, 2
	}
	if i >= len(rest) || rest[i] != '[' {
		return false
	}
	level := len(rest[i+1:]) - len(strings.TrimLeft(rest[i+1:], "="))
	if i+1+level >= len(rest) || rest[i+1+level] != '[' {
		return false
	}
	s.Emit(k, i+2+level)
	s.Delimit(k, "]"+strings.Repeat("=", level)+"]")
	return true
}

// sqlLang returns SQL.
func sqlLang() *Language {
	return &Language{
		Name:    "sql",
		Aliases: []string{"mysql", "postgresql", "postgres", "psql", "plpgsql", "sqlite", "sqlite3", "tsql", "pgsql", "ddl", "dml"},
		Lexer: &CLike{
			Words: wordKinds(
				Keyword, `add all alter analyze and any as asc begin between by
					cascade case cast check collate column commit constraint create
					cross current_date current_time current_timestamp database default
					delete desc distinct drop else end escape except exists explain
					foreign from full grant group having if in index inner insert
					intersect into is join key left like limit natural not offset on
					or order outer over partition primary procedure references replace
					returning revoke right rollback row rows savepoint schema select
					set table temp temporary then to transaction trigger truncate
					union unique update using values view when where window with
					without recursive function returns language declare do loop while
					for return materialized sequence extension conflict nothing
					lateral ilike similar fetch first next only interval`,
				BuiltinType, `bigint binary bit blob boolean bool char character date
					datetime decimal double float int integer json jsonb money nchar
					numeric nvarchar real serial bigserial smallint smallserial text
					time timestamp timestamptz tinyint uuid varbinary varchar bytea`,
				BuiltinFunction, `avg count max min sum coalesce nullif now lower upper length
					substring trim round abs concat date_trunc extract to_char
					row_number rank dense_rank lag lead first_value last_value
					string_agg array_agg json_agg greatest least`,
				Primitive, "true false null unknown"),
			FoldCase:     true,
			Declarators:  map[string]Kind{"function": Function, "procedure": Function},
			LineComments: []string{"--"},
			BlockComment: slashComments,
			Quotes: []Quote{
				{Open: `'`, Close: `'`, Prefixes: []string{"e"}, Escapes: true, Multiline: true},
				{Open: `'`, Close: `'`, Doubled: true, Multiline: true},
				{Open: `"`, Close: `"`, Doubled: true},
				{Open: "`", Close: "`", Doubled: true},
			},
			Sigils:  "@",
			Special: sqlSpecial,
		},
	}
}

// sqlSpecial lexes positional parameters, such as $1, and dollar-quoted
// strings, such as $$...$$ or $body$...$body$.
func sqlSpecial(s *Scanner) bool {
	rest := s.Rest()
	if rest[0] != '$' || len(rest) < 2 {
		return false
	}
	if IsDigit(rest[1]) {
		end := 1
		for end < len(rest) && IsDigit(rest[end]) {
			end++
		}
		s.Emit(Variable, end)
		return true
	}
	end := 1
	if rest[1] != '$' {
		end = IdentEnd(rest, 1, "")
		if end >= len(rest) || rest[end] != '$' {
			return false
		}
	}
	tag := rest[:end+1]
	s.Emit(String, len(tag))
	s.Delimit(String, tag)
	return true
}

// powershellLang returns PowerShell.
func powershellLang() *Language {
	return &Language{
		Name:    "powershell",
		Aliases: []string{"pwsh", "ps1", "psm1", "psd1", "posh"},
		Lexer: &CLike{
			Words: wordKinds(
				Keyword, `begin break catch class clean continue data define do
					dynamicparam else elseif end enum exit filter finally for foreach
					from function hidden if in param process return static switch
					throw trap try until using var while workflow parallel sequence
					inlinescript`,
				BuiltinType, `string int long bool double float decimal char byte object
					array hashtable void datetime scriptblock psobject
					pscustomobject regex xml`),
			FoldCase:     true,
			Declarators:  map[string]Kind{"function": Function, "class": Type, "enum": Type},
			LineComments: []string{"#"},
			BlockComment: [2]string{"<#", "#>"},
			Quotes: []Quote{
				{Open: `"`, Close: `"`, Doubled: true, Multiline: true, Sigil: '$'},
				{Open: `'`, Close: `'`, Doubled: true, Multiline: true},
			},
			IdentChars: "-",
			Special:    powershellSpecial,
		},
	}
}

// psOperators returns the set of PowerShell's operators that are words
// after a dash, such as -eq, without the dash.
var psOperators = sync.OnceValue(func() map[string]bool {
	set := make(map[string]bool)
	for w := range strings.FieldsSeq(`eq ne gt ge lt le like notlike match
		notmatch contains notcontains in notin replace split join is isnot as
		and or xor not band bor bxor bnot shl shr f ceq cne cgt cge clt cle ieq
		ine igt ige ilt ile clike cnotlike cmatch cnotmatch imatch inotmatch
		creplace ireplace csplit isplit`) {
		set[w] = true
	}
	return set
})

// powershellSpecial lexes here-strings, variables, word operators such as
// -eq, and cmdlet names such as Get-ChildItem.
func powershellSpecial(s *Scanner) bool {
	rest := s.Rest()
	switch b := rest[0]; {
	case b == '@' && (strings.HasPrefix(rest, `@"`) || strings.HasPrefix(rest, `@'`)) && strings.TrimSpace(rest[2:]) == "":
		s.Emit(String, 2)
		s.Heredoc(String, rest[1:2]+"@", true, true)
		return true
	case b == '$' && IdentStart(rest, 1, ""):
		end := IdentEnd(rest, 1, "")
		k := Variable
		if w := strings.ToLower(rest[1:end]); w == "true" || w == "false" || w == "null" {
			k = Primitive
		}
		s.Emit(k, end)
		return true
	case b == '-':
		// IdentChars lets a dash join a cmdlet's name, but a dash that
		// starts a word is an operator: a word operator, such as -eq, or
		// the dash before a parameter or an operand, as in -Path or a - 1.
		end := 1
		if IdentStart(rest, 1, "") {
			if e := IdentEnd(rest, 1, ""); psOperators()[strings.ToLower(rest[1:e])] {
				end = e
			}
		}
		s.Emit(Operator, end)
		return true
	case IdentStart(rest, 0, ""):
		end := IdentEnd(rest, 0, "-")
		dash := strings.IndexByte(rest[:end], '-')
		if dash <= 0 || dash == end-1 || !IsLetter(rest[dash+1]) {
			return false
		}
		s.Emit(Function, end)
		return true
	}
	return false
}
