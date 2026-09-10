package validator

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
)

func TestSmellCode(t *testing.T) {
	tests := map[string]diagnosticCode{
		"Bed anti-pattern":         codeBedAntiPattern,
		"Left chair anti-pattern":  codeLeftChairAntiPattern,
		"Right chair anti-pattern": codeRightChairAntiPattern,
		"something else":           codeUnclassified,
	}
	for summary, want := range tests {
		t.Run(summary, func(t *testing.T) {
			if got := smellCode(summary); got != want {
				t.Fatalf("smellCode(%q) = %v, want %v", summary, got, want)
			}
		})
	}
}

func TestValidateShelfSmell(t *testing.T) {
	rangeFor := func(line int) hcl.Range {
		return hcl.Range{Filename: "model.em.hcl", Start: hcl.Pos{Line: line}, End: hcl.Pos{Line: line}}
	}

	t.Run("fewer than two workflows never warns", func(t *testing.T) {
		v := &modelValidator{index: modelIndex{
			workflows:      map[string]workflowIndex{"only": {}},
			scenarioCounts: map[string]int{"only": 5},
		}}
		if diagnostics := v.validateShelfSmell(); diagnostics != nil {
			t.Fatalf("expected no diagnostics, got %v", diagnostics)
		}
	})

	t.Run("fewer than two total scenarios never warns", func(t *testing.T) {
		v := &modelValidator{index: modelIndex{
			workflows:      map[string]workflowIndex{"a": {}, "b": {}},
			scenarioCounts: map[string]int{"a": 1, "b": 0},
		}}
		if diagnostics := v.validateShelfSmell(); diagnostics != nil {
			t.Fatalf("expected no diagnostics, got %v", diagnostics)
		}
	})

	t.Run("scenarios spread across workflows never warns", func(t *testing.T) {
		v := &modelValidator{index: modelIndex{
			workflows:      map[string]workflowIndex{"a": {}, "b": {}},
			scenarioCounts: map[string]int{"a": 1, "b": 1},
		}}
		if diagnostics := v.validateShelfSmell(); diagnostics != nil {
			t.Fatalf("expected no diagnostics, got %v", diagnostics)
		}
	})

	t.Run("one workflow hoarding every scenario warns once", func(t *testing.T) {
		v := &modelValidator{index: modelIndex{
			workflows:        map[string]workflowIndex{"add_pet": {}, "edit_pet": {}},
			scenarioCounts:   map[string]int{"add_pet": 2, "edit_pet": 0},
			definitionRanges: map[string]hcl.Range{"add_pet": rangeFor(3), "edit_pet": rangeFor(9)},
		}}
		diagnostics := v.validateShelfSmell()
		if len(diagnostics) != 1 {
			t.Fatalf("expected exactly one diagnostic, got %d: %v", len(diagnostics), diagnostics)
		}
		got := diagnostics[0]
		if got.Severity != hcl.DiagWarning {
			t.Errorf("severity = %v, want DiagWarning", got.Severity)
		}
		if DiagnosticCode(got) != string(codeShelfAntiPattern) {
			t.Errorf("code = %s, want %s", DiagnosticCode(got), codeShelfAntiPattern)
		}
		if got.Summary != "Shelf anti-pattern" {
			t.Errorf("summary = %q, want %q", got.Summary, "Shelf anti-pattern")
		}
		if got.Subject == nil || *got.Subject != rangeFor(3) {
			t.Errorf("subject = %v, want the owning workflow's definition range %v", got.Subject, rangeFor(3))
		}
	})
}
