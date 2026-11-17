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
	covered := []string{"Actors", "Owners", "Contexts", "Board", "OffBoard", "Workflows", "Chapters", "Storylines", "Hotspots", "Lifecycles", "EvidenceSources", "Evidence", "Decisions", "Edges"}
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

func TestModels_ComparesStorylinesByIDAndChapterOrder(t *testing.T) {
	base := &model.Model{Storylines: []model.Storyline{
		{ID: "care", Chapters: []string{"chapter.treat"}},
		{ID: "intake", Chapters: []string{"chapter.register", "chapter.check_in"}},
	}}
	head := &model.Model{Storylines: []model.Storyline{
		{ID: "billing", Chapters: []string{"chapter.pay"}},
		{ID: "intake", Chapters: []string{"chapter.check_in", "chapter.register"}},
	}}

	got := Models(base, head)

	want := []Change{
		{Change: Added, Kind: "storyline", ID: "billing"},
		{Change: Removed, Kind: "storyline", ID: "care"},
		{Change: Changed, Kind: "storyline", ID: "intake"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Models = %#v, want %#v", got, want)
	}
	if label := got[0].Label(); label != "storyline billing" {
		t.Fatalf("Label = %q, want %q", label, "storyline billing")
	}
}
