package syntax

import "testing"

func TestLua(t *testing.T) {
	testLanguage(t, "lua", []string{"luau", "init.lua"}, luaTests)
}

var luaTests = []kindCase{
	{name: "functions", src: "local function add(a, b) return a + b end\nfunction M.sub(a, b) return a - b end\nfunction obj:method() return self end\nlocal f = function(...) end", want: []Token{
		tok(Keyword, "local"), tok(Keyword, "function"), tok(Function, "add"), tok(Keyword, "return"), tok(Keyword, "end"),
		tok(Keyword, "function"), tok(Variable, "M"), tok(Punctuation, "."), tok(Function, "sub"), tok(Keyword, "function"),
		tok(Function, "method"), tok(BuiltinVariable, "self"), tok(Keyword, "local"), tok(Variable, "f"), tok(Keyword, "function"),
		tok(Operator, "..."),
	}},
	{name: "control flow", src: "if a then b() elseif c then d() else e() end\nfor i = 1, 10 do end\nfor k, v in pairs(t) do end\nwhile x do break end\nrepeat until y\ngoto continue\n::continue::", want: []Token{
		tok(Keyword, "if"), tok(Keyword, "then"), tok(Keyword, "elseif"), tok(Keyword, "else"), tok(Keyword, "end"),
		tok(Keyword, "for"), tok(Keyword, "do"), tok(Keyword, "for"), tok(Keyword, "in"), tok(BuiltinFunction, "pairs"),
		tok(Keyword, "while"), tok(Keyword, "break"), tok(Keyword, "repeat"), tok(Keyword, "until"), tok(Keyword, "goto"),
		tok(Punctuation, "::"),
	}},
	{name: "strings", src: "s = 'a\\n' .. \"b\\\"c\" .. [[long\nstring]] .. [==[with ]] inside]==]", want: []Token{
		tok(String, "'a"), tok(StringEscape, `\n`), tok(String, "'"), tok(Operator, ".."), tok(String, `"b`),
		tok(StringEscape, `\"`), tok(String, `c"`), tok(String, "[[long"), tok(String, "string]]"),
		tok(String, "[==[with ]] inside]==]"),
	}},
	{name: "comments", src: "-- line\n--[[ block\ncomment ]]\n--[==[ level ]] still ]==] x = 1\n--[ not long", want: []Token{
		tok(Comment, "-- line"), tok(Comment, "--[[ block"), tok(Comment, "comment ]]"),
		tok(Comment, "--[==[ level ]] still ]==]"), tok(Variable, "x"), tok(Comment, "--[ not long"),
	}},
	{name: "builtins and primitives", src: "print(type(x), tostring(nil), true, false, select('#', ...), setmetatable(t, mt))", want: []Token{
		tok(BuiltinFunction, "print"), tok(BuiltinFunction, "type"), tok(BuiltinFunction, "tostring"), tok(Primitive, "nil"), tok(Primitive, "true"),
		tok(Primitive, "false"), tok(BuiltinFunction, "select"), tok(String, "'#'"), tok(Operator, "..."), tok(BuiltinFunction, "setmetatable"),
	}},
	{name: "tables and members", src: "local t = { x = 1, ['y'] = 2 }\nt.x = string.format('%d', 1)\nmath.floor(1.5)\nobj:method(1)", want: []Token{
		tok(Keyword, "local"), tok(String, "'y'"), tok(Property, "x"), tok(BuiltinVariable, "string"), tok(Function, "format"),
		tok(BuiltinVariable, "math"), tok(Function, "floor"), tok(Function, "method"),
	}},
	{name: "operators", src: "a = b ~= c and not d or e\nf = g // h\ni = #j .. k", want: []Token{
		tok(Operator, "~="), tok(Keyword, "and"), tok(Keyword, "not"), tok(Keyword, "or"), tok(Operator, "//"),
		tok(Operator, ".."),
	}},
	{name: "numbers", src: "n = 0x1F + 1e3 + 3.14 + 0xA.8p1", want: []Token{
		tok(Number, "0x1F"), tok(Number, "1e3"), tok(Number, "3.14"), tok(Number, "0xA.8p1"),
	}},
	{name: "unclosed long string", src: "s = [[open\nstill", open: true, want: []Token{
		tok(String, "[[open"), tok(String, "still"),
	}},
	{name: "builtin variables", src: "local g = _G\nprint(_ENV, self, string.len(\"x\"))", want: []Token{
		tok(Keyword, "local"), tok(Variable, "g"), tok(BuiltinVariable, "_G"), tok(BuiltinFunction, "print"),
		tok(BuiltinVariable, "_ENV"), tok(BuiltinVariable, "self"), tok(BuiltinVariable, "string"), tok(Function, "len"),
	}},
	{name: "method declaration", src: "function Account:deposit(v) end", want: []Token{
		tok(Keyword, "function"), tok(Variable, "Account"), tok(Punctuation, ":"), tok(Function, "deposit"),
		tok(Variable, "v"), tok(Keyword, "end"),
	}},
}
