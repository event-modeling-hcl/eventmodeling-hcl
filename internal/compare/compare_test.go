package compare

import (
	"reflect"
	"slices"
	"testing"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/model"
)

// Models must compare every top-level collection of the canonical model. A
// new collection that it does not read would make diff miss real changes.
func TestModels_CoversEveryTopLevelCollectionOfTheModel(t *testing.T) {
	// OffBoard is derived from Board and Workflows. Edges are part of the
	// value of their workflow.
	covered := []string{"Actors", "Owners", "Contexts", "Board", "OffBoard", "Workflows", "Chapters", "Hotspots", "Lifecycles", "EvidenceSources", "Evidence", "Decisions", "Edges"}
	fields := reflect.TypeFor[model.Model]()
	for index := range fields.NumField() {
		if name := fields.Field(index).Name; !slices.Contains(covered, name) {
			t.Errorf("model.Model.%s is not compared by Models", name)
		}
	}
}

func TestModels_EdgeChangeMarksItsWorkflowOnly(t *testing.T) {
	workflows := []model.Workflow{{Kind: model.StateChange, ID: "a"}, {Kind: model.StateChange, ID: "b"}}
	base := &model.Model{Workflows: workflows}
	head := &model.Model{Workflows: workflows, Edges: []model.Edge{{WorkflowID: "b", From: "command.b.x", To: "event.c.y"}}}

	got := Models(base, head)

	want := []Change{{Change: Changed, Kind: "workflow", ID: "b", Address: "workflow.b"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Models = %#v, want %#v", got, want)
	}
}
