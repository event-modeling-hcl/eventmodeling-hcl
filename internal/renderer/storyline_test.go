package renderer

import (
	"encoding/json"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

// storylineSource has storylines whose ID order differs from the board order.
// The chapters list of "intake" differs from the board order too. "aside" has
// only an off-board workflow, and "billing" is in no storyline.
const storylineSource = `bounded_context "clinic" {}

state_change "register" {}
state_change "check_in" {}
state_change "treat" {}
state_change "pay" {}
state_change "archive" {}

board {
  workflows = [workflow.register, workflow.check_in, workflow.treat, workflow.pay]
}

chapter "registration" {
  title     = "Registration"
  workflows = [workflow.register]
}

chapter "arrival" {
  title     = "Arrival"
  workflows = [workflow.check_in]
}

chapter "treatment" {
  title     = "Treatment"
  workflows = [workflow.treat]
}

chapter "billing" {
  title     = "Billing"
  workflows = [workflow.pay]
}

chapter "archives" {
  workflows = [workflow.archive]
}

storyline "zz_intake" {
  title    = "Intake"
  chapters = [chapter.arrival, chapter.registration]
}

storyline "care" {
  title    = "Care"
  chapters = [chapter.treatment]
}

storyline "aside" {
  chapters = [chapter.archives]
}
`

func TestBuildViewModel_OrdersStorylinesByTheBoardAndChaptersByStoryline(t *testing.T) {
	loaded, diagnostics := loadTestModel(t, "storylines.em.hcl", []byte(storylineSource))
	if diagnostics.HasErrors() {
		t.Fatalf("load model: %s", diagnostics.Error())
	}

	view := BuildViewModel("storylines.em.hcl", loaded)

	wantStorylines := []Storyline{
		{ID: "zz_intake", Title: "Intake", Chapters: []string{"arrival", "registration"}},
		{ID: "care", Title: "Care", Chapters: []string{"treatment"}},
		{ID: "aside", Title: "aside", Chapters: []string{"archives"}},
	}
	if !reflect.DeepEqual(view.Storylines, wantStorylines) {
		t.Fatalf("storylines = %#v, want board order, then the storylines without a board workflow %#v", view.Storylines, wantStorylines)
	}
	var chapters []string
	for _, chapter := range view.Chapters {
		chapters = append(chapters, chapter.ID)
	}
	if want := []string{"arrival", "registration", "treatment", "archives", "billing"}; !reflect.DeepEqual(chapters, want) {
		t.Fatalf("chapters = %v, want storyline chapters in list order, then the others by ID %v", chapters, want)
	}
}

func TestBuildViewModel_KeepsChapterOrderWithoutStorylines(t *testing.T) {
	source := storylineSource[:strings.Index(storylineSource, `storyline "zz_intake"`)]
	loaded, diagnostics := loadTestModel(t, "chapters.em.hcl", []byte(source))
	if diagnostics.HasErrors() {
		t.Fatalf("load model: %s", diagnostics.Error())
	}

	view := BuildViewModel("chapters.em.hcl", loaded)

	var chapters []string
	for _, chapter := range view.Chapters {
		chapters = append(chapters, chapter.ID)
	}
	if want := []string{"archives", "arrival", "billing", "registration", "treatment"}; !reflect.DeepEqual(chapters, want) {
		t.Fatalf("chapters = %v, want ID order %v", chapters, want)
	}
	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal view: %v", err)
	}
	if !strings.Contains(string(encoded), `"storylines":[]`) {
		t.Fatalf("JSON missing an empty storylines list: %s", encoded)
	}
}

func TestRenderHTML_GroupsChaptersByStoryline(t *testing.T) {
	loaded, diagnostics := loadTestModel(t, "storylines.em.hcl", []byte(storylineSource))
	if diagnostics.HasErrors() {
		t.Fatalf("load model: %s", diagnostics.Error())
	}

	html, err := RenderHTML(BuildViewModel("storylines.em.hcl", loaded))
	if err != nil {
		t.Fatalf("render HTML: %v", err)
	}
	for _, expected := range []string{
		`"storylines":[{"id":"zz_intake","title":"Intake","chapters":["arrival","registration"]}`,
		`chapter-menu-storyline`,
		`storyline-group`,
		`index-storyline-group`,
		`Not in a storyline`,
	} {
		if !strings.Contains(html, expected) {
			t.Errorf("HTML missing storyline hook %q", expected)
		}
	}
}

// TestChapterGroupsFollowTheStorylines runs the top of viewer.js, which builds
// the chapter groups of the picker, the overview, and the Index.
func TestChapterGroupsFollowTheStorylines(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is required to exercise chapter grouping")
	}
	prefix, _, found := strings.Cut(viewerJS, "/* --------------------------- constants")
	if !found {
		t.Fatal("viewer initialization boundary not found")
	}
	slices := `[{"id":"register","elements":[]},{"id":"check_in","elements":[]},{"id":"treat","elements":[]},{"id":"pay","elements":[]},{"id":"loose","elements":[]}]`
	chapters := `[{"id":"arrival","title":"Arrival","slices":["check_in"]},{"id":"registration","title":"Registration","slices":["register"]},` +
		`{"id":"treatment","title":"Treatment","slices":["treat"]},{"id":"billing","title":"Billing","slices":["pay"]}]`
	cases := []struct {
		name       string
		model      string
		assertions string
	}{
		{
			name:  "storylines",
			model: `{"slices":` + slices + `,"chapters":` + chapters + `,"storylines":[{"id":"intake","title":"Intake","chapters":["arrival","registration"]},{"id":"care","title":"Care","chapters":["treatment"]}]}`,
			assertions: `
assert.deepStrictEqual(chapters.map(c=>c.id), ["arrival","registration","treatment","billing","__unchaptered"]);
assert.deepStrictEqual(chapterGroups.map(g=>[g.title, g.ui, g.chapters.map(c=>c.id)]), [
  ["Intake", false, ["arrival","registration"]],
  ["Care", false, ["treatment"]],
  ["Not in a storyline", true, ["billing","__unchaptered"]],
]);
assert.strictEqual(chapterGroups[0].storyline.id, "intake");
assert.strictEqual(chapterGroups[2].storyline, null);
assert.deepStrictEqual(chapterGroups.flatMap(g=>g.chapters), chapters);`,
		},
		{
			name:  "no storylines",
			model: `{"slices":` + slices + `,"chapters":` + chapters + `,"storylines":[]}`,
			assertions: `
assert.strictEqual(STORYLINES.length, 0);
assert.strictEqual(chapterGroups.length, 1);
assert.deepStrictEqual(chapterGroups[0].chapters, chapters);`,
		},
		{
			name:  "no chapters",
			model: `{"slices":` + slices + `}`,
			assertions: `
assert.deepStrictEqual(chapterGroups, []);`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			script := "const assert = require('node:assert/strict');\nconst location = {hash:\"\"};\n" + strings.Replace(prefix, modelMarker, tc.model, 1) + tc.assertions
			cmd := exec.Command(node, "-")
			cmd.Stdin = strings.NewReader(script)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("chapter grouping: %v\n%s", err, output)
			}
		})
	}
}
