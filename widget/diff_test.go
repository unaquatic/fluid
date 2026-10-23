package widget

import (
	"testing"

	"github.com/unaquatic/fluid/buffer"
	"github.com/unaquatic/fluid/testutil"
)

// Golden-test helpers.

func setupDiff(patch string, w, h int, opts ...DiffOption) *Diff {
	d := NewDiff(patch, opts...)
	testutil.SetupWidget(d, w, h)
	return d
}

func renderDiff(patch string, w, h int, opts ...DiffOption) *buffer.Buffer {
	d := setupDiff(patch, w, h, opts...)
	buf := buffer.New(w, h)
	d.RenderSelf(buf, 0)
	return buf
}

// Shared test patches.

const simplePatch = "@@ -1,3 +1,3 @@\n context\n-old line\n+new line"

const multiHunkPatch = `@@ -1,3 +1,3 @@
 line1
-old1
+new1
@@ -10,3 +10,3 @@
 line10
-old10
+new10`

const fullHeaderPatch = `diff --git a/main.go b/main.go
index abc1234..def5678 100644
--- a/main.go
+++ b/main.go
@@ -1,4 +1,5 @@
 package main
+import "fmt"
 func main() {
-    println("hello")
+    fmt.Println("hello")
 }`

const unevenPatch = "@@ -1,4 +1,3 @@\n context\n-remove1\n-remove2\n+add1"

const inlinePatch = "@@ -1,2 +1,2 @@\n-hello world\n+hello earth"

// longPatch exercises large line numbers, long content, and multiple hunks.
const longPatch = `diff --git a/server/handler.go b/server/handler.go
index 1a2b3c4..5d6e7f8 100644
--- a/server/handler.go
+++ b/server/handler.go
@@ -12,8 +12,10 @@
 import (
 	"context"
 	"encoding/json"
+	"errors"
 	"fmt"
 	"net/http"
+	"time"
 )
 
 type Server struct {
@@ -45,14 +47,21 @@
 func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) {
 	ctx := r.Context()
-	data, err := s.fetchData(ctx)
+	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
+	defer cancel()
+
+	data, err := s.fetchData(ctx, r.URL.Query())
 	if err != nil {
-		http.Error(w, "internal error", http.StatusInternalServerError)
+		if errors.Is(err, context.DeadlineExceeded) {
+			http.Error(w, "request timeout", http.StatusGatewayTimeout)
+		} else {
+			http.Error(w, "internal error", http.StatusInternalServerError)
+		}
 		return
 	}
 
-	w.Header().Set("Content-Type", "application/json")
+	w.Header().Set("Content-Type", "application/json; charset=utf-8")
+	w.Header().Set("Cache-Control", "no-cache")
 	json.NewEncoder(w).Encode(data)
 }
@@ -102,7 +111,9 @@
 func (s *Server) Shutdown(ctx context.Context) error {
 	s.mu.Lock()
 	defer s.mu.Unlock()
-	return s.listener.Close()
+	s.logger.Info("shutting down server")
+	s.healthy = false
+	return s.gracefulShutdown(ctx)
 }
 
 func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {`

const longPatchManyLines = `diff --git a/pkg/processor/transform.go b/pkg/processor/transform.go
--- a/pkg/processor/transform.go
+++ b/pkg/processor/transform.go
@@ -88,30 +88,35 @@
 func (p *Processor) Transform(input []Record) ([]Record, error) {
 	results := make([]Record, 0, len(input))
-	for _, rec := range input {
-		if rec.IsEmpty() {
-			continue
-		}
-		transformed, err := p.applyRules(rec)
-		if err != nil {
-			p.logger.Warn("skipping record", "id", rec.ID, "error", err)
-			p.metrics.IncrementSkipped()
-			continue
-		}
-		if transformed.Score < p.threshold {
-			p.metrics.IncrementFiltered()
-			continue
-		}
-		results = append(results, transformed)
-		p.metrics.IncrementProcessed()
-	}
+	errCh := make(chan error, len(input))
+	resultCh := make(chan Record, len(input))
+
+	for _, rec := range input {
+		go func(r Record) {
+			if r.IsEmpty() {
+				return
+			}
+			transformed, err := p.applyRules(r)
+			if err != nil {
+				p.logger.Warn("skipping record", "id", r.ID, "error", err)
+				p.metrics.IncrementSkipped()
+				errCh <- err
+				return
+			}
+			if transformed.Score < p.threshold {
+				p.metrics.IncrementFiltered()
+				return
+			}
+			resultCh <- transformed
+			p.metrics.IncrementProcessed()
+		}(rec)
+	}
 
-	sort.Slice(results, func(i, j int) bool {
-		return results[i].Score > results[j].Score
-	})
+	close(resultCh)
+	for r := range resultCh {
+		results = append(results, r)
+	}
 
 	p.logger.Info("transform complete", "total", len(results))
 	return results, nil
 }`

// Plain-text golden tests.

func TestDiffGolden_UnifiedBasic(t *testing.T) {
	buf := renderDiff(simplePatch, 40, 6)
	testutil.AssertGoldenFrames(t, "diff_unified_basic", buf, nil)
}

func TestDiffGolden_UnifiedFullHeaders(t *testing.T) {
	buf := renderDiff(fullHeaderPatch, 50, 12)
	testutil.AssertGoldenFrames(t, "diff_unified_full_headers", buf, nil)
}

func TestDiffGolden_UnifiedNoGutter(t *testing.T) {
	buf := renderDiff(simplePatch, 30, 6,
		DiffWithShowLineNumbers(false),
		DiffWithShowSigns(false),
	)
	testutil.AssertGoldenFrames(t, "diff_unified_no_gutter", buf, nil)
}

func TestDiffGolden_UnifiedNoSigns(t *testing.T) {
	buf := renderDiff(simplePatch, 30, 6,
		DiffWithShowSigns(false),
	)
	testutil.AssertGoldenFrames(t, "diff_unified_no_signs", buf, nil)
}

func TestDiffGolden_UnifiedMultiHunk(t *testing.T) {
	buf := renderDiff(multiHunkPatch, 45, 10)
	testutil.AssertGoldenFrames(t, "diff_unified_multi_hunk", buf, nil)
}

func TestDiffGolden_UnifiedScrolled(t *testing.T) {
	d := setupDiff(fullHeaderPatch, 50, 5)
	d.SetScrollY(3) // scroll past the first 3 lines
	buf := buffer.New(50, 5)
	d.RenderSelf(buf, 0)
	testutil.AssertGoldenFrames(t, "diff_unified_scrolled", buf, nil)
}

func TestDiffGolden_SplitBasic(t *testing.T) {
	buf := renderDiff(simplePatch, 60, 6, DiffWithViewMode(DiffSplit))
	testutil.AssertGoldenFrames(t, "diff_split_basic", buf, nil)
}

func TestDiffGolden_SplitPadding(t *testing.T) {
	buf := renderDiff(unevenPatch, 60, 6, DiffWithViewMode(DiffSplit))
	testutil.AssertGoldenFrames(t, "diff_split_padding", buf, nil)
}

func TestDiffGolden_SplitMultiHunk(t *testing.T) {
	buf := renderDiff(multiHunkPatch, 70, 10, DiffWithViewMode(DiffSplit))
	testutil.AssertGoldenFrames(t, "diff_split_multi_hunk", buf, nil)
}

func TestDiffGolden_EmptyPatch(t *testing.T) {
	buf := renderDiff("", 30, 4)
	testutil.AssertGoldenFrames(t, "diff_empty", buf, nil)
}

func TestDiffGolden_SplitFullHeaders(t *testing.T) {
	buf := renderDiff(fullHeaderPatch, 80, 12, DiffWithViewMode(DiffSplit))
	testutil.AssertGoldenFrames(t, "diff_split_full_headers", buf, nil)
}

// Large-diff golden tests.

func TestDiffGolden_LongUnified(t *testing.T) {
	buf := renderDiff(longPatch, 70, 30)
	testutil.AssertGoldenFrames(t, "diff_long_unified", buf, nil)
}

func TestDiffGolden_LongSplit(t *testing.T) {
	buf := renderDiff(longPatch, 100, 30, DiffWithViewMode(DiffSplit))
	testutil.AssertGoldenFrames(t, "diff_long_split", buf, nil)
}

func TestDiffGolden_LongManyLinesUnified(t *testing.T) {
	buf := renderDiff(longPatchManyLines, 70, 40)
	testutil.AssertGoldenFrames(t, "diff_long_many_lines_unified", buf, nil)
}

func TestDiffGolden_LongManyLinesSplit(t *testing.T) {
	buf := renderDiff(longPatchManyLines, 100, 40, DiffWithViewMode(DiffSplit))
	testutil.AssertGoldenFrames(t, "diff_long_many_lines_split", buf, nil)
}

func TestDiffGolden_LongScrollSequence(t *testing.T) {
	d := setupDiff(longPatch, 70, 10)
	seq := testutil.NewGoldenSequence()

	// Capture the first viewport.
	buf := buffer.New(70, 10)
	d.RenderSelf(buf, 0)
	seq.AddFrames("top", buf, nil)

	// Capture an interior viewport.
	d.SetScrollY(d.MaxScrollY() / 2)
	buf = buffer.New(70, 10)
	d.RenderSelf(buf, 0)
	seq.AddFrames("middle", buf, nil)

	// Capture the final viewport.
	d.ScrollToBottom()
	buf = buffer.New(70, 10)
	d.RenderSelf(buf, 0)
	seq.AddFrames("bottom", buf, nil)

	testutil.AssertGoldenSequence(t, "diff_long_scroll_sequence", seq)
}

func TestDiffGolden_LongSplitStyled(t *testing.T) {
	buf := renderDiff(longPatch, 100, 15, DiffWithViewMode(DiffSplit))
	testutil.AssertGoldenFrames(t, "diff_long_split_short", buf, nil)
}

func TestDiffGolden_LongNarrowUnified(t *testing.T) {
	// A narrow unified viewport exposes long-line truncation.
	buf := renderDiff(longPatch, 40, 20)
	testutil.AssertGoldenFrames(t, "diff_long_narrow_unified", buf, nil)
}

func TestDiffGolden_LongNarrowSplit(t *testing.T) {
	// A narrow split viewport exposes pane truncation.
	buf := renderDiff(longPatch, 50, 20, DiffWithViewMode(DiffSplit))
	testutil.AssertGoldenFrames(t, "diff_long_narrow_split", buf, nil)
}

// Styled golden tests.

func TestDiffGolden_UnifiedStyled(t *testing.T) {
	buf := renderDiff(simplePatch, 40, 6)
	testutil.AssertGoldenFrames(t, "diff_unified_basic", buf, nil)
}

func TestDiffGolden_SplitStyled(t *testing.T) {
	buf := renderDiff(simplePatch, 60, 6, DiffWithViewMode(DiffSplit))
	testutil.AssertGoldenFrames(t, "diff_split_basic", buf, nil)
}

func TestDiffGolden_SplitInlineStyled(t *testing.T) {
	buf := renderDiff(inlinePatch, 60, 4, DiffWithViewMode(DiffSplit))
	testutil.AssertGoldenFrames(t, "diff_split_inline", buf, nil)
}

// State-transition golden tests.

func TestDiffGolden_ViewModeSwitch(t *testing.T) {
	d := setupDiff(simplePatch, 60, 6)
	seq := testutil.NewGoldenSequence()

	// Capture the initial unified view.
	buf := buffer.New(60, 6)
	d.RenderSelf(buf, 0)
	seq.AddFrames("unified", buf, nil)

	// Capture the same patch after switching to split view.
	d.SetViewMode(DiffSplit)
	buf = buffer.New(60, 6)
	d.RenderSelf(buf, 0)
	seq.AddFrames("split", buf, nil)

	testutil.AssertGoldenSequence(t, "diff_view_mode_switch", seq)
}

func TestDiffGolden_ScrollSequence(t *testing.T) {
	d := setupDiff(fullHeaderPatch, 50, 4)
	seq := testutil.NewGoldenSequence()

	// Capture the initial top viewport.
	buf := buffer.New(50, 4)
	d.RenderSelf(buf, 0)
	seq.AddFrames("scroll_0", buf, nil)

	// Capture an offset viewport.
	d.SetScrollY(2)
	buf = buffer.New(50, 4)
	d.RenderSelf(buf, 0)
	seq.AddFrames("scroll_2", buf, nil)

	// Capture the final viewport.
	d.ScrollToBottom()
	buf = buffer.New(50, 4)
	d.RenderSelf(buf, 0)
	seq.AddFrames("scroll_bottom", buf, nil)

	testutil.AssertGoldenSequence(t, "diff_scroll_sequence", seq)
}

func TestDiffGolden_PatchChange(t *testing.T) {
	d := setupDiff(simplePatch, 40, 6)
	seq := testutil.NewGoldenSequence()

	// Capture the original patch.
	buf := buffer.New(40, 6)
	d.RenderSelf(buf, 0)
	seq.AddFrames("initial", buf, nil)

	// Capture the replacement patch.
	d.SetPatch(inlinePatch)
	buf = buffer.New(40, 6)
	d.RenderSelf(buf, 0)
	seq.AddFrames("after_change", buf, nil)

	testutil.AssertGoldenSequence(t, "diff_patch_change", seq)
}
