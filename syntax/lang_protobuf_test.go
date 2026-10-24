package syntax

import "testing"

func TestProtobuf(t *testing.T) {
	testLanguage(t, "protobuf", []string{"proto", "proto3", "api.proto"}, protobufTests)
}

var protobufTests = []kindCase{
	{name: "file options", src: "syntax = \"proto3\";\npackage acme.v1;\nimport \"google/protobuf/timestamp.proto\";\nimport public \"other.proto\";\noption go_package = \"x/y\";", want: []Token{
		tok(Keyword, "syntax"), tok(Operator, "="), tok(String, `"proto3"`), tok(Keyword, "package"),
		tok(Variable, "acme"), tok(Property, "v1"), tok(Keyword, "import"), tok(String, `"google/protobuf/timestamp.proto"`),
		tok(Keyword, "import"), tok(Keyword, "public"), tok(Keyword, "option"), tok(Variable, "go_package"),
	}},
	{name: "messages", src: "message User {\n  string name = 1;\n  repeated int32 ids = 2 [packed = true];\n  optional bytes data = 3;\n  map<string, Address> addrs = 4;\n  oneof contact { string email = 5; }\n  reserved 6, 7 to 9;\n  google.protobuf.Timestamp at = 10;\n}", want: []Token{
		tok(Keyword, "message"), tok(Type, "User"), tok(BuiltinType, "string"), tok(Variable, "name"), tok(Number, "1"),
		tok(Keyword, "repeated"), tok(BuiltinType, "int32"), tok(Primitive, "true"), tok(Keyword, "optional"),
		tok(BuiltinType, "bytes"), tok(Keyword, "map"), tok(BuiltinType, "string"), tok(Type, "Address"), tok(Keyword, "oneof"),
		tok(Variable, "contact"), tok(Keyword, "reserved"), tok(Number, "6"), tok(Keyword, "to"), tok(Variable, "google"),
		tok(Property, "protobuf"), tok(Type, "Timestamp"),
	}},
	{name: "enums and services", src: "enum Status { STATUS_UNKNOWN = 0; }\nservice Users {\n  rpc Get(GetReq) returns (User);\n  rpc Watch(stream Req) returns (stream Event);\n}", want: []Token{
		tok(Keyword, "enum"), tok(Type, "Status"), tok(Number, "0"), tok(Keyword, "service"), tok(Type, "Users"),
		tok(Keyword, "rpc"), tok(Function, "Get"), tok(Type, "GetReq"), tok(Keyword, "returns"), tok(Type, "User"),
		tok(Keyword, "rpc"), tok(Function, "Watch"), tok(Keyword, "stream"), tok(Type, "Req"), tok(Keyword, "returns"),
		tok(Keyword, "stream"), tok(Type, "Event"),
	}},
	{name: "field options and literals", src: "int32 a = 1 [default = -5];\nstring s = 2 [default = 'x'];\ndouble d = 3 [deprecated = false];", want: []Token{
		tok(BuiltinType, "int32"), tok(Number, "1"), tok(Operator, "-"), tok(Number, "5"), tok(BuiltinType, "string"),
		tok(String, "'x'"), tok(BuiltinType, "double"), tok(Primitive, "false"),
	}},
	{name: "extensions and editions", src: "edition = \"2023\";\nextend Foo { int32 bar = 126; }\nmessage M { extensions 100 to max; }", want: []Token{
		tok(Keyword, "edition"), tok(String, `"2023"`), tok(Keyword, "extend"), tok(Type, "Foo"), tok(Keyword, "message"),
		tok(Keyword, "extensions"), tok(Keyword, "to"), tok(Keyword, "max"),
	}},
	{name: "comments", src: "// line\n/* block\n */ message M {}", want: []Token{
		tok(Comment, "// line"), tok(Comment, "/* block"), tok(Comment, " */"), tok(Keyword, "message"),
	}},
}
