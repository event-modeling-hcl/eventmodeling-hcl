package replcore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRender_RendersEveryShippedExample(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "examples", "*.em.hcl"))
	if err != nil {
		t.Fatalf("find examples: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no example fixtures found")
	}

	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			source, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("read example: %v", readErr)
			}

			result := Render(path, source, Valid)

			for _, diagnostic := range result.Diagnostics {
				if diagnostic.Severity == "Error" {
					t.Errorf("unexpected error diagnostic: %+v", diagnostic)
				}
			}
			if result.HTML == "" {
				t.Fatal("expected non-empty HTML for a valid example")
			}
			if !strings.Contains(result.HTML, "<!doctype html>") {
				t.Error("HTML does not look like a complete document")
			}
		})
	}
}

func TestRender_RefusesEveryInvalidFixture(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "testdata", "invalid", "*.em.hcl"))
	if err != nil {
		t.Fatalf("find invalid fixtures: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no invalid fixtures found")
	}

	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			source, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("read fixture: %v", readErr)
			}

			result := Render(path, source, Valid)

			if result.HTML != "" {
				t.Error("expected no HTML for an invalid model")
			}
			hasError := false
			for _, diagnostic := range result.Diagnostics {
				if diagnostic.Severity == "Error" {
					hasError = true
				}
			}
			if !hasError {
				t.Error("expected at least one error diagnostic")
			}
		})
	}
}

// A couple of fixtures get an exact-code check, not just "some error" — the
// same two the website's Learn pages show as real, verified CLI rejections
// (event-modeling-hcl.github.io/learn/{language,scenarios}.html), so a
// regression here would also be a regression in what that site teaches.
func TestRender_ReportsTheDocumentedCodeForKnownFixtures(t *testing.T) {
	tests := []struct {
		fixture      string
		expectedCode string
	}{
		{"reverse-flow.em.hcl", "EM201"},
		{"state-view-when.em.hcl", "EM301"},
	}

	for _, test := range tests {
		t.Run(test.fixture, func(t *testing.T) {
			path := filepath.Join("..", "..", "testdata", "invalid", test.fixture)
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}

			result := Render(path, source, Valid)

			found := false
			for _, diagnostic := range result.Diagnostics {
				if diagnostic.Code == test.expectedCode {
					found = true
				}
			}
			if !found {
				t.Errorf("expected diagnostic code %s, got %+v", test.expectedCode, result.Diagnostics)
			}
		})
	}
}

func TestRender_WarningsDoNotBlockRendering(t *testing.T) {
	// complete.em.hcl ships with a deliberate, documented open hotspot
	// (EM406, a warning under the default Valid profile) — confirming it
	// still renders is what distinguishes "refuses on error" from
	// "refuses on any diagnostic at all".
	path := filepath.Join("..", "..", "examples", "complete.em.hcl")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example: %v", err)
	}

	result := Render(path, source, Valid)

	if result.HTML == "" {
		t.Fatal("expected HTML despite warning-level diagnostics")
	}
	sawWarning := false
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Severity == "Warning" {
			sawWarning = true
		}
	}
	if !sawWarning {
		t.Error("expected complete.em.hcl to still carry its documented warning")
	}
}

func TestFormat_CanonicalizesAValidModel(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "minimal.em.hcl")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example: %v", err)
	}

	result := Format(path, source)

	if len(result.Diagnostics) != 0 {
		t.Errorf("unexpected diagnostics: %+v", result.Diagnostics)
	}
	if result.Source == "" {
		t.Fatal("expected non-empty formatted source")
	}
	if !strings.Contains(result.Source, "bounded_context") {
		t.Error("formatted source lost recognizable content")
	}
}

func TestFormat_ReportsSyntaxErrorsWithoutSource(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "invalid", "malformed-syntax.em.hcl")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	result := Format(path, source)

	if result.Source != "" {
		t.Error("expected no formatted source for malformed syntax")
	}
	if len(result.Diagnostics) == 0 {
		t.Error("expected at least one diagnostic")
	}
}

// ParseProfile lets callers (cmd/wasm, internal/serve) turn a plain string
// into a Profile without importing internal/validator themselves — the same
// "callers only know replcore" boundary Profile's re-exported constants
// already establish.
func TestParseProfile_RecognizesEachValidName(t *testing.T) {
	tests := []struct {
		name     string
		expected Profile
	}{
		{"workshop", Workshop},
		{"valid", Valid},
		{"strict", Strict},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profile, ok := ParseProfile(test.name)
			if !ok {
				t.Fatalf("ParseProfile(%q) reported not ok", test.name)
			}
			if profile != test.expected {
				t.Errorf("ParseProfile(%q) = %v, want %v", test.name, profile, test.expected)
			}
		})
	}
}

func TestParseProfile_RejectsUnknownName(t *testing.T) {
	profile, ok := ParseProfile("bogus")
	if ok {
		t.Fatalf("ParseProfile(\"bogus\") reported ok")
	}
	if profile != Valid {
		t.Errorf("ParseProfile(\"bogus\") = %v, want the documented Valid fallback", profile)
	}
}
