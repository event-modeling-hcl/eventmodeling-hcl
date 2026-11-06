package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
)

func methodFactsSource(t *testing.T) string {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "method-facts.em.hcl"))
	if err != nil {
		t.Fatalf("read example: %v", err)
	}
	return string(source)
}

func scenarioByID(t *testing.T, workflow Workflow, id string) Scenario {
	t.Helper()
	for _, scenario := range workflow.Scenarios {
		if scenario.ID == id {
			return scenario
		}
	}
	t.Fatalf("workflow %s has no scenario %s", workflow.ID, id)
	return Scenario{}
}

func TestBuild_HoldsScenarioFacts(t *testing.T) {
	built := buildFiles(t, []syntax.File{{Name: "model.em.hcl", Source: []byte(methodFactsSource(t))}})

	if got := scenarioByID(t, workflowByID(t, built, "open_booking"), "booking_is_opened").Case; got != "happy_path" {
		t.Fatalf("case = %q, want happy_path", got)
	}
	empty := scenarioByID(t, workflowByID(t, built, "open_bookings"), "no_booking_yet")
	if !empty.EmptyHistory || empty.Case != "" {
		t.Fatalf("empty-history scenario = %+v, want empty_history and no case", empty)
	}
	deposit := workflowByID(t, built, "record_deposit")
	issued := scenarioByID(t, deposit, "payment_becomes_deposit").Steps
	if last := issued[len(issued)-1]; last.Kind != Then || last.Target != "command" || last.Ref != "command.record_deposit" || last.ExpectNoDispatch {
		t.Fatalf("dispatch step = %+v, want a then command step that issues command.record_deposit", last)
	}
	ignored := scenarioByID(t, deposit, "repeated_payment_is_ignored").Steps
	if last := ignored[len(ignored)-1]; last.Target != "command" || !last.ExpectNoDispatch {
		t.Fatalf("no-dispatch step = %+v, want expect_no_dispatch", last)
	}
	for _, edge := range built.Edges {
		if edge.From == "processor.deposit_translator" && edge.To != "command.record_deposit" {
			t.Fatalf("edge %+v: a dispatch assertion must not add a flow edge", edge)
		}
	}
}

func TestBuild_HoldsStreamFacts(t *testing.T) {
	built := buildFiles(t, []syntax.File{{Name: "model.em.hcl", Source: []byte(methodFactsSource(t))}})

	if got := built.Contexts[0].Aggregates[0].Identity; got != "field_type.booking.booking_id" {
		t.Fatalf("identity = %q, want the qualified field type", got)
	}
	if len(built.Lifecycles) != 1 {
		t.Fatalf("lifecycles = %+v, want one", built.Lifecycles)
	}
	lifecycle := built.Lifecycles[0]
	var states, transitions []string
	for _, state := range lifecycle.States {
		states = append(states, state.ID)
	}
	for _, transition := range lifecycle.Transitions {
		transitions = append(transitions, transition.ID)
	}
	requireOrder(t, "states", states, "canceled", "confirmed", "open")
	requireOrder(t, "transitions", transitions, "cancel_booking", "confirm_booking", "open_booking")
	start := lifecycle.Transitions[2]
	if start.Before == nil || len(start.Before) != 0 || start.After != "state.open" || start.Command != "command.open_booking.open_booking" {
		t.Fatalf("starting transition = %+v, want an empty before list", start)
	}
	encoded, err := json.Marshal(start)
	if err != nil || !strings.Contains(string(encoded), `"before":[]`) {
		t.Fatalf("starting transition JSON = %s, want before:[] (%v)", encoded, err)
	}
	if invalid := lifecycle.InvalidTransitions[0]; invalid.Reason != "A canceled booking cannot be confirmed." || !reflect.DeepEqual(invalid.Before, []string{"state.canceled"}) {
		t.Fatalf("invalid transition = %+v", invalid)
	}
}

func TestBuild_HoldsRecordsWithEffectiveKinds(t *testing.T) {
	source := strings.Replace(methodFactsSource(t), "  kind      = \"accepted_debt\"\n  signal    = \"EM403\"\n", "", 1)
	source = strings.Replace(source, "  kind     = \"assumption\"\n", "", 1)
	built := buildFiles(t, []syntax.File{{Name: "model.em.hcl", Source: []byte(source)}})

	if got := built.Decisions[0].DecisionKind; got != "choice" {
		t.Fatalf("decision_kind = %q, want the effective choice", got)
	}
	kinds := map[string]string{}
	for _, hotspot := range built.Hotspots {
		kinds[hotspot.ID] = hotspot.HotspotKind
	}
	if !reflect.DeepEqual(kinds, map[string]string{"late_cancellation_scenario": "request", "refund_rule": "question"}) {
		t.Fatalf("hotspot kinds = %v, want request and the effective question", kinds)
	}
	if got := built.EvidenceSources[0].SourceKind; got != "document" {
		t.Fatalf("source_kind = %q, want document", got)
	}
	if got := built.Evidence[1].On; got != "event.payments.payment_received.field.payment_reference" {
		t.Fatalf("evidence on = %q, want the field reference as written", got)
	}
}

func TestBuild_HoldsFieldSourcesWithResolvedPaths(t *testing.T) {
	built := buildFiles(t, []syntax.File{{Name: "model.em.hcl", Source: []byte(methodFactsSource(t))}})

	sources := workflowByID(t, built, "open_booking").FieldSources
	if len(sources) != 2 {
		t.Fatalf("field sources = %+v, want two", sources)
	}
	first := sources[0]
	want := FieldSource{
		Inputs: []FieldReference{{Reference: "screen.booking_form.field.name", Element: "screen.open_booking.booking_form", Path: []string{"name"}}},
		Target: FieldReference{Reference: "command.open_booking.field.guest_name", Element: "command.open_booking.open_booking", Path: []string{"guest_name"}},
	}
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("first field source = %+v, want %+v (sorted by target text)", first, want)
	}
	if got := sources[1].Target.Element; got != "event.booking.booking_opened" {
		t.Fatalf("second target element = %q, want the event", got)
	}
	var flags []string
	for _, field := range built.Contexts[1].Events[0].Fields {
		if field.CorrelationKey {
			flags = append(flags, "correlation:"+field.Name)
		}
		if field.DeduplicationKey {
			flags = append(flags, "deduplication:"+field.Name)
		}
	}
	requireOrder(t, "key flags", flags, "correlation:booking_reference", "deduplication:payment_reference")
}

// Moving every record into another member file keeps the IR.
func TestBuild_RecordsAreIndependentOfFileLayout(t *testing.T) {
	source := methodFactsSource(t)
	split := strings.Index(source, `lifecycle "booking"`)
	records := strings.Index(source, `evidence_source "booking_policy"`)
	workflowsEnd := strings.Index(source, `state_change "open_booking"`)
	if split < 0 || records < 0 || workflowsEnd < 0 {
		t.Fatal("example layout changed")
	}
	oneFile := buildFiles(t, []syntax.File{{Name: "model.em.hcl", Source: []byte(source)}})
	moved := buildFiles(t, []syntax.File{
		{Name: "model/b.em.hcl", Source: []byte(source[:split] + source[workflowsEnd:records])},
		{Name: "model/a.em.hcl", Source: []byte(source[records:] + "\n" + source[split:workflowsEnd])},
	})
	first, _ := json.Marshal(oneFile)
	second, _ := json.Marshal(moved)
	if string(first) != string(second) {
		t.Fatalf("IR changed when records moved to another file\none:   %s\nsplit: %s", first, second)
	}
}
