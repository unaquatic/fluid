package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sequence is a golden sequence document of two 2×1 steps.
const sequence = `=== step: off ===
=== text ===
|ab|
=== fg ===
|AA|
=== legend ===
A #ffffff
=== step: on ===
=== text ===
|ab|
=== fg ===
|AA|
=== attrs ===
|BU|
=== legend ===
A #ffffff
`

// writeFiles writes each file of files, by slash-separated path, under dir.
func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for p, doc := range files {
		full := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRunHTML(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"a/testdata/sample.frames.golden":      sample,
		"a/testdata/toggle.seq.golden":         sequence,
		"b/testdata/other.frames.golden":       sample,
		".hidden/testdata/skip.frames.golden":  sample,
		"a/testdata/notes.txt":                 "not a golden file",
		"a/testdata/broken.frames.golden.orig": "neither",
	})
	out := filepath.Join(t.ArtifactDir(), "goldens.html")
	if err := RunHTML([]string{"-o", out, "-bg", "#101010", filepath.Join(dir, "a"), filepath.Join(dir, "b"), filepath.Join(dir, ".hidden")}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	page := string(data)
	for _, want := range []string{
		"filter 4 files",
		"background: #101010;",
		filepath.ToSlash(filepath.Join(dir, "a", "testdata")) + "</h4>",
		filepath.ToSlash(filepath.Join(dir, "b", "testdata")) + "</h4>",
		`sample.frames.golden <small>4×1</small>`,
		`toggle.seq.golden <small>sequence · 2 steps</small>`,
		`step: on <small>2×1</small>`,
		`<span style="color:#ffffff;background:#242424">hi</span>`,
		`<span style="background:#242424">  </span>`,
		`font-weight:700">a</span>`,
		`text-decoration:underline">b</span>`,
		"bg #242424 SurfaceRaised",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	for _, unwanted := range []string{"notes.txt", "broken.frames.golden.orig"} {
		if strings.Contains(page, unwanted) {
			t.Errorf("page has %q", unwanted)
		}
	}
	// A dot directory passed as an argument is walked; only dot directories
	// found under it are skipped.
	if !strings.Contains(page, "skip.frames.golden") {
		t.Errorf("page lacks the file under the .hidden argument")
	}
}

func TestRunHTMLErrors(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"testdata/bad.frames.golden": "=== nope ===\n"})
	out := filepath.Join(t.TempDir(), "goldens.html")
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"bad bg", []string{"-o", out, "-bg", "red", t.TempDir()}, `-bg "red" is not #rrggbb`},
		{"empty bg", []string{"-o", out, "-bg", "", t.TempDir()}, `-bg "" is not #rrggbb`},
		{"bad golden", []string{"-o", out, dir}, "testdata/bad.frames.golden: "},
		{"missing dir", []string{"-o", out, filepath.Join(dir, "missing")}, "missing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RunHTML(tt.args)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("RunHTML(%q) = %v, want an error containing %q", tt.args, err, tt.want)
			}
		})
	}
	if _, err := os.Stat(out); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a failed run wrote %s", out)
	}
}
