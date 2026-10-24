package syntax

// builtinLanguages returns the languages Lookup finds.
func builtinLanguages() []*Language {
	js := jsLexer(false, true)
	ts := jsLexer(true, false)
	css := &CSS{}
	scss := &CSS{SCSS: true}
	sh := newShell()
	html := func(vue, svelte bool) *Markup {
		return &Markup{Script: js, TypeScript: ts, Style: css, SCSS: scss, Vue: vue, Svelte: svelte}
	}
	return []*Language{
		goLang(),
		pythonLang(),
		{Name: "javascript", Aliases: []string{"js", "jsx", "mjs", "cjs", "node", "react"}, Lexer: js},
		{Name: "typescript", Aliases: []string{"ts", "mts", "cts"}, Lexer: ts},
		{Name: "tsx", Lexer: jsLexer(true, true)},
		rustLang(),
		cLang(),
		cppLang(),
		csharpLang(),
		javaLang(),
		kotlinLang(),
		swiftLang(),
		scalaLang(),
		dartLang(),
		zigLang(),
		rubyLang(),
		phpLang(),
		luaLang(),
		sqlLang(),
		shellLang(),
		powershellLang(),
		dockerfileLang(sh),
		makefileLang(sh),
		jsonLang(),
		yamlLang(),
		tomlLang(),
		iniLang(),
		diffLang(),
		markdownLang(),
		{Name: "html", Aliases: []string{"htm", "xhtml", "html5"}, Lexer: html(false, false)},
		{Name: "xml", Aliases: []string{"svg", "xsd", "xsl", "xslt", "plist", "csproj", "fsproj", "vbproj", "props", "targets", "rss", "atom", "wsdl", "xaml", "storyboard"}, Lexer: &Markup{}},
		{Name: "vue", Lexer: html(true, false)},
		{Name: "svelte", Lexer: html(false, true)},
		{Name: "css", Lexer: css},
		{Name: "scss", Aliases: []string{"sass", "less"}, Lexer: scss},
		protobufLang(),
		graphqlLang(),
		hclLang(),
	}
}
