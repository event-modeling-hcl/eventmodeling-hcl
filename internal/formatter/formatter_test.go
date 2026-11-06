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

func TestFormat_SortsScreenKindFirstInSemanticGroup(t *testing.T) {
	source := []byte(`state_change "add_pet" {
  screen "add_pet_request" {
    to = [command.add_pet]
    actor = actor.clinic_staff
    api_endpoint = "POST /pets"
    kind = "api"
    tags = ["api"]
  }
  screen "add_pet_form" {
    to = [command.add_pet]
  }
}
`)
	const want = `state_change "add_pet" {
  screen "add_pet_request" {
    tags         = ["api"]
    kind         = "api"
    api_endpoint = "POST /pets"
    actor        = actor.clinic_staff
    to           = [command.add_pet]
  }

  screen "add_pet_form" {
    to = [command.add_pet]
  }
}
`
	formatted, diagnostics := Format("model.em.hcl", source)
	if diagnostics.HasErrors() {
		t.Fatalf("diagnostics = %s", diagnostics.Error())
	}
	if got := string(formatted); got != want {
		t.Fatalf("formatted = %q, want %q", got, want)
	}
	again, _ := Format("model.em.hcl", formatted)
	if !bytes.Equal(again, formatted) {
		t.Fatalf("second format = %q, want idempotent %q", again, formatted)
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

// Each block of RFC 0007 gets the attribute order of the specification, and
// the order of every list stays as written.
func TestFormat_OrdersMethodFactAttributes(t *testing.T) {
	source := []byte(`lifecycle "booking" {
  state "done" {
    path = "main"
    end  = true
  }

  transition "close" {
    after   = state.done
    before  = [state.open, state.held]
    events  = [event.booking.closed]
    command = command.close.close
  }

  invalid_transition "reopen" {
    before  = [state.done]
    reason  = "A closed booking stays closed."
    command = command.reopen.reopen
  }

  aggregate = aggregate.booking.booking
}

evidence "rule" {
  on        = state.booking.done
  source    = evidence_source.policy
  location  = "page 3"
  certainty = "stated"
}

decision "debt" {
  on        = workflow.close
  reason    = "One list."
  statement = "The list reads two events."
  signal    = "EM403"
  kind      = "accepted_debt"
}

hotspot "refund" {
  on       = workflow.close
  concerns = "scenario"
  options  = ["B", "A"]
  question = "Refund?"
  category = "rule"
  kind     = "request"
}

state_view "list" {
  field_source {
    target      = readmodel.list.field.name
    inputs      = [event.booking.closed.field.name]
    description = "Copy."
  }

  scenario "empty" {
    empty_history = true
    case          = "appear"

    then {
      readmodel          = readmodel.list
      expect_no_dispatch = true
      expect_empty_list  = true
    }
  }
}

bounded_context "booking" {
  aggregate "booking" {
    identity = field_type.booking_id
    title    = "Booking"
  }

  event "closed" {
    field "name" {
      deduplication_key = true
      pii               = true
      correlation_key   = true
      id_attribute      = true
      generated         = true
      type              = "String"
    }
  }
}
`)
	want := `lifecycle "booking" {
  aggregate = aggregate.booking.booking

  state "done" {
    end  = true
    path = "main"
  }

  transition "close" {
    command = command.close.close
    events  = [event.booking.closed]
    before  = [state.open, state.held]
    after   = state.done
  }

  invalid_transition "reopen" {
    command = command.reopen.reopen
    reason  = "A closed booking stays closed."
    before  = [state.done]
  }
}

evidence "rule" {
  certainty = "stated"
  location  = "page 3"
  source    = evidence_source.policy
  on        = state.booking.done
}

decision "debt" {
  kind      = "accepted_debt"
  signal    = "EM403"
  statement = "The list reads two events."
  reason    = "One list."
  on        = workflow.close
}

hotspot "refund" {
  kind     = "request"
  category = "rule"
  question = "Refund?"
  options  = ["B", "A"]
  concerns = "scenario"
  on       = workflow.close
}

state_view "list" {
  field_source {
    description = "Copy."
    inputs      = [event.booking.closed.field.name]
    target      = readmodel.list.field.name
  }

  scenario "empty" {
    case          = "appear"
    empty_history = true

    then {
      expect_empty_list  = true
      expect_no_dispatch = true
      readmodel          = readmodel.list
    }
  }
}

bounded_context "booking" {
  aggregate "booking" {
    title    = "Booking"
    identity = field_type.booking_id
  }

  event "closed" {
    field "name" {
      type              = "String"
      generated         = true
      id_attribute      = true
      correlation_key   = true
      deduplication_key = true
      pii               = true
    }
  }
}
`
	formatted, diagnostics := Format("model.em.hcl", source)
	if diagnostics.HasErrors() {
		t.Fatalf("diagnostics = %s", diagnostics.Error())
	}
	if string(formatted) != want {
		t.Fatalf("formatted =\n%s\nwant\n%s", formatted, want)
	}
}
