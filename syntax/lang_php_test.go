package syntax

import "testing"

func TestPHP(t *testing.T) {
	testLanguage(t, "php", []string{"phtml", "php8", "index.php"}, phpTests)
}

var phpTests = []kindCase{
	{name: "tags and functions", src: "<?php\nnamespace App\\Http;\nuse Foo\\Bar;\nfunction add(int $a, int $b): int {\n    return $a + $b;\n}\n?>\n<?= $x ?>", want: []Token{
		tok(Preproc, "<?php"), tok(Keyword, "namespace"), tok(Type, "App"), tok(Type, "Http"), tok(Keyword, "use"),
		tok(Type, "Foo"), tok(Type, "Bar"), tok(Keyword, "function"), tok(Function, "add"), tok(BuiltinType, "int"),
		tok(Variable, "$a"), tok(BuiltinType, "int"), tok(Variable, "$b"), tok(Keyword, "return"), tok(Variable, "$a"),
		tok(Operator, "+"), tok(Preproc, "?>"), tok(Preproc, "<?="), tok(Variable, "$x"), tok(Preproc, "?>"),
	}},
	{name: "classes", src: "final class User extends Model implements JsonSerializable {\n    public function __construct(private readonly string $name) {}\n    public static function make(): static { return new static(); }\n}\ninterface I {}\ntrait T {}\nenum Suit: string { case Hearts = 'H'; }", want: []Token{
		tok(Keyword, "final"), tok(Keyword, "class"), tok(Type, "User"), tok(Keyword, "extends"), tok(Type, "Model"),
		tok(Keyword, "implements"), tok(Type, "JsonSerializable"), tok(Keyword, "public"), tok(Keyword, "function"),
		tok(Function, "__construct"), tok(Keyword, "private"), tok(Keyword, "readonly"), tok(BuiltinType, "string"),
		tok(Variable, "$name"), tok(Keyword, "static"), tok(Function, "make"), tok(Keyword, "static"),
		tok(Keyword, "return"), tok(Keyword, "new"), tok(Keyword, "static"), tok(Keyword, "interface"), tok(Type, "I"),
		tok(Keyword, "trait"), tok(Type, "T"), tok(Keyword, "enum"), tok(Type, "Suit"), tok(Keyword, "case"),
		tok(Type, "Hearts"), tok(String, "'H'"),
	}},
	{name: "members and static access", src: "$this->name = $obj?->get()['k'];\nself::MAX;\nparent::__construct();\n$x::$y;", want: []Token{
		tok(Variable, "$this"), tok(Operator, "->"), tok(Property, "name"), tok(Variable, "$obj"), tok(Function, "get"),
		tok(String, "'k'"), tok(Type, "self"), tok(Punctuation, "::"), tok(Constant, "MAX"), tok(Type, "parent"),
		tok(Function, "__construct"), tok(Variable, "$x"), tok(Variable, "$y"),
	}},
	{name: "strings", src: "$s = \"Hi $name\\n\";\n$t = 'raw $x \\'q\\'';\n$u = \"span\nlines\";", want: []Token{
		tok(String, `"Hi `), tok(StringEscape, "$name"), tok(StringEscape, `\n`), tok(String, `"`), tok(String, "'raw $x "),
		tok(StringEscape, `\'`), tok(String, "q"), tok(StringEscape, `\'`), tok(String, "'"), tok(String, `"span`),
		tok(String, `lines"`),
	}},
	{name: "heredocs and nowdocs", src: "$h = <<<EOT\nBody $x\n  EOT;\n$n = <<<'NOW'\nraw\nNOW;\necho 1;", want: []Token{
		tok(Operator, "<<<"), tok(String, "EOT"), tok(String, "Body $x"), tok(String, "  EOT;"), tok(Operator, "<<<"),
		tok(String, "'NOW'"), tok(String, "raw"), tok(String, "NOW;"), tok(Keyword, "echo"),
	}},
	{name: "comments and attributes", src: "// line\n# hash\n/* block */\n#[Route('/x', methods: ['GET'])]\nfunction f() {}", want: []Token{
		tok(Comment, "// line"), tok(Comment, "# hash"), tok(Comment, "/* block */"),
		tok(Decorator, "#[Route('/x', methods: ['GET'])]"), tok(Keyword, "function"),
	}},
	{name: "keywords ignore case", src: "IF ($a) { ECHO 'x'; } ELSE { RETURN NULL; }\nforeach ($xs as $k => $v) {}\n$r = match ($x) { default => 1 };", want: []Token{
		tok(Keyword, "IF"), tok(Keyword, "ECHO"), tok(Keyword, "ELSE"), tok(Keyword, "RETURN"), tok(Primitive, "NULL"),
		tok(Keyword, "foreach"), tok(Keyword, "as"), tok(Operator, "=>"), tok(Keyword, "match"), tok(Keyword, "default"),
	}},
	{name: "primitives and constants", src: "$a = true; $b = FALSE; define('MAX', 1); echo PHP_EOL;", want: []Token{
		tok(Primitive, "true"), tok(Primitive, "FALSE"), tok(Function, "define"), tok(Keyword, "echo"),
		tok(Constant, "PHP_EOL"),
	}},
	{name: "numbers", src: "$n = 1_000 + 0x1F + 0b11 + 1.5e3 + 017;", want: []Token{
		tok(Number, "1_000"), tok(Number, "0x1F"), tok(Number, "0b11"), tok(Number, "1.5e3"), tok(Number, "017"),
	}},
	{name: "unclosed heredoc", src: "$h = <<<EOT\nbody", open: true, want: []Token{
		tok(String, "EOT"), tok(String, "body"),
	}},
}
