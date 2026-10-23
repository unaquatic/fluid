package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/unaquatic/fluid/style"
)

// TestLoadTheme checks that LoadTheme gives style.DarculaSolid for no file,
// reads a theme written as JSON, and fails on a file that holds none.
func TestLoadTheme(t *testing.T) {
	th, err := LoadTheme("")
	if err != nil || *th != *style.DarculaSolid() {
		t.Errorf("LoadTheme(\"\") = %v, %v; want style.DarculaSolid", th, err)
	}

	dir := t.TempDir()
	want := style.DefaultDark()
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "theme.json")
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if th, err := LoadTheme(p); err != nil || *th != *want {
		t.Errorf("LoadTheme(%s) = %v, %v; want the theme written there", p, th, err)
	}

	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadTheme(bad); err == nil {
		t.Errorf("LoadTheme(%s) = nil error, want one", bad)
	}
}
