package syntax

import (
	"slices"
	"sync"
	"testing"
	"testing/synctest"
)

// restoreRegistry makes the languages Register adds during t disappear when
// t ends, so that other tests see the built-in languages alone.
func restoreRegistry(t *testing.T) {
	t.Helper()
	prev := registered.Load()
	t.Cleanup(func() { registered.Store(prev) })
}

// TestRegisterAddsLanguage checks that Lookup finds a registered language by
// its name, aliases, file names, and extension in any case, that Languages
// lists it once after the built-in languages, and that it lexes with its
// Lexer.
func TestRegisterAddsLanguage(t *testing.T) {
	restoreRegistry(t)
	before := slices.Collect(Languages())
	my := &Language{
		Name:      "MyLang",
		Aliases:   []string{"ml"},
		Filenames: []string{"Mlfile"},
		Lexer:     &CLike{Words: map[string]Kind{"frob": Keyword}},
	}
	Register(my)

	for _, name := range []string{"mylang", "MYLANG", "ml", "src/x.ml", "Mlfile", "dir/mlfile"} {
		if got := Lookup(name); got != my {
			t.Errorf("Lookup(%q) = %v, want the registered language", name, got)
		}
	}
	after := slices.Collect(Languages())
	if len(after) != len(before)+1 || after[len(after)-1] != my {
		t.Errorf("Languages has %d languages ending in %v, want %d ending in the registered one", len(after), after[len(after)-1], len(before)+1)
	}
	var kinds []Kind
	for tk := range my.Tokens("frob x") {
		kinds = append(kinds, tk.Kind)
	}
	if len(kinds) == 0 || kinds[0] != Keyword {
		t.Errorf("tokens of %q have kinds %v, want a Keyword first", "frob x", kinds)
	}
}

// TestRegisterReplacesSameName checks that a language registered under a
// built-in language's Name replaces it, in Lookup and in Languages.
func TestRegisterReplacesSameName(t *testing.T) {
	restoreRegistry(t)
	old := Lookup("go")
	before := slices.Collect(Languages())
	mine := &Language{Name: "Go", Lexer: old.Lexer}
	Register(mine)

	for _, name := range []string{"go", "main.go"} {
		if got := Lookup(name); got != mine {
			t.Errorf("Lookup(%q) did not find the replacement", name)
		}
	}
	after := slices.Collect(Languages())
	if len(after) != len(before) {
		t.Errorf("Languages has %d languages, want %d", len(after), len(before))
	}
	if slices.Contains(after, old) {
		t.Error("Languages still lists the replaced language")
	}
}

// TestRegisterTakesOverSharedNames checks that an alias a registered
// language shares with an earlier one finds the registered language, while
// the earlier one stays found by its other names.
func TestRegisterTakesOverSharedNames(t *testing.T) {
	restoreRegistry(t)
	markdown := Lookup("markdown")
	md2 := &Language{Name: "md2", Aliases: []string{"md"}, Lexer: &CLike{}}
	Register(md2)

	if got := Lookup("md"); got != md2 {
		t.Errorf("Lookup(%q) did not find the registered language", "md")
	}
	if got := Lookup("markdown"); got != markdown {
		t.Errorf("Lookup(%q) = %v, want the built-in Markdown", "markdown", got)
	}
}

// TestRegisterPanics checks that Register refuses a nil Language, one with
// no Lexer, and one with no Name.
func TestRegisterPanics(t *testing.T) {
	restoreRegistry(t)
	for name, l := range map[string]*Language{
		"nil":      nil,
		"no lexer": {Name: "x"},
		"no name":  {Lexer: &CLike{}},
		"blank":    {Name: "  ", Lexer: &CLike{}},
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("Register did not panic")
				}
			}()
			Register(l)
		})
	}
}

// TestRegisterConcurrent checks that Register is safe beside Lookup and
// Languages, and beside itself, as the race detector sees it.
func TestRegisterConcurrent(t *testing.T) {
	restoreRegistry(t)
	synctest.Test(t, func(t *testing.T) {
		var wg sync.WaitGroup
		for i := range 8 {
			wg.Go(func() {
				for range 100 {
					if i%2 == 0 {
						Register(&Language{Name: "race", Lexer: &CLike{}})
					} else {
						Lookup("race")
						for range Languages() {
						}
					}
				}
			})
		}
		wg.Wait()
	})
	if Lookup("race") == nil {
		t.Error("Lookup did not find the language registered concurrently")
	}
}
