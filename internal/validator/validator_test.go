package validator

import (
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
)

func TestValidateSource_SliceIndexAcceptsArbitraryPrecisionInteger(t *testing.T) {
	// Given a slice whose index exceeds machine-sized integer precision.
	model := `slice "example" {
  title      = "Example"
  index      = 999999999999999999999999999999999999999999999999999999999999
  slice_type = "STATE_CHANGE"
}
`

	// When the model is validated.
	diagnostics := validateModel(t, model)

	// Then it is valid.
	requireValid(t, diagnostics)
}

func TestValidateSource_FieldPIIAcceptsBoolean(t *testing.T) {
	// Given a field marked as personally identifiable information.
	model := `slice "example" {
  title      = "Example"
  slice_type = "STATE_CHANGE"
  event "event" {
    title = "Event"
    type  = "EVENT"
    field "name" {
      type = "String"
      pii  = true
    }
  }
}
`

	// When the model is validated.
	diagnostics := validateModel(t, model)

	// Then it is valid.
	requireValid(t, diagnostics)
}

func TestValidateSource_FieldPIIRejectsNonBoolean(t *testing.T) {
	// Given a field whose pii metadata is not a boolean.
	model := `slice "example" {
  title      = "Example"
  slice_type = "STATE_CHANGE"
  event "event" {
    title = "Event"
    type  = "EVENT"
    field "name" {
      type = "String"
      pii  = "personal"
    }
  }
}
`

	// When the model is validated.
	diagnostics := validateModel(t, model)

	// Then its pii metadata is rejected.
	requireDiagnostic(t, diagnostics, "pii must be a boolean.")
}

func TestValidateSource_FieldExampleAcceptsNativeLiterals(t *testing.T) {
	// Given a field with list, boolean, number, null, and object literals.
	model := `slice "example" {
  title      = "Example"
  slice_type = "STATE_CHANGE"
  event "event" {
    title = "Event"
    type  = "EVENT"
    field "examples" {
      type    = "Custom"
      example = [true, 12, null, { nested = "value" }]
    }
  }
}
`

	// When the model is validated.
	diagnostics := validateModel(t, model)

	// Then all literal forms are accepted.
	requireValid(t, diagnostics)
}

func TestValidateSource_ReturnsAllIndependentDiagnostics(t *testing.T) {
	// Given a model with independent violations at multiple nesting levels.
	model := `slice "example" {
  title      = 1
  slice_type = "INVALID"
  event "event" {
    type = "COMMAND"
  }
}
`

	// When the model is validated.
	diagnostics := validateModel(t, model)

	// Then each independent violation is reported.
	for _, want := range []string{
		"title must be a string.",
		"slice_type must be one of: STATE_CHANGE, STATE_VIEW, AUTOMATION.",
		"Missing required argument",
		`type must be "EVENT" for an event block.`,
	} {
		requireDiagnostic(t, diagnostics, want)
	}
}

func TestValidateSource_ReturnsDiagnosticsInSchemaOrder(t *testing.T) {
	// Given a slice with invalid attributes declared out of schema order.
	model := `slice "example" {
  slice_type = "INVALID"
  status     = "Pending"
  title      = 1
}
`

	// When the model is validated.
	diagnostics := validateModel(t, model)

	// Then diagnostics follow the schema's stable attribute order.
	want := []string{
		"title must be a string.",
		"status must be one of: Created, Done, InProgress.",
		"slice_type must be one of: STATE_CHANGE, STATE_VIEW, AUTOMATION.",
	}
	if got := diagnosticMessages(diagnostics); !slices.Equal(got, want) {
		t.Fatalf("diagnostic messages = %#v, want %#v", got, want)
	}
}

func TestValidateSource_AcceptsCoreSliceChild(t *testing.T) {
	tests := []struct {
		name  string
		child string
	}{
		{"command", `command "command" {
  title = "Command"
  type  = "COMMAND"
}`},
		{"event", `event "event" {
  title = "Event"
  type  = "EVENT"
}`},
		{"readmodel", `readmodel "readmodel" {
  title = "Read model"
  type  = "READMODEL"
}`},
		{"screen", `screen "screen" {
  title = "Screen"
  type  = "SCREEN"
}`},
		{"processor", `processor "processor" {
  title = "Processor"
  type  = "AUTOMATION"
}`},
		{"screen image", `screen_image "image" {
  title = "Image"
}`},
		{"table", `table "table" {
  title = "Table"
}`},
		{"specification", `specification "specification" {
  title     = "Specification"
  linked_id = "example"
}`},
		{"actor", `actor "User" {
  auth_required = false
}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given one permitted direct child of a valid slice.
			model := `slice "example" {
  title      = "Example"
  slice_type = "STATE_CHANGE"
  ` + test.child + `
}
`

			// When the model is validated.
			diagnostics := validateModel(t, model)

			// Then that child is accepted.
			requireValid(t, diagnostics)
		})
	}
}

func TestValidateSource_RejectsStructuralViolation(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  string
	}{
		{"top level block", `event "event" {}`, "Unsupported block type"},
		{"unknown slice attribute", `slice "example" {
  title      = "Example"
  slice_type = "STATE_CHANGE"
  unknown    = true
}`, "Unsupported argument"},
		{"unknown slice block", `slice "example" {
  title      = "Example"
  slice_type = "STATE_CHANGE"
  unknown "value" {}
}`, "Unsupported block type"},
		{"missing element title", `slice "example" {
  title      = "Example"
  slice_type = "STATE_CHANGE"
  command "command" {
    type = "COMMAND"
  }
}`, "Missing required argument"},
		{"mismatched element type", `slice "example" {
  title      = "Example"
  slice_type = "STATE_CHANGE"
  event "event" {
    title = "Event"
    type  = "COMMAND"
  }
}`, `type must be "EVENT"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given a model with one structural violation.

			// When the model is validated.
			diagnostics := validateModel(t, test.model)

			// Then that violation is reported.
			requireDiagnostic(t, diagnostics, test.want)
		})
	}
}

func TestValidateSource_RejectsNonLiteralExpression(t *testing.T) {
	// Given a model containing a variable expression.
	model := `slice "example" {
  title      = var.title
  slice_type = "STATE_CHANGE"
}
`

	// When the model is validated.
	diagnostics := validateModel(t, model)

	// Then the expression is rejected.
	requireDiagnostic(t, diagnostics, "Variables not allowed")
}

func TestValidateSource_RejectsDeferredV1Construct(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  string
	}{
		{"event group block", `event_group "pets" {}`, "Unsupported block type"},
		{"external event block", `external_event "evt-pet" {}`, "Unsupported block type"},
		{"emits attribute", `slice "example" {
  title      = "Example"
  slice_type = "STATE_CHANGE"
  emits      = ["evt-pet"]
}
`, "Unsupported argument"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given one HCL construct deferred from v1.

			// When the model is validated.
			diagnostics := validateModel(t, test.model)

			// Then the construct is rejected.
			requireDiagnostic(t, diagnostics, test.want)
		})
	}
}

func TestValidateSource_RejectsNestedSchemaViolation(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  string
	}{
		{"invalid slice status", `slice "example" {
  title      = "Example"
  status     = "Pending"
  slice_type = "STATE_CHANGE"
}`, "Invalid enum value"},
		{"invalid element context", `slice "example" {
  title      = "Example"
  slice_type = "STATE_CHANGE"
  command "command" {
    title   = "Command"
    type    = "COMMAND"
    context = "PUBLIC"
  }
}`, "Invalid enum value"},
		{"unknown field attribute", `slice "example" {
  title      = "Example"
  slice_type = "STATE_CHANGE"
  command "command" {
    title = "Command"
    type  = "COMMAND"
    field "name" {
      type    = "String"
      unknown = true
    }
  }
}`, "Unsupported argument"},
		{"invalid field type", `slice "example" {
  title      = "Example"
  slice_type = "STATE_CHANGE"
  command "command" {
    title = "Command"
    type  = "COMMAND"
    field "name" {
      type = "Text"
    }
  }
}`, "Invalid enum value"},
		{"missing dependency element type", `slice "example" {
  title      = "Example"
  slice_type = "STATE_CHANGE"
  command "command" {
    title = "Command"
    type  = "COMMAND"
    dependency "event" {
      type  = "OUTBOUND"
      title = "Event"
    }
  }
}`, "Missing required argument"},
		{"invalid specification step type", `slice "example" {
  title      = "Example"
  slice_type = "STATE_CHANGE"
  specification "specification" {
    title     = "Specification"
    linked_id = "example"
    then "result" {
      title = "Result"
      type  = "SPEC_SCREEN"
    }
  }
}`, "Invalid enum value"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given a model with one nested-schema violation.

			// When the model is validated.
			diagnostics := validateModel(t, test.model)

			// Then that violation is reported.
			requireDiagnostic(t, diagnostics, test.want)
		})
	}
}

func validateModel(t *testing.T, model string) hcl.Diagnostics {
	t.Helper()
	return ValidateSource("model.em.hcl", []byte(model))
}

func requireValid(t *testing.T, diagnostics hcl.Diagnostics) {
	t.Helper()
	if diagnostics.HasErrors() {
		t.Fatalf("diagnostics = %s", diagnostics.Error())
	}
}

func requireDiagnostic(t *testing.T, diagnostics hcl.Diagnostics, want string) {
	t.Helper()
	if !diagnostics.HasErrors() {
		t.Fatalf("diagnostics = %#v, want error containing %q", diagnostics, want)
	}
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Summary, want) || strings.Contains(diagnostic.Detail, want) {
			return
		}
	}
	t.Fatalf("diagnostics = %#v, want error containing %q", diagnostics, want)
}

func diagnosticMessages(diagnostics hcl.Diagnostics) []string {
	messages := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		messages = append(messages, diagnostic.Detail)
	}
	return messages
}
