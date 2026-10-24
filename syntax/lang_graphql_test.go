package syntax

import "testing"

func TestGraphQL(t *testing.T) {
	testLanguage(t, "graphql", []string{"gql", "graphqls", "schema.graphql"}, graphqlTests)
}

var graphqlTests = []kindCase{
	{name: "queries", src: "query GetUser($id: ID!, $first: Int = 10) {\n  user(id: $id) {\n    name\n    friends(first: $first) { edges { node { id } } }\n  }\n}", want: []Token{
		tok(Keyword, "query"), tok(Function, "GetUser"), tok(Variable, "$id"), tok(Punctuation, ":"), tok(Type, "ID"),
		tok(Operator, "!"), tok(Variable, "$first"), tok(Type, "Int"), tok(Number, "10"), tok(Function, "user"),
		tok(Variable, "id"), tok(Variable, "$id"), tok(Variable, "name"), tok(Function, "friends"), tok(Variable, "$first"),
	}},
	{name: "mutations and fragments", src: "mutation { createUser(input: {name: \"x\"}) { ...UserFields } }\nfragment UserFields on User { id name }", want: []Token{
		tok(Keyword, "mutation"), tok(Function, "createUser"), tok(String, `"x"`), tok(Operator, "..."),
		tok(Type, "UserFields"), tok(Keyword, "fragment"), tok(Type, "UserFields"), tok(Keyword, "on"), tok(Type, "User"),
	}},
	{name: "subscriptions", src: "subscription OnEvent { event { kind } }", want: []Token{
		tok(Keyword, "subscription"), tok(Function, "OnEvent"),
	}},
	{name: "schema definitions", src: "type Query {\n  user(id: ID!): User\n  users: [User!]!\n}\ninterface Node { id: ID! }\nunion Result = User | Error\nenum Role { ADMIN USER }\ninput NewUser { name: String! }\nscalar Date\nschema { query: Query }\ntype Admin implements Node & Entity { id: ID! }", want: []Token{
		tok(Keyword, "type"), tok(Type, "Query"), tok(Function, "user"), tok(Type, "ID"), tok(Type, "User"),
		tok(Variable, "users"), tok(Punctuation, "["), tok(Type, "User"), tok(Keyword, "interface"), tok(Type, "Node"),
		tok(Keyword, "union"), tok(Type, "Result"), tok(Operator, "|"), tok(Keyword, "enum"), tok(Type, "Role"),
		tok(Keyword, "input"), tok(Type, "NewUser"), tok(Type, "String"), tok(Keyword, "scalar"), tok(Type, "Date"),
		tok(Keyword, "schema"), tok(Keyword, "query"), tok(Type, "Query"), tok(Keyword, "type"), tok(Type, "Admin"),
		tok(Keyword, "implements"), tok(Type, "Node"), tok(Operator, "&"),
	}},
	{name: "directives", src: "extend type User @key(fields: \"id\")\ndirective @auth(role: Role) repeatable on FIELD_DEFINITION\nquery { a @include(if: $show) }", want: []Token{
		tok(Keyword, "extend"), tok(Keyword, "type"), tok(Type, "User"), tok(Decorator, "@key"), tok(String, `"id"`),
		tok(Keyword, "directive"), tok(Decorator, "@auth"), tok(Keyword, "repeatable"), tok(Keyword, "on"),
		tok(Decorator, "@include"), tok(Variable, "$show"),
	}},
	{name: "descriptions and comments", src: "\"\"\"\nMulti-line description\n\"\"\"\ntype T {\n  \"field doc\"\n  f: Int # comment\n}", want: []Token{
		tok(String, `"""`), tok(String, "Multi-line description"), tok(String, `"""`), tok(Keyword, "type"),
		tok(String, `"field doc"`), tok(Type, "Int"), tok(Comment, "# comment"),
	}},
	{name: "literals", src: "query { a(flag: true, n: null, f: 1.5, i: -2, s: \"esc\\\"q\") }", want: []Token{
		tok(Primitive, "true"), tok(Primitive, "null"), tok(Number, "1.5"), tok(Operator, "-"), tok(Number, "2"),
		tok(String, `"esc`), tok(StringEscape, `\"`), tok(String, `q"`),
	}},
}
