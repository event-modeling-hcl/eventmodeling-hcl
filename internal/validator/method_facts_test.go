package validator

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
)

// methodFactsExample is examples/method-facts.em.hcl, the complete example of
// spec RFC 0007. Each case below edits it and states the core error codes
// that the edit must give. Judgment signals are not core errors.
func methodFactsExample(t *testing.T) string {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "method-facts.em.hcl"))
	if err != nil {
		t.Fatalf("read example: %v", err)
	}
	return string(source)
}

// edit is one change to the example: replace the first occurrence of old with
// new. An empty old appends new to the end of the model.
type edit struct{ old, new string }

func applyEdits(t *testing.T, source string, edits []edit) string {
	t.Helper()
	for _, change := range edits {
		if change.old == "" {
			source += "\n" + change.new + "\n"
			continue
		}
		if !strings.Contains(source, change.old) {
			t.Fatalf("example does not contain %q", change.old)
		}
		source = strings.Replace(source, change.old, change.new, 1)
	}
	return source
}

func coreErrorCodes(diagnostics hcl.Diagnostics) []string {
	var codes []string
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == hcl.DiagError && !IsJudgment(diagnostic) {
			if code := DiagnosticCode(diagnostic); !slices.Contains(codes, code) {
				codes = append(codes, code)
			}
		}
	}
	slices.Sort(codes)
	return codes
}

// todoAutomation is an automation with a to-do list, for the view form cases.
const todoAutomation = `automation "review_deposits" {
  readmodel "deposits_to_review" {
    question = "Which deposits wait for a review?"
    from     = [event.booking.deposit_recorded]
  }

  scenario "review_list" {
%s
  }
}`

func todoScenario(body string) string {
	return strings.Replace(todoAutomation, "%s", body, 1)
}

func TestMethodFacts_AcceptanceTable(t *testing.T) {
	tests := []struct {
		name  string
		edits []edit
		want  []string
	}{
		{name: "complete example is valid"},

		// Part A: scenario facts.
		{name: "read model case in a state_change scenario", edits: []edit{{`case = "happy_path"

    when {
      command = command.open_booking`, `case = "appear"

    when {
      command = command.open_booking`}}, want: []string{"EM304"}},
		{name: "command case in an automation view form scenario", edits: []edit{{"", todoScenario(`    case = "state_violation"
    given { event = event.booking.deposit_recorded }
    then { readmodel = readmodel.deposits_to_review }`)}}, want: []string{"EM304"}},
		{name: "unknown case", edits: []edit{{`case = "happy_path"`, `case = "happy"`}}, want: []string{"EM008"}},
		{name: "case that is not a string", edits: []edit{{`case = "happy_path"`, `case = 1`}}, want: []string{"EM007"}},
		{name: "case is not compared with the steps", edits: []edit{{`case = "happy_path"`, `case = "refused_input"`}}},
		{name: "view scenario without given and without empty_history", edits: []edit{{"empty_history = true\n", ""}}, want: []string{"EM301"}},
		{name: "empty_history with a given step", edits: []edit{{"empty_history = true\n", "empty_history = true\n    given {\n      event = event.booking.booking_opened\n    }\n"}}, want: []string{"EM301"}},
		{name: "automation view form with empty_history", edits: []edit{{"", todoScenario(`    empty_history = true
    then {
      expect_empty_list = true
      readmodel         = readmodel.deposits_to_review
    }`)}}},
		{name: "empty_history on a state_change scenario", edits: []edit{{`case = "happy_path"`, "case          = \"happy_path\"\n    empty_history = false"}}, want: []string{"EM304"}},
		{name: "empty_history on an action form scenario", edits: []edit{{`case = "business_duplicate"`, "case          = \"business_duplicate\"\n    empty_history = true"}}, want: []string{"EM304"}},
		{name: "then command after a when command", edits: []edit{{"", `automation "redo_deposit" {
  command "redo" {
    external_trigger = true
  }

  scenario "redo_twice" {
    when { command = command.redo }
    then { command = command.redo }
  }
}`}}, want: []string{"EM303"}},
		{name: "then command in a state_change scenario", edits: []edit{{`then {
      event = event.booking.booking_opened
    }`, `then {
      command = command.open_booking
    }`}}, want: []string{"EM303"}},
		{name: "then command of another workflow", edits: []edit{{`then {
      command = command.record_deposit
    }`, `then {
      command = command.open_booking.open_booking
    }`}}, want: []string{"EM101"}},
		{name: "then command of the own workflow in the qualified form", edits: []edit{{`then {
      command = command.record_deposit
    }`, `then {
      command = command.record_deposit.record_deposit
    }`}}},
		{name: "expect_no_dispatch on a then event step", edits: []edit{{`then {
      event = event.booking.booking_opened
    }`, `then {
      expect_no_dispatch = true
      event              = event.booking.booking_opened
    }`}}, want: []string{"EM304"}},
		{name: "expect_no_dispatch with a field block", edits: []edit{{`expect_no_dispatch = true
      command            = command.record_deposit`, `expect_no_dispatch = true
      command            = command.record_deposit

      field "amount" {
        type = field_type.booking.amount
      }`}}, want: []string{"EM304"}},
		{name: "issued and not issued in one scenario", edits: []edit{{`expect_no_dispatch = true
      command            = command.record_deposit
    }`, `expect_no_dispatch = true
      command            = command.record_deposit
    }

    then {
      command = command.record_deposit.record_deposit
    }`}}, want: []string{"EM305"}},
		{name: "dispatch and no dispatch of a command without an edge", edits: []edit{
			{`  command "record_deposit" {`, "  command \"flag_deposit\" {\n    external_trigger = true\n  }\n\n  command \"record_deposit\" {"},
			{`then {
      command = command.record_deposit
    }`, `then {
      command = command.record_deposit
    }

    then {
      command = command.flag_deposit
    }`},
			{`expect_no_dispatch = true
      command            = command.record_deposit`, `expect_no_dispatch = true
      command            = command.flag_deposit`},
		}},

		// Part B: stream facts.
		{name: "event of the aggregate without the identity field", edits: []edit{{`event "booking_confirmed" {
    aggregate = aggregate.booking
    fields    = [field_type.booking_id]
  }`, `event "booking_confirmed" {
    aggregate = aggregate.booking
  }`}}, want: []string{"EM018"}},
		{name: "identity field with the same name and a built-in type", edits: []edit{{`event "booking_confirmed" {
    aggregate = aggregate.booking
    fields    = [field_type.booking_id]
  }`, `event "booking_confirmed" {
    aggregate = aggregate.booking

    field "booking_id" {
      type = "UUID"
    }
  }`}}, want: []string{"EM018"}},
		{name: "identity that is a string", edits: []edit{{"identity = field_type.booking_id", `identity = "UUID"`}}, want: []string{"EM101"}},
		{name: "no identity and events without an identity field", edits: []edit{
			{"    identity = field_type.booking_id\n", ""},
			{`event "booking_confirmed" {
    aggregate = aggregate.booking
    fields    = [field_type.booking_id]
  }`, `event "booking_confirmed" {
    aggregate = aggregate.booking
  }`},
		}},
		{name: "two lifecycles for one aggregate", edits: []edit{{"", `lifecycle "booking_again" {
  aggregate = aggregate.booking.booking

  state "open" {}
}`}}, want: []string{"EM017"}},
		{name: "lifecycle without a state", edits: []edit{
			{"  aggregate \"booking\" {", "  aggregate \"guest\" {}\n\n  aggregate \"booking\" {"},
			{"", "lifecycle \"guest\" {\n  aggregate = aggregate.booking.guest\n}"},
		}, want: []string{"EM017"}},
		{name: "transition with an empty events list", edits: []edit{{"events  = [event.booking.booking_confirmed]", "events  = []"}}, want: []string{"EM017"}},
		{name: "invalid transition with an empty before list", edits: []edit{{"before  = [state.canceled]", "before  = []"}}, want: []string{"EM017"}},
		{name: "invalid transition with a blank reason", edits: []edit{{`reason  = "A canceled booking cannot be confirmed."`, `reason  = " "`}}, want: []string{"EM007"}},
		{name: "state and command both valid and invalid", edits: []edit{{"before  = [state.canceled]", "before  = [state.open]"}}, want: []string{"EM017"}},
		{name: "transition event of another aggregate", edits: []edit{
			{"", "bounded_context \"other\" {\n  aggregate \"thing\" {}\n\n  event \"thing_happened\" {\n    aggregate = aggregate.thing\n  }\n}"},
			{"events  = [event.booking.booking_confirmed]", "events  = [event.booking.booking_confirmed, event.other.thing_happened]"},
		}, want: []string{"EM017"}},
		{name: "state reference inside a workflow", edits: []edit{{"to    = [command.confirm_booking]", "to    = [command.confirm_booking, state.open]"}}, want: []string{"EM101"}},
		{name: "unknown state inside a lifecycle", edits: []edit{{"after   = state.confirmed", "after   = state.missing"}}, want: []string{"EM102"}},
		{name: "lifecycle aggregate in the local form", edits: []edit{{"aggregate = aggregate.booking.booking\n\n  state", "aggregate = aggregate.booking\n\n  state"}}, want: []string{"EM101"}},
		{name: "transition whose command does not flow to its events", edits: []edit{{"events  = [event.booking.booking_confirmed]", "events  = [event.booking.booking_canceled]"}}},
		{name: "creates_aggregate is not compared with the lifecycle", edits: []edit{{`  command "open_booking" {
    fields`, `  command "open_booking" {
    creates_aggregate = true
    fields`}}},
		{name: "rename a state", edits: []edit{{`state "open" {`, `state "opened" {`}}, want: []string{"EM102"}},

		// Part C: workshop records.
		{name: "unknown certainty", edits: []edit{{`certainty = "stated"`, `certainty = "likely"`}}, want: []string{"EM008"}},
		{name: "stated evidence without a source", edits: []edit{{"  source    = evidence_source.booking_policy\n", ""}}, want: []string{"EM000"}},
		{name: "evidence on another evidence", edits: []edit{{"on          = event.payments.payment_received.field.payment_reference", "on          = evidence.canceled_booking_rule"}}, want: []string{"EM101"}},
		{name: "evidence on a decision and a hotspot", edits: []edit{
			{"on          = event.payments.payment_received.field.payment_reference", "on          = decision.one_open_bookings_list"},
			{"on        = invalid_transition.booking.confirm_canceled_booking", "on        = hotspot.refund_rule"},
		}},
		{name: "evidence source of the wrong kind", edits: []edit{{"source    = evidence_source.booking_policy", "source    = decision.one_open_bookings_list"}}, want: []string{"EM101"}},
		{name: "decision on a hotspot", edits: []edit{{"on        = readmodel.open_bookings.open_bookings", "on        = hotspot.refund_rule"}}, want: []string{"EM101"}},
		{name: "signal on a choice", edits: []edit{{`kind      = "accepted_debt"`, `kind      = "choice"`}}, want: []string{"EM000"}},
		{name: "signal on a decision without kind", edits: []edit{{"  kind      = \"accepted_debt\"\n", ""}}, want: []string{"EM000"}},
		{name: "signal that is not a judgment code", edits: []edit{{`signal    = "EM403"`, `signal    = "EM201"`}}, want: []string{"EM008"}},
		{name: "empty statement", edits: []edit{{`statement = "The open bookings list reads two events."`, `statement = ""`}}, want: []string{"EM007"}},
		{name: "decision on a workflow", edits: []edit{{"on        = readmodel.open_bookings.open_bookings", "on        = workflow.open_bookings"}}},
		{name: "decision on the whole model", edits: []edit{{"  on        = readmodel.open_bookings.open_bookings\n", ""}}},
		{name: "request without concerns", edits: []edit{{"  concerns = \"scenario\"\n", ""}}, want: []string{"EM000"}},
		{name: "concerns without kind", edits: []edit{{"  kind     = \"request\"\n", ""}}, want: []string{"EM000"}},
		{name: "unknown concerns", edits: []edit{{`concerns = "scenario"`, `concerns = "story"`}}, want: []string{"EM008"}},
		{name: "hotspot on a decision", edits: []edit{{"on       = workflow.cancel_booking", "on       = decision.one_open_bookings_list"}}},
		{name: "hotspot on a hotspot", edits: []edit{{"on       = workflow.cancel_booking", "on       = hotspot.refund_rule"}}, want: []string{"EM101"}},
		{name: "hotspot on a chapter", edits: []edit{{"on       = workflow.cancel_booking", "on       = chapter.booking"}}, want: []string{"EM101"}},
		{name: "hotspot on a lifecycle state", edits: []edit{{"on       = workflow.cancel_booking", "on       = state.booking.canceled"}}},
		{name: "hotspot on a state in the lifecycle form", edits: []edit{{"on       = workflow.cancel_booking", "on       = state.canceled"}}, want: []string{"EM101"}},
		{name: "rename a decision named by evidence", edits: []edit{
			{"on          = event.payments.payment_received.field.payment_reference", "on          = decision.one_open_bookings_list"},
			{`decision "one_open_bookings_list" {`, `decision "single_list" {`},
		}, want: []string{"EM102"}},
		{name: "one option", edits: []edit{{`options  = ["The guest loses the deposit.", "The guest gets the deposit back."]`, `options  = ["Keep it."]`}}},
		{name: "empty option", edits: []edit{{`options  = ["The guest loses the deposit.", "The guest gets the deposit back."]`, `options  = [""]`}}, want: []string{"EM007"}},

		// Part D: field facts.
		{name: "event field as the input of a command field", edits: []edit{{"inputs = [screen.booking_form.field.name]", "inputs = [event.booking.booking_opened.field.guest_name]"}}, want: []string{"EM202"}},
		{name: "input without a canonical edge to the target", edits: []edit{{"inputs = [event.booking.booking_opened.field.guest_name]", "inputs = [event.booking.booking_confirmed.field.booking_id]"}}, want: []string{"EM202"}},
		{name: "two inputs for a command field", edits: []edit{{"inputs = [screen.booking_form.field.name]", "inputs = [screen.booking_form.field.name, screen.booking_form.field.name]"}}, want: []string{"EM202"}},
		{name: "empty inputs", edits: []edit{{"inputs = [screen.booking_form.field.name]", "inputs = []"}}, want: []string{"EM202"}},
		{name: "two inputs for a read model field", edits: []edit{{"inputs = [event.booking.booking_opened.field.guest_name]", "inputs = [event.booking.booking_opened.field.guest_name, event.booking.booking_canceled.field.booking_id]"}}},
		{name: "two field sources with the same target", edits: []edit{{`  field_source {
    inputs = [screen.booking_form.field.name]
    target = command.open_booking.field.guest_name
  }`, `  field_source {
    inputs = [screen.booking_form.field.name]
    target = command.open_booking.field.guest_name
  }

  field_source {
    inputs = [screen.booking_form.field.name]
    target = command.open_booking.open_booking.field.guest_name
  }`}}, want: []string{"EM202"}},
		{name: "field name that the element does not declare", edits: []edit{{"target = command.open_booking.field.guest_name", "target = command.open_booking.field.guest"}}, want: []string{"EM102"}},
		{name: "field source of another workflow element", edits: []edit{{"target = command.open_booking.field.guest_name", "target = command.confirm_booking.confirm_booking.field.booking_id"}}, want: []string{"EM101"}},
		{name: "index in a field reference", edits: []edit{{"target = command.open_booking.field.guest_name", "target = command.open_booking.field.guest_name[0]"}}, want: []string{"EM101"}},
		{name: "field reference without the field segment", edits: []edit{{"target = command.open_booking.field.guest_name", "target = command.open_booking.guest_name"}}, want: []string{"EM101"}},
		{name: "converted list subfields at distinct nested paths", edits: nestedListEdits("unit_price", "unit_price")},
		{name: "missing subfield", edits: nestedListEdits("unit_price", "unit_cost"), want: []string{"EM102"}},
		{name: "two field sources for one nested target", edits: append(nestedListEdits("unit_price", "unit_price"), edit{`  scenario "booking_is_opened" {`, `  field_source {
    inputs = [screen.booking_form.field.cart.field.unit_price_cents]
    target = command.open_booking.field.cart.field.unit_price
  }

  scenario "booking_is_opened" {`}), want: []string{"EM202"}},
		{name: "evidence on a subfield", edits: append(nestedListEdits("unit_price", "unit_price"), edit{"on          = event.payments.payment_received.field.payment_reference", "on          = command.open_booking.open_booking.field.cart.field.unit_price"})},
		{name: "correlation_key that is not a Boolean", edits: []edit{{"correlation_key = true", `correlation_key = "yes"`}}, want: []string{"EM007"}},
		{name: "deduplication_key on an internal event field", edits: []edit{{`event "booking_canceled" {
    aggregate = aggregate.booking
    fields    = [field_type.booking_id]
  }`, `event "booking_canceled" {
    aggregate = aggregate.booking
    fields    = [field_type.booking_id]

    field "request_id" {
      type              = "UUID"
      deduplication_key = true
    }
  }`}}},
		{name: "key flag on a field_type", edits: []edit{{`field_type "guest_name" {
    type = "String"`, `field_type "guest_name" {
    type            = "String"
    correlation_key = true`}}, want: []string{"EM000"}},
	}
	base := methodFactsExample(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := applyEdits(t, base, test.edits)
			diagnostics := ValidateSourceWithProfile("method-facts.em.hcl", []byte(source), Valid)
			if got := coreErrorCodes(diagnostics); !slices.Equal(got, test.want) {
				t.Fatalf("core error codes = %v, want %v\n%s", got, test.want, diagnostics.Error())
			}
		})
	}
}

// nestedListEdits gives the screen and the command of open_booking a list
// field cart with subfields, and a field source that converts integer cents
// on the screen into a decimal amount on the command. A second source with
// the same final name under another parent stays a distinct target.
// commandSubfield names the command subfield that the source targets.
func nestedListEdits(declared, commandSubfield string) []edit {
	return []edit{
		{`    field "name" {
      type = field_type.booking.guest_name
    }`, `    field "name" {
      type = field_type.booking.guest_name
    }

    field "cart" {
      type        = "Custom"
      cardinality = "List"

      subfield "unit_price_cents" {
        type = "Int"
      }
    }`},
		{`    fields = [field_type.booking.booking_id, field_type.booking.guest_name]
    to     = [event.booking.booking_opened]`, `    fields = [field_type.booking.booking_id, field_type.booking.guest_name]
    to     = [event.booking.booking_opened]

    field "cart" {
      type        = "Custom"
      cardinality = "List"

      subfield "` + declared + `" {
        type = "Decimal"
      }
    }

    field "refund" {
      type = "Custom"

      subfield "unit_price" {
        type = "Decimal"
      }
    }`},
		{`  field_source {
    inputs = [command.open_booking.field.guest_name]`, `  field_source {
    description = "Convert the item price from integer cents to a decimal currency amount."
    inputs      = [screen.booking_form.field.cart.field.unit_price_cents]
    target      = command.open_booking.field.cart.field.` + commandSubfield + `
  }

  field_source {
    inputs = [screen.booking_form.field.cart.field.unit_price_cents]
    target = command.open_booking.field.refund.field.unit_price
  }

  field_source {
    inputs = [command.open_booking.field.guest_name]`},
	}
}

// A decision on a judgment signal never changes the signal or its severity.
func TestMethodFacts_AcceptedDebtKeepsItsSignal(t *testing.T) {
	base := []byte(methodFactsExample(t))
	without := []byte(applyEdits(t, string(base), []edit{{"  signal    = \"EM403\"\n", ""}, {`kind      = "accepted_debt"`, `kind      = "choice"`}}))
	for _, profile := range []Profile{Workshop, Valid, Strict} {
		withDebt := ValidateSourceWithProfile("model.em.hcl", base, profile)
		withChoice := ValidateSourceWithProfile("model.em.hcl", without, profile)
		if got, want := severities(withDebt, "EM403"), severities(withChoice, "EM403"); len(got) != 1 || !slices.Equal(got, want) {
			t.Fatalf("%s: EM403 severities with debt = %v, without = %v", profile, got, want)
		}
	}
}

// Hotspot kind, category, options, and concerns never change EM406. Status
// alone controls it.
func TestMethodFacts_HotspotStatusAloneControlsEM406(t *testing.T) {
	base := methodFactsExample(t)
	open := ValidateSourceWithProfile("model.em.hcl", []byte(base), Strict)
	if got := severities(open, "EM406"); !slices.Equal(got, []hcl.DiagnosticSeverity{hcl.DiagError, hcl.DiagError}) {
		t.Fatalf("EM406 severities = %v, want two errors for the assumption and the request", got)
	}
	resolved := applyEdits(t, base, []edit{{`  kind     = "request"`, "  kind     = \"request\"\n  status   = \"resolved\""}})
	if got := severities(ValidateSourceWithProfile("model.em.hcl", []byte(resolved), Strict), "EM406"); len(got) != 1 {
		t.Fatalf("EM406 count after resolving the request = %d, want 1", len(got))
	}
}

func severities(diagnostics hcl.Diagnostics, code string) []hcl.DiagnosticSeverity {
	var found []hcl.DiagnosticSeverity
	for _, diagnostic := range diagnostics {
		if DiagnosticCode(diagnostic) == code {
			found = append(found, diagnostic.Severity)
		}
	}
	return found
}
