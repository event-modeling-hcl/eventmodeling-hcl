package app

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestLiteralEmptyTitleIsDistinctFromOmission(t *testing.T) {
	const prefix = "board {\n workflows = [workflow.place_order]\n}\n"
	absent, absentDiagnostics := ValidatedModel("absent.em.hcl", []byte(prefix+"state_change \"place_order\" {}\n"), Valid)
	explicit, explicitDiagnostics := ValidatedModel("explicit.em.hcl", []byte(prefix+"state_change \"place_order\" {\n title = \"\"\n}\n"), Valid)
	if absentDiagnostics.HasCoreErrors() || explicitDiagnostics.HasCoreErrors() {
		t.Fatalf("invalid title fixtures: %v, %v", absentDiagnostics, explicitDiagnostics)
	}
	absentJSON, _ := json.Marshal(absent)
	explicitJSON, _ := json.Marshal(explicit)
	if bytes.Equal(absentJSON, explicitJSON) {
		t.Fatal("IR loses the distinction between an absent title and a literal empty title")
	}
}

func TestExportPreservesLiteralEmptyWorkflowTitle(t *testing.T) {
	result := Export("explicit.em.hcl", []byte("board {\n workflows = [workflow.place_order]\n}\nstate_change \"place_order\" {\n title = \"\"\n}\n"))
	if result.Diagnostics.HasErrors() {
		t.Fatal(result.Diagnostics)
	}
	var document struct {
		Slices []struct {
			Title string `json:"title"`
		} `json:"slices"`
	}
	if err := json.Unmarshal([]byte(result.JSON), &document); err != nil {
		t.Fatal(err)
	}
	if len(document.Slices) != 1 || document.Slices[0].Title != "" {
		t.Fatalf("exported title = %+v, want literal empty title", document.Slices)
	}
}
