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

// maxBody is the size of the largest request body the server reads.
const maxBody = 16 << 20

// server serves the editor page and the API it calls.
type server struct {
	library library
	// theme is the theme whose roles the palette offers.
	theme *style.Theme
}

// themeInfo is the theme as GET /api/theme returns it.
type themeInfo struct {
	// Name is the theme's name.
	Name string `json:"name"`
	// Fg colors text by default, and Bg shows behind transparent cells.
	Fg string `json:"fg"`
	Bg string `json:"bg"`
	// Groups holds the theme's colors by role.
	Groups []swatchGroup `json:"groups"`
}

// libraryInfo is the library as GET /api/library returns it.
type libraryInfo struct {
	Files  []fileEntry `json:"files"`
	Colors []usedColor `json:"colors"`
}

// handler returns the handler of the editor and its API:
//
//	GET  /                  the editor page; ?open=p opens golden file p in it
//	GET  /api/theme         themeInfo
//	GET  /api/library       libraryInfo
//	GET  /api/open?path=p   the design of golden file p; step=n picks a sequence step
//	POST /api/parse         the design of the golden document in the body; step=n as above
//	POST /api/frames        the golden frames document of the design in the body
//	POST /api/ansi          the design in the body as text with escape sequences
//	POST /api/save?path=p   writes the design in the body to p; overwrite=1 replaces it
//
// It serves only requests to a loopback host from the same origin.
func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(page)
	})
	mux.HandleFunc("GET /api/theme", s.serveTheme)
	mux.HandleFunc("GET /api/library", s.serveLibrary)
	mux.HandleFunc("GET /api/open", s.serveOpen)
	mux.HandleFunc("POST /api/parse", s.serveParse)
	mux.HandleFunc("POST /api/frames", s.serveFrames)
	mux.HandleFunc("POST /api/ansi", s.serveANSI)
	mux.HandleFunc("POST /api/save", s.serveSave)
	return localOnly(http.NewCrossOriginProtection().Handler(mux))
}

// serveTheme writes the themeInfo of s.theme.
func (s *server) serveTheme(w http.ResponseWriter, _ *http.Request) {
	th := s.theme
	writeJSON(w, themeInfo{
		Name:   th.Name,
		Fg:     formatHex(th.ResolveTextBase()),
		Bg:     formatHex(th.Bg),
		Groups: themePalette(th),
	})
}

// serveLibrary writes the libraryInfo of the golden files in the library.
func (s *server) serveLibrary(w http.ResponseWriter, _ *http.Request) {
	files, colors, err := s.library.catalog()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, libraryInfo{Files: files, Colors: colors})
}

// serveOpen writes the design of the golden file that the path parameter
// names, or of its step numbered by the step parameter.
func (s *server) serveOpen(w http.ResponseWriter, r *http.Request) {
	snaps, err := s.library.load(r.FormValue("path"))
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

// serveParse writes the design of the golden frames or sequence document in
// the request body, or of its step numbered by the step parameter.
func (s *server) serveParse(w http.ResponseWriter, r *http.Request) {
	doc, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	snaps, err := parseDoc(string(doc))
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	writeStep(w, r, snaps)
}

// serveFrames writes the golden frames document of the design in the
// request body.
func (s *server) serveFrames(w http.ResponseWriter, r *http.Request) {
	d, ok := readDesign(w, r)
	if !ok {
		return
	}
	doc, err := d.frames()
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	writeText(w, doc)
}

// serveANSI writes the design in the request body as design.ansi does.
func (s *server) serveANSI(w http.ResponseWriter, r *http.Request) {
	d, ok := readDesign(w, r)
	if !ok {
		return
	}
	text, err := d.ansi()
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	writeText(w, text)
}

// saveResult is what POST /api/save returns.
type saveResult struct {
	// Path is the file written, relative to the library's root.
	Path string `json:"path"`
	// Frames is the document written.
	Frames string `json:"frames"`
}

// serveSave writes the golden frames document of the design in the request
// body to the library file that the path parameter names. It fails with 409
// Conflict when the file exists, unless the overwrite parameter is 1.
func (s *server) serveSave(w http.ResponseWriter, r *http.Request) {
	d, ok := readDesign(w, r)
	if !ok {
		return
	}
	doc, err := d.frames()
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	p := r.FormValue("path")
	err = s.library.save(p, doc, r.FormValue("overwrite") == "1")
	switch {
	case errors.Is(err, fs.ErrExist):
		http.Error(w, p+" exists", http.StatusConflict)
	case errors.Is(err, errBadPath):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	default:
		writeJSON(w, saveResult{Path: p, Frames: doc})
	}
}

// readDesign decodes the design in the request body. On failure it writes
// the error and returns false.
func readDesign(w http.ResponseWriter, r *http.Request) (design, bool) {
	var d design
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody)).Decode(&d); err != nil {
		http.Error(w, "design: "+err.Error(), http.StatusBadRequest)
		return d, false
	}
	return d, true
}

// writeStep writes the design of the snapshot that the step parameter
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
	writeJSON(w, newDesign(snaps[step].Buf, snaps[step].Names))
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

// localOnly serves requests through h only when their Host names a loopback
// address, so that a page which points its own domain at this machine
// cannot reach the server through the browser.
func localOnly(h http.Handler) http.Handler {
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

// listenAndServe serves s on port of localhost until ctx is done.
func listenAndServe(ctx context.Context, s *server, port int) error {
	srv := &http.Server{
		Addr:              net.JoinHostPort("127.0.0.1", strconv.Itoa(port)),
		Handler:           s.handler(),
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
	fmt.Printf("design editor on http://localhost:%d, files under %s (Ctrl-C stops it)\n", port, s.library.root.Name())
	if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
