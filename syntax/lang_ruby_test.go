package syntax

import "testing"

func TestRuby(t *testing.T) {
	testLanguage(t, "ruby", []string{"rb", "gemspec", "rake", "Gemfile", "Rakefile", "Vagrantfile", "app.rb"}, rubyTests)
}

var rubyTests = []kindCase{
	{name: "classes and methods", src: "class Foo < Bar\n  include Comparable\n  attr_reader :name\n  def initialize(name)\n    @name = name\n  end\n  def valid? = !@name.nil?\n  def save!; end\nend\nmodule Util; end", want: []Token{
		tok(Keyword, "class"), tok(Type, "Foo"), tok(Operator, "<"), tok(Type, "Bar"), tok(Builtin, "include"),
		tok(Type, "Comparable"), tok(Builtin, "attr_reader"), tok(String, ":name"), tok(Keyword, "def"),
		tok(Function, "initialize"), tok(Variable, "@name"), tok(Keyword, "end"), tok(Keyword, "def"),
		tok(Function, "valid?"), tok(Operator, "!"), tok(Variable, "@name"), tok(Property, "nil?"), tok(Keyword, "def"),
		tok(Function, "save!"), tok(Keyword, "end"), tok(Keyword, "end"), tok(Keyword, "module"), tok(Type, "Util"),
	}},
	{name: "control flow", src: "if x then y elsif z then w else v end\nunless a; end\nwhile b do end\nuntil c; end\ncase n when 1 then :one else :other end\nbegin; rescue E => e; retry; ensure; end\nitems.each do |i| next if i; redo; end\nyield; return; break", want: []Token{
		tok(Keyword, "if"), tok(Keyword, "then"), tok(Keyword, "elsif"), tok(Keyword, "else"), tok(Keyword, "end"),
		tok(Keyword, "unless"), tok(Keyword, "while"), tok(Keyword, "do"), tok(Keyword, "until"), tok(Keyword, "case"),
		tok(Keyword, "when"), tok(String, ":one"), tok(String, ":other"), tok(Keyword, "begin"), tok(Keyword, "rescue"),
		tok(Type, "E"), tok(Operator, "=>"), tok(Keyword, "retry"), tok(Keyword, "ensure"), tok(Property, "each"),
		tok(Keyword, "do"), tok(Operator, "|"), tok(Keyword, "next"), tok(Keyword, "redo"), tok(Keyword, "yield"),
		tok(Keyword, "return"), tok(Keyword, "break"),
	}},
	{name: "strings and interpolation", src: "s = \"Hi #{name}, #{a + b}\\n\"\nt = 'single #{x} \\' q'\nu = `ls #{dir}`", want: []Token{
		tok(String, `"Hi `), tok(StringEscape, "#{"), tok(Variable, "name"), tok(StringEscape, "}"), tok(String, ", "),
		tok(StringEscape, "#{"), tok(Variable, "a"), tok(Operator, "+"), tok(Variable, "b"), tok(StringEscape, "}"),
		tok(StringEscape, `\n`), tok(String, `"`), tok(String, "'single #{x} "), tok(StringEscape, `\'`),
		tok(String, " q'"), tok(String, "`ls "), tok(StringEscape, "#{"), tok(Variable, "dir"), tok(StringEscape, "}"),
	}},
	{name: "strings span lines", src: "s = \"one\ntwo #{x}\"\ny = 1", want: []Token{
		tok(String, `"one`), tok(String, "two "), tok(StringEscape, "#{"), tok(String, `"`), tok(Variable, "y"),
	}},
	{name: "heredocs", src: "sql = <<~SQL\n  SELECT #{cols}\n  FROM t\nSQL\nmsg = <<-EOS\n  text\n  EOS\nraw = <<'END'\nno #{interp}\nEND\nputs sql", want: []Token{
		tok(Operator, "<<~"), tok(String, "SQL"), tok(String, "  SELECT #{cols}"), tok(String, "  FROM t"),
		tok(String, "SQL"), tok(Operator, "<<-"), tok(String, "EOS"), tok(String, "  text"), tok(String, "  EOS"),
		tok(Operator, "<<"), tok(String, "'END'"), tok(String, "no #{interp}"), tok(String, "END"), tok(Builtin, "puts"),
	}},
	{name: "appending is not a heredoc", src: "list << item\nlist << other", want: []Token{
		tok(Variable, "list"), tok(Operator, "<<"), tok(Variable, "item"), tok(Variable, "list"),
	}},
	{name: "symbols and variables", src: "h = { a: 1, :b => 2 }\n$stdout.puts(@@count, @ivar)\nv = x ? y : z", want: []Token{
		tok(Variable, "a"), tok(Punctuation, ":"), tok(String, ":b"), tok(Operator, "=>"), tok(Variable, "$stdout"),
		tok(Variable, "@@count"), tok(Variable, "@ivar"), tok(Operator, "?"), tok(Punctuation, ":"), tok(Variable, "z"),
	}},
	{name: "regular expressions", src: "if s =~ /^\\d+$/i then end\nx = a / b", want: []Token{
		tok(Operator, "=~"), tok(Regex, `/^\d+$/i`), tok(Keyword, "then"), tok(Variable, "a"), tok(Operator, "/"),
	}},
	{name: "block comments", src: "=begin\nblock comment\n=end\nx = 1", want: []Token{
		tok(Comment, "=begin"), tok(Comment, "block comment"), tok(Comment, "=end"), tok(Variable, "x"),
	}},
	{name: "primitives and constants", src: "x = nil || true && false\nMAX = 10\nFoo::BAR", want: []Token{
		tok(Primitive, "nil"), tok(Primitive, "true"), tok(Primitive, "false"), tok(Constant, "MAX"), tok(Type, "Foo"),
		tok(Punctuation, "::"), tok(Constant, "BAR"),
	}},
	{name: "builtins and calls", src: "require 'json'\nrequire_relative 'x'\nraise ArgumentError, 'bad'\nputs [1].map(&:to_s).join(', ')", want: []Token{
		tok(Builtin, "require"), tok(String, "'json'"), tok(Builtin, "require_relative"), tok(Builtin, "raise"),
		tok(Type, "ArgumentError"), tok(Builtin, "puts"), tok(Function, "map"), tok(Function, "join"),
	}},
	{name: "comments", src: "# comment\nx = 1 # trailing", want: []Token{
		tok(Comment, "# comment"), tok(Variable, "x"), tok(Comment, "# trailing"),
	}},
	{name: "numbers", src: "n = 1_000 + 0x1f + 0b1 + 1.5e3 + 2r", want: []Token{
		tok(Number, "1_000"), tok(Number, "0x1f"), tok(Number, "0b1"), tok(Number, "1.5e3"), tok(Number, "2r"),
	}},
	{name: "unclosed heredoc", src: "x = <<~EOS\nbody", open: true, want: []Token{
		tok(String, "EOS"), tok(String, "body"),
	}},
}
