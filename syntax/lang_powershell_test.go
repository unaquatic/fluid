package syntax

import "testing"

func TestPowerShell(t *testing.T) {
	testLanguage(t, "powershell", []string{"pwsh", "ps1", "psm1", "posh", "build.ps1"}, powershellTests)
}

var powershellTests = []kindCase{
	{name: "functions and parameters", src: "function Get-Info {\n  param([string]$Path, [int]$Depth = 1)\n  Get-ChildItem -Path $Path -Recurse | Where-Object { $_.Length -gt 1kb }\n}", want: []Token{
		tok(Keyword, "function"), tok(Function, "Get-Info"), tok(Keyword, "param"), tok(Type, "string"),
		tok(Variable, "$Path"), tok(Type, "int"), tok(Variable, "$Depth"), tok(Function, "Get-ChildItem"),
		tok(Operator, "-"), tok(Variable, "Path"), tok(Variable, "$Path"), tok(Operator, "-"), tok(Variable, "Recurse"),
		tok(Operator, "|"), tok(Function, "Where-Object"), tok(Variable, "$_"), tok(Property, "Length"),
		tok(Operator, "-gt"), tok(Number, "1kb"),
	}},
	{name: "control flow", src: "if ($a -eq 1) { } elseif ($b) { } else { }\nforeach ($i in $list) { break }\nwhile ($true) { continue }\nswitch ($x) { 'a' { } }\ntry { throw 'x' } catch { } finally { }\nreturn", want: []Token{
		tok(Keyword, "if"), tok(Operator, "-eq"), tok(Keyword, "elseif"), tok(Keyword, "else"), tok(Keyword, "foreach"),
		tok(Keyword, "in"), tok(Keyword, "break"), tok(Keyword, "while"), tok(Primitive, "$true"), tok(Keyword, "continue"),
		tok(Keyword, "switch"), tok(String, "'a'"), tok(Keyword, "try"), tok(Keyword, "throw"), tok(Keyword, "catch"),
		tok(Keyword, "finally"), tok(Keyword, "return"),
	}},
	{name: "strings", src: "$s = \"Hi $name `n\"\n$t = 'it''s'\n$u = \"say \"\"hi\"\"\"", want: []Token{
		tok(String, `"Hi `), tok(StringEscape, "$name"), tok(String, " `n\""), tok(String, "'it"), tok(StringEscape, "''"),
		tok(String, "s'"), tok(String, `"say `), tok(StringEscape, `""`), tok(String, "hi"), tok(StringEscape, `""`),
		tok(String, `"`),
	}},
	{name: "here-strings", src: "$h = @\"\nline $x\n\"@\n$l = @'\nraw\n'@\n$after = 1", want: []Token{
		tok(String, `@"`), tok(String, "line $x"), tok(String, `"@`), tok(String, "@'"), tok(String, "raw"),
		tok(String, "'@"), tok(Variable, "$after"),
	}},
	{name: "operators", src: "$a = 1 + 2 * 3 - 4\n$b -and $c -or -not $d\n$e -replace 'x', 'y'\n$f -like '*.ps1'\n$i++; $j--; $k -= 1", want: []Token{
		tok(Operator, "+"), tok(Operator, "*"), tok(Operator, "-"), tok(Number, "4"), tok(Operator, "-and"),
		tok(Operator, "-or"), tok(Operator, "-not"), tok(Operator, "-replace"), tok(Operator, "-like"),
		tok(Operator, "++"), tok(Operator, "--"), tok(Operator, "-="),
	}},
	{name: "comments", src: "# line\n<# block\ncomment #>\n$x = 1", want: []Token{
		tok(Comment, "# line"), tok(Comment, "<# block"), tok(Comment, "comment #>"), tok(Variable, "$x"),
	}},
	{name: "types and primitives", src: "[bool]$b = $false\n$n = $null\n[datetime]::Now\n[hashtable]$h = @{}", want: []Token{
		tok(Type, "bool"), tok(Primitive, "$false"), tok(Primitive, "$null"), tok(Type, "datetime"),
		tok(Punctuation, "::"), tok(Type, "hashtable"),
	}},
	{name: "keywords ignore case", src: "IF ($x) { RETURN }\nFunction F { }", want: []Token{
		tok(Keyword, "IF"), tok(Keyword, "RETURN"), tok(Keyword, "Function"), tok(Function, "F"),
	}},
	{name: "pipelines and members", src: "Get-Process | Sort-Object CPU -Descending | Select-Object -First 5\n$p.Kill()\n$env:PATH", want: []Token{
		tok(Function, "Get-Process"), tok(Function, "Sort-Object"), tok(Variable, "CPU"), tok(Operator, "-"),
		tok(Variable, "Descending"), tok(Function, "Select-Object"), tok(Number, "5"), tok(Variable, "$p"),
		tok(Function, "Kill"), tok(Variable, "$env"),
	}},
	{name: "unclosed here-string", src: "$h = @\"\nopen", open: true, want: []Token{
		tok(String, `@"`), tok(String, "open"),
	}},
}
