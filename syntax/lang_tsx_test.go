package syntax

import "testing"

func TestTSX(t *testing.T) {
	testLanguage(t, "tsx", []string{"App.tsx"}, tsxTests)
}

var tsxTests = []kindCase{
	{name: "components", src: "export const App: React.FC<Props> = ({ items }) => <ul>{items.map(i => <li key={i}>{i}</li>)}</ul>;", want: []Token{
		tok(Keyword, "export"), tok(Keyword, "const"), tok(Type, "App"), tok(Type, "React"), tok(Constant, "FC"),
		tok(Operator, "<"), tok(Type, "Props"), tok(Operator, ">"), tok(Operator, "=>"), tok(Tag, "ul"),
		tok(Variable, "items"), tok(Function, "map"), tok(Tag, "li"), tok(Attribute, "key"), tok(Tag, "li"),
		tok(Tag, "ul"), tok(Punctuation, ";"),
	}},
	{name: "generic arrow functions", src: "const f = <T,>(x: T) => x;\nconst g = <T extends object>(x: T) => x;\nconst h = 1;", want: []Token{
		tok(Operator, "<"), tok(Type, "T"), tok(Punctuation, ","), tok(Operator, ">"), tok(Operator, "<"), tok(Type, "T"),
		tok(Keyword, "extends"), tok(BuiltinType, "object"), tok(Keyword, "const"), tok(Variable, "h"),
	}},
	{name: "typed props and handlers", src: "function Button({ label }: { label: string }) {\n  return (\n    <button type=\"button\" onClick={(e: MouseEvent) => go(e)}>\n      {label}\n    </button>\n  );\n}", want: []Token{
		tok(Keyword, "function"), tok(Function, "Button"), tok(BuiltinType, "string"), tok(Keyword, "return"),
		tok(Tag, "button"), tok(Attribute, "type"), tok(String, `"button"`), tok(Attribute, "onClick"),
		tok(Type, "MouseEvent"), tok(Function, "go"), tok(Variable, "label"), tok(Tag, "button"),
	}},
	{name: "fragments and conditionals", src: "const v = ok ? <><A /></> : null;", want: []Token{
		tok(Operator, "?"), tok(Punctuation, "<>"), tok(Tag, "A"), tok(Punctuation, "/>"), tok(Punctuation, "</>"),
		tok(Punctuation, ":"), tok(Primitive, "null"),
	}},
}
