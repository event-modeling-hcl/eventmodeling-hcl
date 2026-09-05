package formatter

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestFormat_OrdersAttributesAndPreservesTraversals(t *testing.T) {
	source := []byte(`state_change "add_pet" {
  command "add_pet" {
    to = [event.clinic.pet_added]
    tags = ["write"]
    aggregate = aggregate.clinic.pet
    title = "Add Pet"
  }
}
`)

	formatted, diagnostics := Format("model.em.hcl", source)

	if diagnostics.HasErrors() {
		t.Fatalf("diagnostics = %s", diagnostics.Error())
	}
	const want = `state_change "add_pet" {
  command "add_pet" {
    title     = "Add Pet"
    tags      = ["write"]
    aggregate = aggregate.clinic.pet
    to        = [event.clinic.pet_added]
  }
}
`
	if got := string(formatted); got != want {
		t.Fatalf("formatted = %q, want %q", got, want)
	}
}

func TestFormat_IsIdempotentForShippedExamples(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "examples", "*.em.hcl"))
	if err != nil {
		t.Fatalf("find examples: %v", err)
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read example: %v", err)
			}
			once, diagnostics := Format(path, source)
			if diagnostics.HasErrors() {
				t.Fatalf("first diagnostics = %s", diagnostics.Error())
			}
			twice, diagnostics := Format(path, once)
			if diagnostics.HasErrors() {
				t.Fatalf("second diagnostics = %s", diagnostics.Error())
			}
			if !bytes.Equal(once, twice) {
				t.Fatal("format is not idempotent")
			}
		})
	}
}

func TestFormat_PreservesLeadingComments(t *testing.T) {
	source := []byte("# Model comment\nstate_change \"example\" {}\n")

	formatted, diagnostics := Format("model.em.hcl", source)

	if diagnostics.HasErrors() {
		t.Fatalf("diagnostics = %s", diagnostics.Error())
	}
	if got, want := string(formatted[:len("# Model comment\n")]), "# Model comment\n"; got != want {
		t.Fatalf("leading comment = %q, want %q", got, want)
	}
}
