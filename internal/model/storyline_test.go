package model

import (
	"reflect"
	"testing"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
)

// storylineModel declares its storylines out of ID order. The chapters list of
// "intake" is out of ID order and out of board order.
var storylineModel = []string{
	`state_change "register" {}
state_change "check_in" {}
state_change "treat" {}
board {
  workflows = [workflow.register, workflow.check_in, workflow.treat]
}
`,
	`chapter "registration" {
  workflows = [workflow.register]
}
chapter "arrival" {
  workflows = [workflow.check_in]
}
chapter "treatment" {
  workflows = [workflow.treat]
}
`,
	`storyline "intake" {
  title       = "Intake"
  description = "The pet arrives."
  chapters    = [chapter.registration, chapter.arrival]
}
storyline "care" {
  chapters = [chapter.treatment]
}
`,
}

func TestBuild_SortsStorylinesByIDAndKeepsTheirChapterOrder(t *testing.T) {
	built := buildFiles(t, []syntax.File{{Name: "model.em.hcl", Source: []byte(storylineModel[0] + storylineModel[1] + storylineModel[2])}})

	want := []Storyline{
		{ID: "care", Chapters: []string{"chapter.treatment"}},
		{ID: "intake", Title: "Intake", TitlePresent: true, Description: "The pet arrives.", Chapters: []string{"chapter.registration", "chapter.arrival"}},
	}
	if !reflect.DeepEqual(built.Storylines, want) {
		t.Fatalf("storylines = %#v, want %#v", built.Storylines, want)
	}
}

func TestBuild_StorylinesAreIndependentOfFileLayout(t *testing.T) {
	oneFile := buildFiles(t, []syntax.File{{Name: "model.em.hcl", Source: []byte(storylineModel[0] + storylineModel[1] + storylineModel[2])}})
	split := buildFiles(t, []syntax.File{
		{Name: "folder/a.em.hcl", Source: []byte(storylineModel[2])},
		{Name: "folder/b.em.hcl", Source: []byte(storylineModel[1])},
		{Name: "folder/c.em.hcl", Source: []byte(storylineModel[0])},
	})

	if !reflect.DeepEqual(oneFile, split) {
		t.Fatalf("split model = %+v, want the one-file model %+v", split, oneFile)
	}
}

func TestBuild_ModelWithoutStorylinesHasNone(t *testing.T) {
	built := buildFiles(t, []syntax.File{{Name: "model.em.hcl", Source: []byte(storylineModel[0] + storylineModel[1])}})

	if built.Storylines != nil {
		t.Fatalf("storylines = %#v, want nil", built.Storylines)
	}
}
