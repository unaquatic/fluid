package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/unaquatic/fluid/style"
	"github.com/unaquatic/fluid/testutil"
)

// page is the editor that the server serves at /.
//
//go:embed design.html
var page []byte

// MaxBody is the size of the largest request body the server reads.
const MaxBody = 16 << 20

// Server serves the editor page and the API it calls.
type Server struct {
	Library Library
	// Theme is the theme whose roles the palette offers.
	Theme *style.Theme
}

// ThemeInfo is the theme as GET /api/theme returns it.
type ThemeInfo struct {
	// Name is the theme's name.
	Name string `json:"name"`
	// Fg colors text by default, and Bg shows behind transparent cells.
	Fg string `json:"fg"`
	Bg string `json:"bg"`
	// Groups holds the theme's colors by role.
	Groups []SwatchGroup `json:"groups"`
}

// LibraryInfo is the library as GET /api/library returns it.
type LibraryInfo struct {
	Files  []FileEntry `json:"files"`
	Colors []UsedColor `json:"colors"`
}

// Handler returns the handler of the editor and its API:
//
//	GET  /                  the editor page; ?open=p opens golden file p in it
//	GET  /api/theme         ThemeInfo
//	GET  /api/library       LibraryInfo
//	GET  /api/open?path=p   the Design of golden file p; step=n picks a sequence step
//	POST /api/parse         the Design of the golden document in the body; step=n as above
//	POST /api/frames        the golden frames document of the Design in the body
//	POST /api/ansi          the Design in the body as text with escape sequences
//	POST /api/save?path=p   writes the Design in the body to p; overwrite=1 replaces it
//
// It serves only requests to a loopback host from the same origin.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(page)
	})
	mux.HandleFunc("GET /api/theme", s.ServeTheme)
	mux.HandleFunc("GET /api/library", s.ServeLibrary)
	mux.HandleFunc("GET /api/open", s.ServeOpen)
	mux.HandleFunc("POST /api/parse", s.ServeParse)
	mux.HandleFunc("POST /api/frames", s.ServeFrames)
	mux.HandleFunc("POST /api/ansi", s.ServeANSI)
	mux.HandleFunc("POST /api/save", s.ServeSave)
	return LocalOnly(http.NewCrossOriginProtection().Handler(mux))
}

// ServeTheme writes the ThemeInfo of s.Theme.
func (s *Server) ServeTheme(w http.ResponseWriter, _ *http.Request) {
	th := s.Theme
	writeJSON(w, ThemeInfo{
		Name:   th.Name,
		Fg:     Hex(th.ResolveTextBase()),
		Bg:     Hex(th.Bg),
		Groups: ThemePalette(th),
	})
}

// ServeLibrary writes the LibraryInfo of the golden files in the library.
func (s *Server) ServeLibrary(w http.ResponseWriter, _ *http.Request) {
	files, colors, err := s.Library.Catalog()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, LibraryInfo{Files: files, Colors: colors})
}

// ServeOpen writes the Design of the golden file that the path parameter
// names, or of its step numbered by the step parameter.
func (s *Server) ServeOpen(w http.ResponseWriter, r *http.Request) {
	snaps, err := s.Library.Load(r.FormValue("path"))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeStep(w, r, snaps)
}

// ServeParse writes the Design of the golden frames or sequence document in
// the request body, or of its step numbered by the step parameter.
func (s *Server) ServeParse(w http.ResponseWriter, r *http.Request) {
	doc, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxBody))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	snaps, err := ParseDoc(string(doc))
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	writeStep(w, r, snaps)
}

// ServeFrames writes the golden frames document of the Design in the
// request body.
func (s *Server) ServeFrames(w http.ResponseWriter, r *http.Request) {
	d, ok := readDesign(w, r)
	if !ok {
		return
	}
	doc, err := d.Frames()
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	writeText(w, doc)
}

// ServeANSI writes the Design in the request body as Design.ANSI does.
func (s *Server) ServeANSI(w http.ResponseWriter, r *http.Request) {
	d, ok := readDesign(w, r)
	if !ok {
		return
	}
	text, err := d.ANSI()
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	writeText(w, text)
}

// SaveResult is what POST /api/save returns.
type SaveResult struct {
	// Path is the file written, relative to the library's root.
	Path string `json:"path"`
	// Frames is the document written.
	Frames string `json:"frames"`
}

// ServeSave writes the golden frames document of the Design in the request
// body to the library file that the path parameter names. It fails with 409
// Conflict when the file exists, unless the overwrite parameter is 1.
func (s *Server) ServeSave(w http.ResponseWriter, r *http.Request) {
	d, ok := readDesign(w, r)
	if !ok {
		return
	}
	doc, err := d.Frames()
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	p := r.FormValue("path")
	err = s.Library.Save(p, doc, r.FormValue("overwrite") == "1")
	switch {
	case errors.Is(err, fs.ErrExist):
		http.Error(w, p+" exists", http.StatusConflict)
	case errors.Is(err, ErrBadPath):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	default:
		writeJSON(w, SaveResult{Path: p, Frames: doc})
	}
}

// readDesign decodes the Design in the request body. On failure it writes
// the error and returns false.
func readDesign(w http.ResponseWriter, r *http.Request) (Design, bool) {
	var d Design
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBody)).Decode(&d); err != nil {
		http.Error(w, "design: "+err.Error(), http.StatusBadRequest)
		return d, false
	}
	return d, true
}

// writeStep writes the Design of the snapshot that the step parameter
// numbers, the first by default.
func writeStep(w http.ResponseWriter, r *http.Request, snaps []testutil.Snapshot) {
	step := 0
	if v := r.FormValue("step"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n >= len(snaps) {
			http.Error(w, fmt.Sprintf("step %q is not between 0 and %d", v, len(snaps)-1), http.StatusBadRequest)
			return
		}
		step = n
	}
	writeJSON(w, NewDesign(snaps[step].Buf, snaps[step].Names))
}

// writeJSON writes v as a JSON response.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}

// writeText writes text as a plain text response.
func writeText(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	io.WriteString(w, text)
}

// LocalOnly serves requests through h only when their Host names a loopback
// address, so that a page which points its own domain at this machine
// cannot reach the server through the browser.
func LocalOnly(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			http.Error(w, "host "+r.Host+" is not loopback", http.StatusForbidden)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// ListenAndServe serves s on port of localhost until ctx is done.
func ListenAndServe(ctx context.Context, s *Server, port int) error {
	srv := &http.Server{
		Addr:              net.JoinHostPort("127.0.0.1", strconv.Itoa(port)),
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()
	fmt.Printf("design editor on http://localhost:%d, files under %s (Ctrl-C stops it)\n", port, s.Library.Root.Name())
	if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
