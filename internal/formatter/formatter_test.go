package formatter

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

func TestFormat_PreservesFieldShorthandsAndRanksFieldsAttribute(t *testing.T) {
	source := []byte(`bounded_context "clinic" {
  field_type "pet_id" { type = "UUID" }
  field_type "pet_name" { type = "String" }

  event "pet_added" {
    from   = [event.clinic.pet_added]
    fields = [field_type.pet_name]
    title  = "Pet Added"

    field "pet_id" {}
  }
}
`)

	once, diagnostics := Format("model.em.hcl", source)
	if diagnostics.HasErrors() {
		t.Fatalf("first diagnostics = %s", diagnostics.Error())
	}
	twice, diagnostics := Format("model.em.hcl", once)
	if diagnostics.HasErrors() {
		t.Fatalf("second diagnostics = %s", diagnostics.Error())
	}
	if !bytes.Equal(once, twice) {
		t.Fatalf("format is not idempotent:\n%s", once)
	}

	got := string(once)
	if !strings.Contains(got, "fields = [field_type.pet_name]") {
		t.Fatalf("list shorthand not preserved:\n%s", got)
	}
	if strings.Count(got, "type = ") != 2 {
		t.Fatalf("empty-body field was expanded:\n%s", got)
	}
	title := strings.Index(got, "title")
	fields := strings.Index(got, "fields")
	from := strings.Index(got, "from")
	if !(title < fields && fields < from) {
		t.Fatalf("attribute order = title:%d fields:%d from:%d, want title < fields < from:\n%s", title, fields, from, got)
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
