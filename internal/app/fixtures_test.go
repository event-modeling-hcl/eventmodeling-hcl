package app

import (
	"os"
	"path/filepath"
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
			if result.Diagnostics.HasErrors() {
				t.Fatalf("diagnostics = %#v, want no errors", result.Diagnostics)
			}
			if result.HTML == "" {
				t.Fatal("HTML is empty for valid example")
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
			if !result.Diagnostics.HasErrors() {
				t.Fatalf("diagnostics = %#v, want an error", result.Diagnostics)
			}
			if result.HTML != "" {
				t.Fatal("HTML is non-empty for invalid fixture")
			}
		})
	}
}
