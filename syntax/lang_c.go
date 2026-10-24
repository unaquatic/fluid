package syntax

import (
	"slices"
	"strings"
)

// wordKinds builds a Words map from pairs of a Kind and a space-separated
// list of words.
func wordKinds(pairs ...any) map[string]Kind {
	m := make(map[string]Kind)
	for i := 0; i+1 < len(pairs); i += 2 {
		words(m, pairs[i].(Kind), pairs[i+1].(string))
	}
	return m
}

// slashComments are the comments of C: // to the end of the line and
// /* */ blocks.
var slashComments = [2]string{"/*", "*/"}

// cQuotes returns C's double-quoted strings and single-quoted characters,
// which may follow one of prefixes.
func cQuotes(prefixes ...string) []Quote {
	q := []Quote{
		{Open: `"`, Close: `"`, Escapes: true},
		{Open: `'`, Close: `'`, Escapes: true},
	}
	if len(prefixes) > 0 {
		q = append(q,
			Quote{Open: `"`, Close: `"`, Prefixes: prefixes, Escapes: true},
			Quote{Open: `'`, Close: `'`, Prefixes: prefixes, Escapes: true},
		)
	}
	return q
}

const (
	cKeywords = `auto break case const continue default do else enum extern
		for goto if inline register restrict return sizeof static struct
		switch typedef union volatile while _Alignas _Alignof _Atomic _Generic
		_Noreturn _Static_assert _Thread_local alignas alignof static_assert
		thread_local typeof typeof_unqual constexpr`
	cTypes = `char double float int long short signed unsigned void bool
		_Bool _Complex size_t ssize_t ptrdiff_t intptr_t uintptr_t intmax_t
		uintmax_t int8_t int16_t int32_t int64_t uint8_t uint16_t uint32_t
		uint64_t wchar_t FILE va_list`
	cppKeywords = `and and_eq asm bitand bitor catch class compl concept
		consteval constinit const_cast co_await co_return co_yield decltype
		delete dynamic_cast explicit export final friend mutable namespace new
		noexcept not not_eq operator or or_eq override private protected public
		reinterpret_cast requires static_cast template this throw try typeid
		typename using virtual xor xor_eq`
)

// cLang returns C.
func cLang() *Language {
	return &Language{
		Name:    "c",
		Aliases: []string{"h"},
		Lexer: &CLike{
			Words:         wordKinds(Keyword, cKeywords, BuiltinType, cTypes, Primitive, "true false NULL nullptr"),
			Declarators:   map[string]Kind{"struct": Type, "enum": Type, "union": Type},
			LineComments:  []string{"//"},
			BlockComment:  slashComments,
			Quotes:        cQuotes("L", "u", "U", "u8"),
			Preproc:       true,
			CapsConstants: true,
		},
	}
}

// cppLang returns C++.
func cppLang() *Language {
	return &Language{
		Name:    "cpp",
		Aliases: []string{"c++", "cc", "cxx", "hpp", "hh", "hxx", "h++", "ino"},
		Lexer: &CLike{
			Words: wordKinds(Keyword, cKeywords+" "+cppKeywords,
				BuiltinType, cTypes+" char8_t char16_t char32_t", Primitive, "true false NULL nullptr"),
			Declarators: map[string]Kind{
				"class": Type, "struct": Type, "enum": Type, "union": Type, "namespace": Type, "concept": Type,
			},
			LineComments:  []string{"//"},
			BlockComment:  slashComments,
			Quotes:        cQuotes("L", "u", "U", "u8"),
			Preproc:       true,
			CapsConstants: true,
			Special:       cppSpecial,
		},
	}
}

// cppSpecial lexes C++ raw strings, such as R"x(...)x".
func cppSpecial(s *Scanner) bool {
	rest := s.Rest()
	i := strings.Index(rest[:min(len(rest), 4)], `R"`)
	if i < 0 || !slices.Contains([]string{"", "L", "u", "U", "u8"}, rest[:i]) {
		return false
	}
	open := strings.IndexByte(rest[i+2:], '(')
	if open < 0 || open > 16 || strings.ContainsAny(rest[i+2:i+2+open], ` ()\"`) {
		return false
	}
	s.Emit(String, i+3+open)
	s.Delimit(String, ")"+rest[i+2:i+2+open]+`"`)
	return true
}

// csharpLang returns C#.
func csharpLang() *Language {
	return &Language{
		Name:    "csharp",
		Aliases: []string{"c#", "cs"},
		Lexer: &CLike{
			Words: wordKinds(
				Keyword, `abstract as base break case catch checked class const continue
					default delegate do else enum event explicit extern finally
					fixed for foreach goto if implicit in interface internal is lock
					namespace new operator out override params private protected
					public readonly ref return sealed sizeof stackalloc static
					struct switch this throw try typeof unchecked unsafe using
					virtual volatile while async await init nameof partial record
					required var when where with yield`,
				BuiltinType, `bool byte char decimal double dynamic float int long nint
					nuint object sbyte short string uint ulong ushort void`,
				Primitive, "true false null"),
			Declarators: map[string]Kind{
				"class": Type, "struct": Type, "interface": Type, "enum": Type, "record": Type, "namespace": Type,
			},
			LineComments: []string{"//"},
			BlockComment: slashComments,
			Quotes: []Quote{
				{Open: `$"""`, Close: `"""`, Multiline: true, Interp: "{", InterpClose: "}"},
				{Open: `"""`, Close: `"""`, Multiline: true},
				{Open: `$@"`, Close: `"`, Doubled: true, Multiline: true, Interp: "{", InterpClose: "}"},
				{Open: `@$"`, Close: `"`, Doubled: true, Multiline: true, Interp: "{", InterpClose: "}"},
				{Open: `@"`, Close: `"`, Doubled: true, Multiline: true},
				{Open: `$"`, Close: `"`, Escapes: true, Interp: "{", InterpClose: "}"},
				{Open: `"`, Close: `"`, Escapes: true},
				{Open: `'`, Close: `'`, Escapes: true},
			},
			Preproc:       true,
			CapitalTypes:  true,
			CapsConstants: true,
		},
	}
}

// javaLang returns Java.
func javaLang() *Language {
	return &Language{
		Name: "java",
		Lexer: &CLike{
			Words: wordKinds(
				Keyword, `abstract assert break case catch class const continue default do
					else enum extends final finally for goto if implements import
					instanceof interface native new package permits private
					protected public record return sealed static strictfp super
					switch synchronized this throw throws transient try var when
					while yield`,
				BuiltinType, "boolean byte char double float int long short void",
				Primitive, "true false null"),
			MemberWords: []string{"class", "this"},
			Declarators: map[string]Kind{
				"class": Type, "interface": Type, "enum": Type, "record": Type,
			},
			LineComments: []string{"//"},
			BlockComment: slashComments,
			Quotes: []Quote{
				{Open: `"""`, Close: `"""`, Escapes: true, Multiline: true},
				{Open: `"`, Close: `"`, Escapes: true},
				{Open: `'`, Close: `'`, Escapes: true},
			},
			IdentChars:    "$",
			Decorators:    true,
			CapitalTypes:  true,
			CapsConstants: true,
		},
	}
}

// kotlinLang returns Kotlin.
func kotlinLang() *Language {
	return &Language{
		Name:    "kotlin",
		Aliases: []string{"kt", "kts"},
		Lexer: &CLike{
			Words: wordKinds(
				Keyword, `as break class continue do else for fun if in interface is
					object package return super this throw try typealias typeof val
					var when while by catch constructor dynamic finally import init
					where abstract annotation companion const crossinline enum
					external final infix inline inner internal lateinit noinline
					open operator out override private protected public reified
					sealed suspend tailrec vararg`,
				Primitive, "true false null"),
			Declarators: map[string]Kind{
				"fun": Function, "class": Type, "interface": Type, "object": Type, "typealias": Type,
			},
			LineComments:   []string{"//"},
			BlockComment:   slashComments,
			NestedComments: true,
			Quotes: []Quote{
				{Open: `"""`, Close: `"""`, Multiline: true, Interp: "${", InterpClose: "}", Sigil: '$'},
				{Open: `"`, Close: `"`, Escapes: true, Interp: "${", InterpClose: "}", Sigil: '$'},
				{Open: `'`, Close: `'`, Escapes: true},
			},
			Decorators:    true,
			CapitalTypes:  true,
			CapsConstants: true,
		},
	}
}

// swiftLang returns Swift.
func swiftLang() *Language {
	return &Language{
		Name: "swift",
		Lexer: &CLike{
			Words: wordKinds(
				Keyword, `associatedtype class deinit enum extension fileprivate
					func import init inout internal let open operator private
					precedencegroup protocol public rethrows static struct subscript
					typealias var break case catch continue default defer do else
					fallthrough for guard if in repeat return throw switch where
					while as is try await async throws super self Self some any
					macro nonisolated isolated consume borrowing consuming
					convenience dynamic final indirect lazy mutating nonmutating
					optional override required unowned weak willSet didSet get set
					package actor`,
				Primitive, "true false nil"),
			MemberWords: []string{"self", "init"},
			Declarators: map[string]Kind{
				"func": Function, "class": Type, "struct": Type, "enum": Type, "protocol": Type,
				"extension": Type, "typealias": Type, "actor": Type,
			},
			LineComments:   []string{"//"},
			BlockComment:   slashComments,
			NestedComments: true,
			Quotes: []Quote{
				{Open: `"""`, Close: `"""`, Escapes: true, Multiline: true, Interp: `\(`, InterpClose: ")"},
				{Open: `"`, Close: `"`, Escapes: true, Interp: `\(`, InterpClose: ")"},
			},
			Decorators:   true,
			CapitalTypes: true,
			Special:      swiftSpecial,
		},
	}
}

// swiftSpecial lexes Swift's raw strings, such as #"..."#, and directives,
// such as #if and #available.
func swiftSpecial(s *Scanner) bool {
	if s.Peek(0) != '#' {
		return false
	}
	rest := s.Rest()
	// Count the hashes of a raw string's delimiter, but only so many, so
	// that a long run of them stays linear to lex.
	n := 1
	for n < len(rest) && n < 16 && rest[n] == '#' {
		n++
	}
	switch {
	case strings.HasPrefix(rest[n:], `"""`):
		s.Emit(String, n+3)
		s.Delimit(String, `"""`+rest[:n])
		return true
	case strings.HasPrefix(rest[n:], `"`):
		s.Emit(String, n+1)
		s.Delimit(String, `"`+rest[:n])
		return true
	case n == 1 && IdentStart(rest, 1, ""):
		s.Emit(Preproc, IdentEnd(rest, 1, ""))
		return true
	}
	return false
}

// scalaLang returns Scala.
func scalaLang() *Language {
	interpolators := []string{"s", "f", "raw", "json", "sql", "q"}
	return &Language{
		Name:    "scala",
		Aliases: []string{"sc", "sbt"},
		Lexer: &CLike{
			Words: wordKinds(
				Keyword, `abstract case catch class def do else enum export extends
					final finally for forSome given if implicit import lazy match new
					object override package private protected return sealed super
					then this throw trait try type val var while with yield using
					extension inline opaque open transparent derives end infix`,
				Primitive, "true false null"),
			Declarators: map[string]Kind{
				"def": Function, "class": Type, "trait": Type, "object": Type, "type": Type, "enum": Type,
			},
			LineComments:   []string{"//"},
			BlockComment:   slashComments,
			NestedComments: true,
			Quotes: []Quote{
				{Open: `"""`, Close: `"""`, Prefixes: interpolators, Multiline: true, Interp: "${", InterpClose: "}", Sigil: '$'},
				{Open: `"`, Close: `"`, Prefixes: interpolators, Escapes: true, Interp: "${", InterpClose: "}", Sigil: '$'},
				{Open: `"""`, Close: `"""`, Multiline: true},
				{Open: `"`, Close: `"`, Escapes: true},
				{Open: `'`, Close: `'`, Escapes: true},
			},
			Decorators:   true,
			CapitalTypes: true,
		},
	}
}

// dartLang returns Dart.
func dartLang() *Language {
	interp := func(q Quote) Quote {
		q.Interp, q.InterpClose, q.Sigil = "${", "}", '$'
		return q
	}
	return &Language{
		Name: "dart",
		Lexer: &CLike{
			Words: wordKinds(
				Keyword, `abstract as assert async await base break case catch class
					const continue covariant default deferred do dynamic else enum
					export extends extension external factory final finally for get
					hide if implements import in interface is late library mixin new
					of on operator part required rethrow return sealed set show
					static super switch sync this throw try typedef var when while
					with yield`,
				BuiltinType, "bool double int num void",
				Primitive, "true false null"),
			Declarators: map[string]Kind{
				"class": Type, "enum": Type, "mixin": Type, "extension": Type, "typedef": Type,
			},
			LineComments:   []string{"//"},
			BlockComment:   slashComments,
			NestedComments: true,
			Quotes: []Quote{
				{Open: `'''`, Close: `'''`, Prefixes: []string{"r"}, Multiline: true},
				{Open: `"""`, Close: `"""`, Prefixes: []string{"r"}, Multiline: true},
				{Open: `'`, Close: `'`, Prefixes: []string{"r"}},
				{Open: `"`, Close: `"`, Prefixes: []string{"r"}},
				interp(Quote{Open: `'''`, Close: `'''`, Escapes: true, Multiline: true}),
				interp(Quote{Open: `"""`, Close: `"""`, Escapes: true, Multiline: true}),
				interp(Quote{Open: `'`, Close: `'`, Escapes: true}),
				interp(Quote{Open: `"`, Close: `"`, Escapes: true}),
			},
			IdentChars:   "$",
			Decorators:   true,
			CapitalTypes: true,
		},
	}
}

// goLang returns Go.
func goLang() *Language {
	return &Language{
		Name:    "go",
		Aliases: []string{"golang"},
		Lexer: &CLike{
			Words: wordKinds(
				Keyword, `break case chan const continue default defer else
					fallthrough for func go goto if import interface map package range
					return select struct switch type var`,
				BuiltinType, `bool byte complex64 complex128 error float32 float64 int int8
					int16 int32 int64 rune string uint uint8 uint16 uint32 uint64
					uintptr any comparable`,
				BuiltinFunction, `append cap clear close complex copy delete imag len make
					max min new panic print println real recover`,
				Primitive, "true false nil",
				Constant, "iota"),
			Declarators:  map[string]Kind{"func": Function, "type": Type},
			LineComments: []string{"//"},
			Directives:   []string{"//go:", "//line "},
			BlockComment: slashComments,
			Quotes: []Quote{
				{Open: `"`, Close: `"`, Escapes: true},
				{Open: "`", Close: "`", Multiline: true},
				{Open: `'`, Close: `'`, Escapes: true},
			},
		},
	}
}

// rustLang returns Rust.
func rustLang() *Language {
	return &Language{
		Name:    "rust",
		Aliases: []string{"rs"},
		Lexer: &CLike{
			Words: wordKinds(
				Keyword, `as async await break const continue crate dyn else enum
					extern fn for if impl in let loop match mod move mut pub ref
					return self Self static struct super trait type union unsafe use
					where while yield macro_rules`,
				BuiltinType, `bool char str u8 u16 u32 u64 u128 usize i8 i16 i32 i64 i128
					isize f32 f64`,
				Constant, "Some None Ok Err",
				Primitive, "true false"),
			MemberWords: []string{"await"},
			Declarators: map[string]Kind{
				"fn": Function, "struct": Type, "enum": Type, "trait": Type, "type": Type, "union": Type,
			},
			LineComments:   []string{"//"},
			BlockComment:   slashComments,
			NestedComments: true,
			Quotes: []Quote{
				{Open: `"`, Close: `"`, Prefixes: []string{"b", "c"}, Escapes: true, Multiline: true},
				{Open: `"`, Close: `"`, Escapes: true, Multiline: true},
			},
			Macros:        true,
			CapitalTypes:  true,
			CapsConstants: true,
			Special:       rustSpecial,
		},
	}
}

// rustSpecial lexes Rust's raw strings, characters, lifetimes, and
// attributes.
func rustSpecial(s *Scanner) bool {
	rest := s.Rest()
	switch b := rest[0]; b {
	case 'r', 'b', 'c':
		// A byte character, such as b'x'.
		if b == 'b' && len(rest) > 1 && rest[1] == '\'' {
			if n := charLiteral(rest[1:]); n > 0 {
				s.Emit(String, 1+n)
				return true
			}
		}
		i := 1
		if b != 'r' {
			if len(rest) < 2 || rest[1] != 'r' {
				return false
			}
			i = 2
		}
		j := i
		for j < len(rest) && rest[j] == '#' {
			j++
		}
		if j >= len(rest) || rest[j] != '"' {
			return false
		}
		s.Emit(String, j+1)
		s.Delimit(String, `"`+rest[i:j])
		return true
	case '\'':
		if n := charLiteral(rest); n > 0 {
			s.Emit(String, n)
			return true
		}
		if IdentStart(rest, 1, "") {
			s.Emit(Attribute, IdentEnd(rest, 1, ""))
			return true
		}
	case '#':
		if strings.HasPrefix(rest, "#[") || strings.HasPrefix(rest, "#![") {
			s.Emit(Decorator, bracketEnd(rest, strings.IndexByte(rest, '[')))
			return true
		}
	}
	return false
}

// charLiteral returns the length of the character literal, such as 'a' or
// '\n', at the start of s, or 0 if none is there.
func charLiteral(s string) int {
	if len(s) < 3 {
		return 0
	}
	if s[1] == '\\' {
		if i := strings.IndexByte(s[2:min(len(s), 14)], '\''); i > 0 || (i == 0 && len(s) > 3 && s[3] == '\'') {
			if i == 0 {
				return 4
			}
			return i + 3
		}
		return 0
	}
	n := runeLen(s, 1)
	if len(s) > 1+n && s[1+n] == '\'' && s[1] != '\'' {
		return 2 + n
	}
	return 0
}

// bracketEnd returns the offset in s just past the ']' that closes the '['
// at i, or len(s) if the line does not close it.
func bracketEnd(s string, i int) int {
	depth := 0
	for j := i; j < len(s); j++ {
		switch s[j] {
		case '[':
			depth++
		case ']':
			if depth--; depth == 0 {
				return j + 1
			}
		case '"':
			if k := strings.IndexByte(s[j+1:], '"'); k >= 0 {
				j += k + 1
			}
		}
	}
	return len(s)
}

// zigLang returns Zig.
func zigLang() *Language {
	return &Language{
		Name: "zig",
		Lexer: &CLike{
			Words: wordKinds(
				Keyword, `addrspace align allowzero and anyframe anytype asm async
					await break callconv catch comptime const continue defer else enum
					errdefer error export extern fn for if inline linksection noalias
					noinline nosuspend opaque or orelse packed pub resume return
					struct suspend switch test threadlocal try union unreachable
					usingnamespace var volatile while`,
				BuiltinType, `i8 i16 i32 i64 i128 isize u8 u16 u32 u64 u128 usize f16 f32
					f64 f80 f128 bool void noreturn type anyerror anyopaque
					comptime_int comptime_float c_char c_short c_ushort c_int c_uint
					c_long c_ulong c_longlong c_ulonglong c_longdouble`,
				Primitive, "true false null undefined"),
			Declarators:  map[string]Kind{"fn": Function},
			LineComments: []string{"//"},
			Quotes:       cQuotes(),
			CapitalTypes: true,
			Special:      zigSpecial,
		},
	}
}

// zigSpecial lexes Zig's multiline string lines, builtin functions such as
// @import, and integer types of any width, such as u7.
func zigSpecial(s *Scanner) bool {
	rest := s.Rest()
	switch b := rest[0]; {
	case strings.HasPrefix(rest, `\\`):
		s.EmitRest(String)
		return true
	case b == '@' && IdentStart(rest, 1, ""):
		s.Emit(BuiltinFunction, IdentEnd(rest, 1, ""))
		return true
	case (b == 'u' || b == 'i') && len(rest) > 1 && IsDigit(rest[1]):
		end := IdentEnd(rest, 0, "")
		for i := 1; i < end; i++ {
			if !IsDigit(rest[i]) {
				return false
			}
		}
		s.Emit(BuiltinType, end)
		return true
	}
	return false
}

// protobufLang returns Protocol Buffers.
func protobufLang() *Language {
	return &Language{
		Name:    "protobuf",
		Aliases: []string{"proto", "proto3"},
		Lexer: &CLike{
			Words: wordKinds(
				Keyword, `syntax edition package import option message enum service
					rpc returns stream oneof map reserved extend extensions to max
					repeated optional required weak public group`,
				BuiltinType, `double float int32 int64 uint32 uint64 sint32 sint64 fixed32
					fixed64 sfixed32 sfixed64 bool string bytes`,
				Primitive, "true false"),
			Declarators: map[string]Kind{
				"message": Type, "enum": Type, "service": Type, "rpc": Function,
			},
			LineComments: []string{"//"},
			BlockComment: slashComments,
			Quotes:       cQuotes(),
			CapitalTypes: true,
		},
	}
}

// graphqlLang returns GraphQL.
func graphqlLang() *Language {
	return &Language{
		Name:    "graphql",
		Aliases: []string{"gql", "graphqls"},
		Lexer: &CLike{
			Words: wordKinds(
				Keyword, `query mutation subscription fragment on type interface
					union enum input scalar schema extend directive implements
					repeatable`,
				Primitive, "true false null"),
			Declarators: map[string]Kind{
				"type": Type, "interface": Type, "union": Type, "enum": Type, "input": Type,
				"scalar": Type, "fragment": Type, "query": Function, "mutation": Function,
				"subscription": Function,
			},
			LineComments: []string{"#"},
			Quotes: []Quote{
				{Open: `"""`, Close: `"""`, Multiline: true},
				{Open: `"`, Close: `"`, Escapes: true},
			},
			Sigils:       "$",
			Decorators:   true,
			CapitalTypes: true,
		},
	}
}

// hclLang returns HCL, the language of Terraform.
func hclLang() *Language {
	return &Language{
		Name:    "hcl",
		Aliases: []string{"terraform", "tf", "tfvars", "nomad"},
		Lexer: &CLike{
			Words: wordKinds(
				Keyword, `resource data variable output module provider locals
					terraform for in if else endif endfor dynamic content moved import
					check removed backend required_providers lifecycle provisioner
					connection`,
				BuiltinType, "string number bool list map set object tuple any",
				Primitive, "true false null"),
			LineComments: []string{"#", "//"},
			BlockComment: slashComments,
			Quotes: []Quote{
				{Open: `"`, Close: `"`, Escapes: true, Interp: "${", InterpClose: "}"},
			},
			AssignProperties: true,
			Special:          hclSpecial,
		},
	}
}

// hclSpecial lexes the start of a heredoc, such as <<EOT or <<-EOT.
func hclSpecial(s *Scanner) bool {
	if !s.HasPrefix("<<") {
		return false
	}
	n := 2
	if s.Peek(2) == '-' {
		n = 3
	}
	rest := s.Rest()
	if !IdentStart(rest, n, "") {
		return false
	}
	end := IdentEnd(rest, n, "")
	s.Emit(Operator, n)
	s.Emit(String, end-n)
	s.Heredoc(String, rest[n:end], true, false)
	return true
}
