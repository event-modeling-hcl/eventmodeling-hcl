package renderer

import (
	"reflect"
	"testing"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/model"
)

func TestBuildContextMap_DerivesCustomerSupplierEdge(t *testing.T) {
	view := BuildViewModel("context-map.em.hcl", contextMapEdgeModel(model.StateView, false))

	if got, want := view.ContextMap.Edges, []ContextEdge{{
		Upstream: "catalogue", Downstream: "lending", Pattern: "customer_supplier", Via: []string{"project_book"},
	}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("context map edges = %#v, want %#v", got, want)
	}
}

func TestBuildContextMap_DerivesAnticorruptionEdge(t *testing.T) {
	// Anti-corruption requires a translation: an automation with an external edge.
	view := BuildViewModel("context-map.em.hcl", contextMapEdgeModel(model.Automation, true))

	if got, want := view.ContextMap.Edges[0].Pattern, "anticorruption"; got != want {
		t.Fatalf("edge pattern = %q, want %q", got, want)
	}
	if !findContextNode(t, view.ContextMap, "catalogue").External {
		t.Fatal("upstream context external = false, want true")
	}
}

func TestBuildContextMap_AutomationWithoutExternalEdgeIsCustomerSupplier(t *testing.T) {
	view := BuildViewModel("context-map.em.hcl", contextMapEdgeModel(model.Automation, false))

	if got, want := view.ContextMap.Edges[0].Pattern, "customer_supplier"; got != want {
		t.Fatalf("edge pattern = %q, want %q", got, want)
	}
	if got, want := view.Slices[0].Type, "automation"; got != want {
		t.Fatalf("slice type = %q, want %q", got, want)
	}
}

func TestBuildViewModel_AutomationWithExternalEdgeRendersAsTranslation(t *testing.T) {
	view := BuildViewModel("context-map.em.hcl", contextMapEdgeModel(model.Automation, true))

	slice := findSlice(t, view, "project_book")
	if got, want := slice.Type, "translation"; got != want {
		t.Fatalf("slice type = %q, want %q", got, want)
	}
	event := findElement(t, slice, "event__catalogue__book_added")
	readModel := findElement(t, slice, "project_book__readmodel__book_availability")
	if !event.External || !event.Given || event.Stage >= readModel.Stage {
		t.Fatalf("external input = %+v, read model stage %d; want an external upstream event before the read model", event, readModel.Stage)
	}
}

func TestBuildContextMap_NodeFieldsReflectContextAndOwner(t *testing.T) {
	view := BuildViewModel("context-map.em.hcl", &model.Model{
		Owners: []model.Owner{{Kind: "team", ID: "library_team", Title: "Library Team"}},
		Contexts: []model.Context{
			{
				ID: "catalogue", Title: "Catalogue", Owner: "team.library_team",
				Aggregates: []model.Aggregate{{ID: "book"}},
				Events:     []model.Event{{ID: "book_added"}, {ID: "book_removed"}},
			},
			{ID: "partner", Title: "Partner", External: true},
		},
	})

	catalogue := findContextNode(t, view.ContextMap, "catalogue")
	if got, want := catalogue.Team, "Library Team"; got != want {
		t.Fatalf("catalogue team = %q, want %q", got, want)
	}
	if got, want := catalogue.Events, 2; got != want {
		t.Fatalf("catalogue events = %d, want %d", got, want)
	}
	if got, want := catalogue.Aggregates, 1; got != want {
		t.Fatalf("catalogue aggregates = %d, want %d", got, want)
	}
	if !findContextNode(t, view.ContextMap, "partner").External {
		t.Fatal("partner external = false, want true")
	}
}

func TestBuildContextMap_IsDeterministic(t *testing.T) {
	source := contextMapEdgeModel(model.StateView, false)
	first := BuildViewModel("context-map.em.hcl", source).ContextMap
	second := BuildViewModel("context-map.em.hcl", source).ContextMap

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("context maps differ: first %#v, second %#v", first, second)
	}
}

func TestBuildContextMap_UsesEmptyEdgesSlice(t *testing.T) {
	view := BuildViewModel("context-map.em.hcl", &model.Model{
		Contexts: []model.Context{{ID: "catalogue", Title: "Catalogue"}},
	})

	if view.ContextMap.Edges == nil {
		t.Fatal("context map edges = nil, want empty slice")
	}
	if got := len(view.ContextMap.Edges); got != 0 {
		t.Fatalf("context map edges = %d, want 0", got)
	}
}

// contextMapEdgeModel is a workflow in lending that consumes a catalogue event.
// The catalogue context is external when upstreamExternal is set, and then the
// consuming edge is external too, as the IR records it.
func contextMapEdgeModel(kind model.WorkflowKind, upstreamExternal bool) *model.Model {
	return &model.Model{
		Contexts: []model.Context{
			{ID: "catalogue", Title: "Catalogue", External: upstreamExternal, Events: []model.Event{{ID: "book_added", Title: "Book added"}}},
			{ID: "lending", Title: "Lending", Aggregates: []model.Aggregate{{ID: "lending_book", Title: "Lending book"}}},
		},
		Board: []string{"project_book"},
		Workflows: []model.Workflow{{
			Kind: kind, ID: "project_book", Title: "Project book",
			Elements: []model.Element{{
				Kind: model.ReadModel, ID: "book_availability", Title: "Book availability",
				Semantic: model.Semantic{Aggregate: "aggregate.lending.lending_book"},
				From:     []string{"event.catalogue.book_added"},
			}},
			Scenarios: []model.Scenario{{Steps: []model.Step{
				{Kind: model.Given, Target: "event", Ref: "event.catalogue.book_added"},
			}}},
		}},
		Edges: []model.Edge{{
			WorkflowID: "project_book", From: "event.catalogue.book_added", To: "readmodel.book_availability",
			External: upstreamExternal,
		}},
	}
}

func findContextNode(t *testing.T, contextMap ContextMap, id string) ContextNode {
	t.Helper()
	for _, node := range contextMap.Nodes {
		if node.ID == id {
			return node
		}
	}
	t.Fatalf("context node %q not found", id)
	return ContextNode{}
}
