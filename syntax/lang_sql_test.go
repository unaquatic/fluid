package syntax

import "testing"

func TestSQL(t *testing.T) {
	testLanguage(t, "sql", []string{"mysql", "postgresql", "postgres", "psql", "sqlite", "plpgsql", "schema.sql"}, sqlTests)
}

var sqlTests = []kindCase{
	{name: "select", src: "SELECT u.id, COUNT(*) AS n\nFROM users u\nLEFT JOIN orders o ON o.user_id = u.id\nWHERE u.active = TRUE AND u.name LIKE 'A%'\nGROUP BY u.id HAVING COUNT(*) > 1\nORDER BY n DESC LIMIT 10 OFFSET 5;", want: []Token{
		tok(Keyword, "SELECT"), tok(Variable, "u"), tok(Property, "id"), tok(Builtin, "COUNT"), tok(Operator, "*"),
		tok(Keyword, "AS"), tok(Variable, "n"), tok(Keyword, "FROM"), tok(Variable, "users"), tok(Keyword, "LEFT"),
		tok(Keyword, "JOIN"), tok(Keyword, "ON"), tok(Property, "user_id"), tok(Keyword, "WHERE"), tok(Property, "active"),
		tok(Primitive, "TRUE"), tok(Keyword, "AND"), tok(Keyword, "LIKE"), tok(String, "'A%'"), tok(Keyword, "GROUP"),
		tok(Keyword, "BY"), tok(Keyword, "HAVING"), tok(Keyword, "ORDER"), tok(Keyword, "DESC"), tok(Keyword, "LIMIT"),
		tok(Number, "10"), tok(Keyword, "OFFSET"), tok(Number, "5"),
	}},
	{name: "data definition", src: "CREATE TABLE IF NOT EXISTS t (\n  id BIGSERIAL PRIMARY KEY,\n  name VARCHAR(255) NOT NULL DEFAULT '',\n  data JSONB,\n  created_at TIMESTAMPTZ REFERENCES u(id)\n);\nDROP INDEX idx;\nALTER TABLE t ADD COLUMN c INT;", want: []Token{
		tok(Keyword, "CREATE"), tok(Keyword, "TABLE"), tok(Keyword, "IF"), tok(Keyword, "NOT"), tok(Keyword, "EXISTS"),
		tok(Variable, "id"), tok(Type, "BIGSERIAL"), tok(Keyword, "PRIMARY"), tok(Keyword, "KEY"), tok(Variable, "name"),
		tok(Type, "VARCHAR"), tok(Number, "255"), tok(Keyword, "NOT"), tok(Primitive, "NULL"), tok(Keyword, "DEFAULT"),
		tok(String, "''"), tok(Type, "JSONB"), tok(Type, "TIMESTAMPTZ"), tok(Keyword, "REFERENCES"), tok(Keyword, "DROP"),
		tok(Keyword, "INDEX"), tok(Keyword, "ALTER"), tok(Keyword, "ADD"), tok(Keyword, "COLUMN"), tok(Type, "INT"),
	}},
	{name: "data manipulation", src: "INSERT INTO t (a, b) VALUES (1, 'x') ON CONFLICT DO NOTHING RETURNING id;\nUPDATE t SET a = a + 1 WHERE b IS NULL;\nDELETE FROM t WHERE id IN (SELECT id FROM s);\nWITH RECURSIVE r AS (SELECT 1) SELECT * FROM r UNION ALL SELECT 2;", want: []Token{
		tok(Keyword, "INSERT"), tok(Keyword, "INTO"), tok(Keyword, "VALUES"), tok(Keyword, "ON"), tok(Keyword, "CONFLICT"),
		tok(Keyword, "DO"), tok(Keyword, "NOTHING"), tok(Keyword, "RETURNING"), tok(Keyword, "UPDATE"), tok(Keyword, "SET"),
		tok(Keyword, "IS"), tok(Primitive, "NULL"), tok(Keyword, "DELETE"), tok(Keyword, "IN"), tok(Keyword, "WITH"),
		tok(Keyword, "RECURSIVE"), tok(Keyword, "UNION"), tok(Keyword, "ALL"),
	}},
	{name: "strings and quoted names", src: "SELECT 'O''Brien', \"Quoted Col\", `back`, E'esc\\n', 'multi\nline';", want: []Token{
		tok(String, "'O"), tok(StringEscape, "''"), tok(String, "Brien'"), tok(String, `"Quoted Col"`), tok(String, "`back`"),
		tok(String, `E'esc`), tok(StringEscape, `\n`), tok(String, "'"), tok(String, "'multi"), tok(String, "line'"),
	}},
	{name: "dollar quoting", src: "SELECT $$body$$, $tag$x$tag$;\nCREATE FUNCTION f() RETURNS int AS $$\n  SELECT 1;\n$$ LANGUAGE sql;", want: []Token{
		tok(String, "$$body$$"), tok(String, "$tag$x$tag$"), tok(Keyword, "CREATE"), tok(Keyword, "FUNCTION"),
		tok(Function, "f"), tok(Keyword, "RETURNS"), tok(Type, "int"), tok(Keyword, "AS"), tok(String, "$$"),
		tok(String, "  SELECT 1;"), tok(String, "$$"), tok(Keyword, "LANGUAGE"),
	}},
	{name: "comments", src: "-- line\n/* block\n comment */ SELECT 1;", want: []Token{
		tok(Comment, "-- line"), tok(Comment, "/* block"), tok(Comment, " comment */"), tok(Keyword, "SELECT"),
	}},
	{name: "parameters", src: "SELECT * FROM t WHERE id = $1 AND name = @name AND x = ?;", want: []Token{
		tok(Variable, "$1"), tok(Keyword, "AND"), tok(Variable, "@name"), tok(Operator, "?"),
	}},
	{name: "keywords ignore case", src: "select coalesce(a, 0) from t where b is not null", want: []Token{
		tok(Keyword, "select"), tok(Builtin, "coalesce"), tok(Keyword, "from"), tok(Keyword, "where"), tok(Keyword, "is"),
		tok(Keyword, "not"), tok(Primitive, "null"),
	}},
	{name: "window functions and casts", src: "SELECT ROW_NUMBER() OVER (PARTITION BY a ORDER BY b), CAST(x AS TEXT), my_fn(y) FROM t;", want: []Token{
		tok(Builtin, "ROW_NUMBER"), tok(Keyword, "OVER"), tok(Keyword, "PARTITION"), tok(Keyword, "BY"),
		tok(Keyword, "ORDER"), tok(Keyword, "CAST"), tok(Keyword, "AS"), tok(Type, "TEXT"), tok(Function, "my_fn"),
	}},
	{name: "numbers and operators", src: "SELECT 1, 1.5, 1e3, -2, a <> b, c || d, e >= f;", want: []Token{
		tok(Number, "1"), tok(Number, "1.5"), tok(Number, "1e3"), tok(Operator, "-"), tok(Number, "2"), tok(Operator, "<>"),
		tok(Operator, "||"), tok(Operator, ">="),
	}},
	{name: "unclosed dollar quote", src: "SELECT $$open\nstill", open: true, want: []Token{
		tok(String, "$$open"), tok(String, "still"),
	}},
}
