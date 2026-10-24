package syntax

import "testing"

func TestHCL(t *testing.T) {
	testLanguage(t, "hcl", []string{"terraform", "tf", "tfvars", "main.tf", "prod.tfvars"}, hclTests)
}

var hclTests = []kindCase{
	{name: "blocks and attributes", src: "terraform {\n  required_version = \">= 1.5\"\n}\nprovider \"aws\" {\n  region = var.region\n}\nresource \"aws_instance\" \"web\" {\n  ami           = \"ami-123\"\n  instance_type = \"t3.micro\"\n  count         = 2\n}", want: []Token{
		tok(Keyword, "terraform"), tok(Property, "required_version"), tok(Operator, "="), tok(String, `">= 1.5"`),
		tok(Keyword, "provider"), tok(String, `"aws"`), tok(Property, "region"), tok(Variable, "var"), tok(Property, "region"),
		tok(Keyword, "resource"), tok(String, `"aws_instance"`), tok(String, `"web"`), tok(Property, "ami"),
		tok(Property, "instance_type"), tok(Property, "count"), tok(Number, "2"),
	}},
	{name: "interpolation", src: "name = \"${var.prefix}-web-${count.index}\"\ntags = { Name = \"x\", Env = local.env }", want: []Token{
		tok(Property, "name"), tok(String, `"`), tok(StringEscape, "${"), tok(Variable, "var"), tok(Property, "prefix"),
		tok(StringEscape, "}"), tok(String, "-web-"), tok(StringEscape, "${"), tok(Variable, "count"),
		tok(Property, "index"), tok(StringEscape, "}"), tok(Property, "tags"), tok(Property, "Name"), tok(Property, "Env"),
		tok(Variable, "local"), tok(Property, "env"),
	}},
	{name: "expressions", src: "x = var.enabled ? 1 : 0\ny = [for s in var.list : upper(s) if s != \"\"]\nz = lookup(var.m, \"k\", null)\nw = length(var.l) > 0 && true", want: []Token{
		tok(Operator, "?"), tok(Keyword, "for"), tok(Keyword, "in"), tok(Function, "upper"), tok(Keyword, "if"),
		tok(Operator, "!="), tok(Function, "lookup"), tok(Primitive, "null"), tok(Function, "length"), tok(Operator, "&&"),
		tok(Primitive, "true"),
	}},
	{name: "heredocs", src: "user_data = <<-EOT\n  #!/bin/bash\n  echo ${var.x}\n  EOT\nnext = 1", want: []Token{
		tok(Property, "user_data"), tok(Operator, "<<-"), tok(String, "EOT"), tok(String, "  #!/bin/bash"),
		tok(String, "  echo ${var.x}"), tok(String, "  EOT"), tok(Property, "next"),
	}},
	{name: "variables, outputs, and modules", src: "variable \"region\" {\n  type    = string\n  default = \"us-east-1\"\n}\nvariable \"tags\" { type = map(string) }\noutput \"ip\" { value = aws_instance.web.public_ip }\nlocals { env = \"prod\" }\nmodule \"vpc\" { source = \"./vpc\" }\ndata \"aws_ami\" \"x\" {}\ndynamic \"ingress\" {\n  content {}\n}", want: []Token{
		tok(Keyword, "variable"), tok(Property, "type"), tok(BuiltinType, "string"), tok(Property, "default"), tok(Keyword, "variable"),
		tok(BuiltinType, "map"), tok(BuiltinType, "string"), tok(Keyword, "output"), tok(Property, "value"), tok(Variable, "aws_instance"),
		tok(Property, "web"), tok(Property, "public_ip"), tok(Keyword, "locals"), tok(Keyword, "module"),
		tok(Property, "source"), tok(Keyword, "data"), tok(Keyword, "dynamic"), tok(Keyword, "content"),
	}},
	{name: "comments", src: "# hash\n// slash\n/* block\n */ a = 1", want: []Token{
		tok(Comment, "# hash"), tok(Comment, "// slash"), tok(Comment, "/* block"), tok(Comment, " */"), tok(Property, "a"),
	}},
	{name: "numbers and comparisons", src: "a = 1.5\nb = 1e3\nc = -2\nd = a == b", want: []Token{
		tok(Number, "1.5"), tok(Number, "1e3"), tok(Operator, "-"), tok(Number, "2"), tok(Variable, "a"), tok(Operator, "=="),
	}},
}
