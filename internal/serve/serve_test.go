package serve

import (
	"context"
	"net"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/replcore"
)

func TestServingURLUsesTheActualBoundPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	got := servingURL(listener, "127.0.0.1")
	if strings.HasSuffix(got, ":0") {
		t.Fatalf("serving URL retained requested port zero: %q", got)
	}
	if !strings.HasPrefix(got, "http://127.0.0.1:") {
		t.Fatalf("serving URL = %q", got)
	}
}

func validResult(t *testing.T) replcore.RenderResult {
	t.Helper()
	path := filepath.Join("..", "..", "examples", "minimal.em.hcl")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example: %v", err)
	}
	return replcore.Render(path, source, replcore.Valid)
}

func invalidResult(t *testing.T) replcore.RenderResult {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "invalid", "reverse-flow.em.hcl")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return replcore.Render(path, source, replcore.Valid)
}

// Behavior 1: the served hash changes iff the servable content changes.

func TestState_HashIsStableForTheSameResult(t *testing.T) {
	s := &state{}
	result := validResult(t)

	s.update(result)
	_, hash1 := s.snapshot()
	s.update(result)
	_, hash2 := s.snapshot()

	if hash1 != hash2 {
		t.Errorf("hash changed for an identical result: %q -> %q", hash1, hash2)
	}
}

func TestState_HashChangesWhenHTMLChanges(t *testing.T) {
	s := &state{}

	s.update(validResult(t))
	_, hash1 := s.snapshot()
	s.update(replcore.RenderResult{HTML: "<html>different</html>"})
	_, hash2 := s.snapshot()

	if hash1 == hash2 {
		t.Error("hash did not change when HTML changed")
	}
}

func TestState_HashChangesWhenDiagnosticsChangeEvenWithEmptyHTML(t *testing.T) {
	// Two different error states both render no HTML at all — hashing only
	// the HTML would make them indistinguishable, so the served page would
	// never reload from one error to a different one.
	s := &state{}

	s.update(invalidResult(t))
	_, hash1 := s.snapshot()
	s.update(replcore.RenderResult{Diagnostics: []replcore.Diagnostic{
		{Code: "EM999", Severity: "Error", Summary: "a different problem"},
	}})
	_, hash2 := s.snapshot()

	if hash1 == hash2 {
		t.Error("hash did not change between two different error results")
	}
}

// Behaviors 2, 3, 4, 5: the real production mux, exercised end to end.

func TestMux_RootServesShellPageWithIframeAndPollScript(t *testing.T) {
	s := &state{}
	s.update(validResult(t))

	rec := httptest.NewRecorder()
	newMux(s).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<iframe src="/diagram">`) {
		t.Error("shell page missing the diagram iframe")
	}
	if !strings.Contains(body, `fetch("/hash")`) {
		t.Error("shell page missing the poll-reload script")
	}
}

func TestMux_DiagramServesRenderedHTMLVerbatimForAValidModel(t *testing.T) {
	s := &state{}
	result := validResult(t)
	s.update(result)

	rec := httptest.NewRecorder()
	newMux(s).ServeHTTP(rec, httptest.NewRequest("GET", "/diagram", nil))

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != result.HTML {
		t.Error("/diagram body does not match the rendered HTML exactly")
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html content type, got %q", ct)
	}
}

func TestMux_DiagramServesReadableDiagnosticsPageForAnInvalidModel(t *testing.T) {
	s := &state{}
	result := invalidResult(t)
	if result.HTML != "" {
		t.Fatal("test fixture unexpectedly produced HTML")
	}
	s.update(result)

	rec := httptest.NewRecorder()
	newMux(s).ServeHTTP(rec, httptest.NewRequest("GET", "/diagram", nil))

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if body == "" {
		t.Fatal("expected a non-blank diagnostics page")
	}
	for _, diagnostic := range result.Diagnostics {
		if !strings.Contains(body, diagnostic.Code) {
			t.Errorf("diagnostics page missing code %q", diagnostic.Code)
		}
		if !strings.Contains(body, diagnostic.Summary) {
			t.Errorf("diagnostics page missing summary %q", diagnostic.Summary)
		}
	}
}

func TestMux_HashReturnsTheCurrentHash(t *testing.T) {
	s := &state{}
	s.update(validResult(t))
	_, want := s.snapshot()

	rec := httptest.NewRecorder()
	newMux(s).ServeHTTP(rec, httptest.NewRequest("GET", "/hash", nil))

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != want {
		t.Errorf("/hash returned %q, want %q", got, want)
	}
}

// Behavior 6: after the watched file changes on disk, the served state
// updates to match.

func TestWatch_RegeneratesWhenTheWatchedFileChanges(t *testing.T) {
	originalPollInterval := pollInterval
	pollInterval = 20 * time.Millisecond
	defer func() { pollInterval = originalPollInterval }()

	dir := t.TempDir()
	path := filepath.Join(dir, "model.em.hcl")
	initial, err := os.ReadFile(filepath.Join("..", "..", "examples", "minimal.em.hcl"))
	if err != nil {
		t.Fatalf("read seed example: %v", err)
	}
	if err := os.WriteFile(path, initial, 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}

	s := &state{}
	if err := regenerate(s, path, replcore.Valid); err != nil {
		t.Fatalf("initial regenerate: %v", err)
	}
	_, hashBefore := s.snapshot()

	startHash, err := fileSourceHash(path)
	if err != nil {
		t.Fatalf("hash seed file: %v", err)
	}

	invalid, err := os.ReadFile(filepath.Join("..", "..", "testdata", "invalid", "reverse-flow.em.hcl"))
	if err != nil {
		t.Fatalf("read invalid fixture: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watch(ctx, s, path, replcore.Valid, startHash)
	if err := os.WriteFile(path, invalid, 0o644); err != nil {
		t.Fatalf("rewrite file: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		result, hashAfter := s.snapshot()
		if hashAfter != hashBefore {
			if result.HTML != "" {
				t.Fatal("expected the rewritten (invalid) file to clear the HTML")
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("watch did not regenerate after the watched file changed")
}

func TestWatch_RegeneratesWhenContentChangesWithoutMTimeAdvancing(t *testing.T) {
	originalPollInterval := pollInterval
	pollInterval = 20 * time.Millisecond
	defer func() { pollInterval = originalPollInterval }()

	dir := t.TempDir()
	path := filepath.Join(dir, "model.em.hcl")
	initial, err := os.ReadFile(filepath.Join("..", "..", "examples", "minimal.em.hcl"))
	if err != nil {
		t.Fatalf("read seed example: %v", err)
	}
	if err := os.WriteFile(path, initial, 0o644); err != nil {
		t.Fatalf("write seed: %v", err)
	}

	s := &state{}
	if err := regenerate(s, path, replcore.Valid); err != nil {
		t.Fatalf("initial regenerate: %v", err)
	}
	_, before := s.snapshot()
	startHash, err := fileSourceHash(path)
	if err != nil {
		t.Fatalf("hash seed: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat seed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watch(ctx, s, path, replcore.Valid, startHash)

	invalid, err := os.ReadFile(filepath.Join("..", "..", "testdata", "invalid", "reverse-flow.em.hcl"))
	if err != nil {
		t.Fatalf("read invalid fixture: %v", err)
	}
	if err := os.WriteFile(path, invalid, 0o644); err != nil {
		t.Fatalf("rewrite file: %v", err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatalf("restore mtime: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		result, after := s.snapshot()
		if after != before {
			if result.HTML != "" {
				t.Fatal("expected invalid replacement to clear HTML")
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("watch did not regenerate after a same-mtime content change")
}
