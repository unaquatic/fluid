package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/unaquatic/fluid/style"
)

// sample is a golden frames document of a 4×1 design.
const sample = `=== text ===
|hi  |
=== fg ===
|AA  |
=== bg ===
|░░░░|
=== legend ===
A #ffffff
░ #242424 SurfaceRaised
`

// newServer serves a library rooted in a new directory that holds
// testdata/sample.frames.golden, and returns the server and the directory.
func newServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "testdata"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "testdata", "sample.frames.golden"), []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	srv := httptest.NewServer((&server{library: library{root: root}, theme: style.DarculaSolid()}).handler())
	t.Cleanup(srv.Close)
	return srv, dir
}

// call sends a request and returns the response's status and body.
func call(t *testing.T, method, url, contentType, body string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, string(data)
}

// sampleJSON returns the design of sample as JSON.
func sampleJSON(t *testing.T) string {
	t.Helper()
	snaps, err := parseDoc(sample)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(newDesign(snaps[0].Buf, snaps[0].Names))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestHandler(t *testing.T) {
	srv, _ := newServer(t)
	designJSON := sampleJSON(t)
	tests := []struct {
		name, method, path, body string
		status                   int
		want                     string // a part of the body
	}{
		{"page", "GET", "/", "", 200, "<title>fluid design</title>"},
		{"theme", "GET", "/api/theme", "", 200, `"name":"TextBase"`},
		{"library", "GET", "/api/library", "", 200, `{"path":"testdata/sample.frames.golden","size":"4×1"}`},
		{"library colors", "GET", "/api/library", "", 200, `{"hex":"#242424","name":"SurfaceRaised","fg":0,"bg":1}`},
		{"open", "GET", "/api/open?path=testdata/sample.frames.golden", "", 200, `"names":{"#242424":"SurfaceRaised"}`},
		{"open step", "GET", "/api/open?path=testdata/sample.frames.golden&step=1", "", 400, `step "1" is not between 0 and 0`},
		{"open missing", "GET", "/api/open?path=testdata/none.frames.golden", "", 404, "no such file"},
		{"open outside", "GET", "/api/open?path=../sample.frames.golden", "", 400, "not a golden file"},
		{"open other", "GET", "/api/open?path=go.mod", "", 400, "not a golden file"},
		{"parse", "POST", "/api/parse", sample, 200, `{"ch":"h","fg":"#ffffff","bg":"#242424"}`},
		{"parse junk", "POST", "/api/parse", "hello", 422, "comes before the first section"},
		{"frames", "POST", "/api/frames", designJSON, 200, sample},
		{"frames bad", "POST", "/api/frames", `{"width":1,"height":1,"cells":[[{"ch":"ab"}]]}`, 422, "not one code point"},
		{"frames junk", "POST", "/api/frames", `{`, 400, "design: unexpected EOF"},
		{"ansi", "POST", "/api/ansi", designJSON, 200, "\x1b[0;38;2;255;255;255;48;2;36;36;36mhi"},
		{"missing", "GET", "/missing", "", 404, "not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ct := ""
			if tt.method == "POST" {
				ct = "application/json"
			}
			status, body := call(t, tt.method, srv.URL+tt.path, ct, tt.body)
			if status != tt.status || !strings.Contains(body, tt.want) {
				t.Errorf("%s %s = %d\n%s\nwant %d holding %q", tt.method, tt.path, status, body, tt.status, tt.want)
			}
		})
	}
}

func TestSave(t *testing.T) {
	srv, dir := newServer(t)
	designJSON := sampleJSON(t)
	save := func(path string) (int, string) {
		return call(t, "POST", srv.URL+"/api/save?"+path, "application/json", designJSON)
	}
	if status, body := save("path=designs/new.frames.golden"); status != 200 || !strings.Contains(body, `"path":"designs/new.frames.golden"`) {
		t.Fatalf("save = %d %s, want 200", status, body)
	}
	got, err := os.ReadFile(filepath.Join(dir, "designs", "new.frames.golden"))
	if err != nil || string(got) != sample {
		t.Fatalf("saved file = %q, %v; want the sample", got, err)
	}
	if status, _ := save("path=designs/new.frames.golden"); status != http.StatusConflict {
		t.Errorf("saving over a file = %d, want 409", status)
	}
	if status, _ := save("path=designs/new.frames.golden&overwrite=1"); status != 200 {
		t.Errorf("saving over a file with overwrite=1 = %d, want 200", status)
	}
	for _, p := range []string{
		"path=../escape.frames.golden",
		"path=" + filepath.Join(dir, "abs.frames.golden"),
		"path=designs/new.txt",
		"path=.frames.golden",
		"path=",
	} {
		if status, body := save(p); status != http.StatusBadRequest {
			t.Errorf("save %s = %d %s, want 400", p, status, body)
		}
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "escape.frames.golden")); err == nil {
		t.Error("save wrote a file outside the root")
	}
}

// TestSaveFollowsNoLinkOut checks that a symbolic link inside the root
// cannot lead a save outside it.
func TestSaveFollowsNoLinkOut(t *testing.T) {
	srv, dir := newServer(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, "out")); err != nil {
		t.Skip(err)
	}
	status, _ := call(t, "POST", srv.URL+"/api/save?path=out/x.frames.golden", "application/json", sampleJSON(t))
	if status == 200 {
		t.Error("save through a link out of the root succeeded")
	}
	if _, err := os.Stat(filepath.Join(outside, "x.frames.golden")); err == nil {
		t.Error("save wrote a file outside the root")
	}
}

func TestHandlerRejectsOtherSites(t *testing.T) {
	srv, _ := newServer(t)
	h := srv.Config.Handler
	req := httptest.NewRequest("GET", "/api/theme", nil)
	req.Host = "attacker.example:4098"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("request to host %s = %d, want 403", req.Host, rec.Code)
	}

	req = httptest.NewRequest("POST", "/api/save?path=x.frames.golden", strings.NewReader(sampleJSON(t)))
	req.Host = "localhost:4098"
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Origin", "https://attacker.example")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("cross-site save = %d, want 403", rec.Code)
	}
}

// TestPageCallsEveryEndpoint checks that the page uses each API endpoint
// that the handler serves.
func TestPageCallsEveryEndpoint(t *testing.T) {
	for _, path := range regexp.MustCompile(`/api/\w+`).FindAllString(handlerDoc(t), -1) {
		if !regexp.MustCompile(`['"` + "`" + `]` + path + `\b`).Match(page) {
			t.Errorf("the page does not call %s", path)
		}
	}
}

// handlerDoc returns the source of server.handler's doc comment.
func handlerDoc(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile("design_server.go")
	if err != nil {
		t.Fatal(err)
	}
	doc, _, ok := strings.Cut(string(src), "func (s *server) handler()")
	if !ok {
		t.Fatal("design_server.go has no server.handler")
	}
	return doc[strings.LastIndex(doc, "// handler returns"):]
}
