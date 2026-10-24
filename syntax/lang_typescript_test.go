package syntax

import "testing"

func TestTypeScript(t *testing.T) {
	testLanguage(t, "typescript", []string{"ts", "mts", "cts", "app.ts"}, typescriptTests)
}

var typescriptTests = []kindCase{
	{name: "type annotations", src: "let a: string = 'x';\nlet b: number | undefined;\nconst c: boolean[] = [];\nfunction f(x: unknown): never { throw x }\nlet d: any, e: void, g: bigint, h: symbol, o: object;", want: []Token{
		tok(Type, "string"), tok(Type, "number"), tok(Operator, "|"), tok(Primitive, "undefined"), tok(Type, "boolean"),
		tok(Keyword, "function"), tok(Function, "f"), tok(Type, "unknown"), tok(Type, "never"), tok(Keyword, "throw"),
		tok(Type, "any"), tok(Type, "void"), tok(Type, "bigint"), tok(Type, "symbol"), tok(Type, "object"),
	}},
	{name: "type declarations", src: "interface User { id: number; name?: string }\ntype ID = string | number;\nenum Color { Red, Green }\nnamespace NS {}\nabstract class Shape implements HasArea {}", want: []Token{
		tok(Keyword, "interface"), tok(Type, "User"), tok(Variable, "id"), tok(Type, "number"), tok(Variable, "name"),
		tok(Operator, "?"), tok(Keyword, "type"), tok(Type, "ID"), tok(Keyword, "enum"), tok(Type, "Color"),
		tok(Type, "Red"), tok(Keyword, "namespace"), tok(Type, "NS"), tok(Keyword, "abstract"), tok(Keyword, "class"),
		tok(Type, "Shape"), tok(Keyword, "implements"), tok(Type, "HasArea"),
	}},
	{name: "generics and assertions", src: "const f = <T>(x: T): T => x;\nlet y = z as const;\nlet w = <string>v;\nfn<Array<T>>();", want: []Token{
		tok(Operator, "<"), tok(Type, "T"), tok(Operator, ">"), tok(Operator, "=>"), tok(Keyword, "as"),
		tok(Keyword, "const"), tok(Operator, "<"), tok(Type, "string"), tok(Operator, ">"), tok(Variable, "v"),
		tok(Variable, "fn"), tok(Builtin, "Array"),
	}},
	{name: "class members", src: "class A {\n  private readonly x = 1;\n  public static y?: string;\n  protected abstract z(): void;\n  override m() {}\n  declare n: number;\n  accessor k = 0;\n}", want: []Token{
		tok(Keyword, "private"), tok(Keyword, "readonly"), tok(Keyword, "public"), tok(Keyword, "static"),
		tok(Keyword, "protected"), tok(Keyword, "abstract"), tok(Function, "z"), tok(Keyword, "override"),
		tok(Function, "m"), tok(Keyword, "declare"), tok(Keyword, "accessor"),
	}},
	{name: "type operators", src: "type K = keyof T;\ntype R = T extends (infer U)[] ? U : never;\nconst c = {} satisfies C;\nfunction isS(x: unknown): x is string {}\nfunction assert(v): asserts v {}\ntype U1 = unique symbol;", want: []Token{
		tok(Keyword, "keyof"), tok(Keyword, "extends"), tok(Keyword, "infer"), tok(Type, "never"),
		tok(Keyword, "satisfies"), tok(Function, "isS"), tok(Keyword, "is"), tok(Type, "string"), tok(Keyword, "asserts"),
		tok(Keyword, "unique"), tok(Type, "symbol"),
	}},
	{name: "member names are not keywords", src: "event.type; x.interface; y.declare()", want: []Token{
		tok(Property, "type"), tok(Property, "interface"), tok(Function, "declare"),
	}},
	{name: "decorators", src: "@Injectable()\nexport class S {\n  constructor(@Inject(T) private t: T) {}\n}", want: []Token{
		tok(Decorator, "@Injectable"), tok(Keyword, "export"), tok(Keyword, "class"), tok(Type, "S"),
		tok(Function, "constructor"), tok(Decorator, "@Inject"), tok(Keyword, "private"),
	}},
	{name: "no jsx", src: "const x = a < b && c > d;\nconst y = <T,>(v: T) => v;", want: []Token{
		tok(Operator, "<"), tok(Operator, ">"), tok(Operator, "<"), tok(Type, "T"),
	}},
	{name: "templates and regular expressions", src: "const s = `id: ${user.id as number}`;\nconst r = /x/g;", want: []Token{
		tok(String, "`id: "), tok(StringEscape, "${"), tok(Property, "id"), tok(Keyword, "as"), tok(Type, "number"),
		tok(StringEscape, "}"), tok(Regex, "/x/g"),
	}},
}
