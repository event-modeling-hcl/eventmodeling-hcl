package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readExample(t *testing.T, relPath string) (string, []byte) {
	t.Helper()
	path := filepath.Join("..", "..", relPath)
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", relPath, err)
	}
	return path, source
}

func TestRender_ValidModelProducesHTMLWithNoErrorDiagnostics(t *testing.T) {
	path, source := readExample(t, "examples/minimal.em.hcl")

	result := Render(path, source, Valid)

	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Severity == "Error" {
			t.Errorf("unexpected error diagnostic: %+v", diagnostic)
		}
	}
	if result.HTML == "" {
		t.Fatal("expected non-empty HTML for a valid model")
	}
	if !strings.Contains(result.HTML, "<!doctype html>") {
		t.Error("HTML does not look like a complete document")
	}
}

func TestRender_InvalidModelReturnsExpectedDiagnosticAndEmptyHTML(t *testing.T) {
	path, source := readExample(t, "testdata/invalid/dangling-reference.em.hcl")

	result := Render(path, source, Valid)

	if result.HTML != "" {
		t.Errorf("expected empty HTML for an invalid model, got %d bytes", len(result.HTML))
	}
	if len(result.Diagnostics) == 0 {
		t.Fatal("expected at least one diagnostic")
	}
	found := false
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "EM102" && diagnostic.Severity == "Error" {
			found = true
			if diagnostic.Line == 0 {
				t.Errorf("expected a non-zero line for diagnostic %+v", diagnostic)
			}
			if diagnostic.Filename != path {
				t.Errorf("expected Filename %q, got %q", path, diagnostic.Filename)
			}
		}
	}
	if !found {
		t.Errorf("expected an EM102 error diagnostic, got %+v", result.Diagnostics)
	}
}

func TestFormat_RoundTripsAlreadyFormattedSource(t *testing.T) {
	path, source := readExample(t, "examples/minimal.em.hcl")

	result := Format(path, source)

	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Severity == "Error" {
			t.Errorf("unexpected error diagnostic: %+v", diagnostic)
		}
	}
	if result.Source == "" {
		t.Fatal("expected non-empty formatted source")
	}
	// Formatting an already-canonical document is idempotent.
	second := Format(path, []byte(result.Source))
	if second.Source != result.Source {
		t.Error("expected formatting to be idempotent")
	}
}

func TestValidate_ProfileSwitchingChangesSeverity(t *testing.T) {
	path, source := readExample(t, "examples/complete.em.hcl")

	workshop := Validate(path, source, Workshop)
	valid := Validate(path, source, Valid)
	strict := Validate(path, source, Strict)

	severityFor := func(diagnostics []Diagnostic, code string) (string, bool) {
		for _, diagnostic := range diagnostics {
			if diagnostic.Code == code {
				return diagnostic.Severity, true
			}
		}
		return "", false
	}

	workshopSeverity, ok := severityFor(workshop, "EM406")
	if !ok || workshopSeverity != "Info" {
		t.Errorf("workshop: expected EM406 Info, got %q (found=%v)", workshopSeverity, ok)
	}
	validSeverity, ok := severityFor(valid, "EM406")
	if !ok || validSeverity != "Warning" {
		t.Errorf("valid: expected EM406 Warning, got %q (found=%v)", validSeverity, ok)
	}
	strictSeverity, ok := severityFor(strict, "EM406")
	if !ok || strictSeverity != "Error" {
		t.Errorf("strict: expected EM406 Error, got %q (found=%v)", strictSeverity, ok)
	}
}

func TestValidate_PopulatesFilenameForCLIFormatting(t *testing.T) {
	path, source := readExample(t, "testdata/invalid/dangling-reference.em.hcl")

	diagnostics := Validate(path, source, Valid)

	if len(diagnostics) == 0 {
		t.Fatal("expected at least one diagnostic")
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Filename != path {
			t.Errorf("expected Filename %q, got %q", path, diagnostic.Filename)
		}
	}
}

func TestParseProfile_RecognizesKnownNames(t *testing.T) {
	cases := map[string]Profile{"workshop": Workshop, "valid": Valid, "strict": Strict}
	for name, expected := range cases {
		profile, ok := ParseProfile(name)
		if !ok || profile != expected {
			t.Errorf("ParseProfile(%q) = %v, %v; want %v, true", name, profile, ok, expected)
		}
	}
	if _, ok := ParseProfile("bogus"); ok {
		t.Error("expected ParseProfile(\"bogus\") to report false")
	}
}
