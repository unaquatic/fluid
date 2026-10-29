package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/unaquatic/fluid/style"
)

// TestLoadTheme checks that loadTheme gives style.DarculaSolid for no file,
// reads a theme written as JSON, and fails on a file that holds none.
func TestLoadTheme(t *testing.T) {
	th, err := loadTheme("")
	if err != nil || *th != *style.DarculaSolid() {
		t.Errorf("loadTheme(\"\") = %v, %v; want style.DarculaSolid", th, err)
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
	if th, err := loadTheme(p); err != nil || *th != *want {
		t.Errorf("loadTheme(%s) = %v, %v; want the theme written there", p, th, err)
	}

	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadTheme(bad); err == nil {
		t.Errorf("loadTheme(%s) = nil error, want one", bad)
	}
}
