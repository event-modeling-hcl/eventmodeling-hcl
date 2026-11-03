package source

import (
	"strings"
	"testing"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
	"github.com/hashicorp/hcl/v2"
)

const addressCatalog = `actor "clerk" {
  auth_required = true
}

team "desk" {}

system "partner" {}

bounded_context "clinic" {
  aggregate "pet" {}

  field_type "pet_id" {
    type = "Int"
  }

  event "pet_registered" {
    field "pet_id" {
      type = field_type.pet_id
    }
  }
}

chapter "intake" {
  workflows = [workflow.register_pet]
}

hotspot "open_question" {
  question = "Who registers pets?"
}

board {
  workflows = [workflow.register_pet]
}
`

const addressWorkflow = `state_change "register_pet" {
  screen "pet_form" {
    to = [command.register]
  }

  command "register" {
    to = [event.clinic.pet_registered]
  }

  table "pets" {
    title = "Pets"
  }

  scenario "registers" {
    given {}
  }
}
`

func TestAddressAt_NamesTheInnermostAddressableDeclaration(t *testing.T) {
	document := decodeFiles(t, map[string]string{"a.em.hcl": addressCatalog, "b.em.hcl": addressWorkflow})

	for _, test := range []struct {
		file   string
		marker string
		want   string
	}{
		{"a.em.hcl", "auth_required", "actor.clerk"},
		{"a.em.hcl", `team "desk"`, "team.desk"},
		{"a.em.hcl", `system "partner"`, "system.partner"},
		{"a.em.hcl", `aggregate "pet"`, "aggregate.clinic.pet"},
		{"a.em.hcl", `type = "Int"`, "field_type.clinic.pet_id"},
		{"a.em.hcl", "type = field_type.pet_id", "event.clinic.pet_registered"},
		{"b.em.hcl", "to = [command.register]", "screen.register_pet.pet_form"},
		{"b.em.hcl", `command "register"`, "command.register_pet.register"},
		{"b.em.hcl", `title = "Pets"`, "workflow.register_pet"},
		{"b.em.hcl", "given {}", "workflow.register_pet"},
	} {
		got, ok := document.AddressAt(rangeOf(t, test.file, sources[test.file], test.marker))
		if !ok || got.String() != test.want {
			t.Errorf("AddressAt(%s %q) = %q, %v; want %q", test.file, test.marker, got, ok, test.want)
		}
	}
}

func TestAddressAt_GivesNoAddressOutsideAddressableDeclarations(t *testing.T) {
	document := decodeFiles(t, map[string]string{"a.em.hcl": addressCatalog, "b.em.hcl": addressWorkflow})

	for _, marker := range []string{"workflows = [workflow.register_pet]\n}\n\nhotspot", `question = "Who`, "board {"} {
		if got, ok := document.AddressAt(rangeOf(t, "a.em.hcl", addressCatalog, marker)); ok {
			t.Errorf("AddressAt(%q) = %q, want no address", marker, got)
		}
	}
	if got, ok := document.AddressAt(document.Parsed().Start()); ok {
		t.Errorf("AddressAt(model start) = %q, want no address", got)
	}
}

// A block that starts at byte 0 keeps its address for a diagnostic on its
// header. Only the empty start range of the model has no address.
func TestAddressAt_BlockAtFileStartKeepsItsAddress(t *testing.T) {
	document := decodeFiles(t, map[string]string{"a.em.hcl": addressCatalog})
	header := hcl.Range{
		Filename: "a.em.hcl",
		Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
		End:      hcl.Pos{Line: 1, Column: 15, Byte: len(`actor "clerk"`)},
	}

	if got, ok := document.AddressAt(header); !ok || got.String() != "actor.clerk" {
		t.Fatalf("AddressAt(header at byte 0) = %q, %v; want actor.clerk", got, ok)
	}
}

// Each member file starts its byte offsets at 0. An offset in one file must
// never match a block of another file.
func TestAddressAt_MatchesTheFileOfTheRange(t *testing.T) {
	first := "actor \"first\" {\n  auth_required = true\n}\n"
	second := "actor \"second\" {\n  auth_required = true\n}\n"
	document := decodeFiles(t, map[string]string{"a.em.hcl": first, "b.em.hcl": second})

	got, ok := document.AddressAt(rangeOf(t, "b.em.hcl", second, "auth_required"))
	if !ok || got.String() != "actor.second" {
		t.Fatalf("AddressAt(b.em.hcl) = %q, %v; want actor.second", got, ok)
	}
}

var sources = map[string]string{"a.em.hcl": addressCatalog, "b.em.hcl": addressWorkflow}

func decodeFiles(t *testing.T, files map[string]string) *Document {
	t.Helper()
	var members []syntax.File
	for name, source := range files {
		members = append(members, syntax.File{Name: name, Source: []byte(source)})
	}
	parsed, diagnostics := syntax.ParseFiles(members)
	if diagnostics.HasErrors() {
		t.Fatalf("parse: %s", diagnostics.Error())
	}
	return Decode(parsed)
}

// rangeOf returns a one-byte range at the first occurrence of marker in source.
func rangeOf(t *testing.T, file, source, marker string) hcl.Range {
	t.Helper()
	offset := strings.Index(source, marker)
	if offset < 0 {
		t.Fatalf("marker %q not in %s", marker, file)
	}
	return hcl.Range{Filename: file, Start: hcl.Pos{Byte: offset}, End: hcl.Pos{Byte: offset + 1}}
}
