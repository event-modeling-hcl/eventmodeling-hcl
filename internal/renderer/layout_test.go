package renderer

import (
	"fmt"
	"testing"
)

func TestBuildViewModel_AutomationFeedbackPreservesVisualOrder(t *testing.T) {
	for _, test := range []struct {
		name     string
		external bool
		offBoard bool
		wantType string
	}{
		{name: "automation", wantType: "automation"},
		{name: "translation", external: true, wantType: "translation"},
		{name: "off-board automation", offBoard: true, wantType: "automation"},
		{name: "off-board translation", external: true, offBoard: true, wantType: "translation"},
	} {
		t.Run(test.name, func(t *testing.T) {
			board := "[workflow.notify]"
			if test.offBoard {
				board = "[]"
			}
			source := fmt.Sprintf(automationFeedbackSource, board, test.external)
			loaded, diagnostics := loadTestModel(t, "automation-feedback.em.hcl", []byte(source))
			if diagnostics.HasErrors() {
				t.Fatalf("load feedback model: %s", diagnostics.Error())
			}

			view := BuildViewModel("Feedback", loaded)
			slices := view.Slices
			if test.offBoard {
				slices = view.OffBoard
			}
			if len(slices) != 1 {
				t.Fatalf("slice count = %d, want 1", len(slices))
			}
			slice := &slices[0]
			if slice.Type != test.wantType {
				t.Errorf("slice type = %q, want %q", slice.Type, test.wantType)
			}
			order := []string{
				"event__upstream__queued",
				"notify__readmodel__pending",
				"notify__processor__notifier",
				"notify__command__send",
				"event__notifications__sent",
			}
			for stage, id := range order {
				if element := findElement(t, slice, id); element.Stage != stage {
					t.Errorf("element %q stage = %d, want %d", id, element.Stage, stage)
				}
			}
			if slice.StageCount != len(order) {
				t.Errorf("stage count = %d, want %d", slice.StageCount, len(order))
			}
			input := findElement(t, slice, order[0])
			if !input.Given || input.External != test.external {
				t.Errorf("upstream event given/external = %t/%t, want true/%t", input.Given, input.External, test.external)
			}
			if findElement(t, slice, order[4]).Given {
				t.Error("outcome event is marked as an upstream input")
			}
			wantEdges := []Edge{
				{From: order[0], To: order[1]},
				{From: order[1], To: order[2]},
				{From: order[2], To: order[3]},
				{From: order[3], To: order[4]},
				{From: order[4], To: order[1]},
			}
			if len(view.Edges) != len(wantEdges) {
				t.Errorf("edge count = %d, want %d", len(view.Edges), len(wantEdges))
			}
			for _, edge := range wantEdges {
				if !hasEdge(view, edge.From, edge.To) {
					t.Errorf("missing semantic edge %q -> %q", edge.From, edge.To)
				}
			}
		})
	}
}

// The sent event removes completed work from the same read model that drives
// the processor. Its feedback edge must remain semantic, not push the read
// model to the right of the resulting event.
const automationFeedbackSource = `
board {
  workflows = %s
}

bounded_context "upstream" {
  external = %t
  event "queued" {}
}

bounded_context "notifications" {
  event "sent" {}
}

automation "notify" {
  readmodel "pending" {
    question = "Which queued notifications have not been sent?"
    from     = [event.upstream.queued, event.notifications.sent]
    to       = [processor.notifier]
  }

  processor "notifier" {
    to = [command.send]
  }

  command "send" {
    to = [event.notifications.sent]
  }
}
`
