package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const v04Model = `bounded_context "clinic" {
  field_type "pet_id" {
    type = "UUID"
  }

  event "pet_registered" {
    field "pet_id" {}
  }
}

state_change "register_pet" {
  command "register" {
    to = [event.clinic.pet_registered]

    field "pet_id" {}
  }
}
`

func TestParseCommand_RecognizesMigrateInvocations(t *testing.T) {
	for _, test := range []struct {
		args  []string
		write bool
	}{
		{args: []string{"migrate", "model.em.hcl"}},
		{args: []string{"migrate", "-w", "model.em.hcl"}, write: true},
	} {
		command, err := parseCommand("emhcl", test.args)
		if err != nil {
			t.Fatalf("parseCommand(%q): %v", test.args, err)
		}
		if command != (cliCommand{kind: migrateCommand, path: "model.em.hcl", write: test.write}) {
			t.Fatalf("command = %#v", command)
		}
	}
	if _, err := parseCommand("emhcl", []string{"migrate", "model.hcl"}); err == nil || err.Error() != modelPathError {
		t.Fatalf("err = %v, want %q", err, modelPathError)
	}
	if !strings.Contains(usageMessage("emhcl"), "migrate [-w] <model.em.hcl | folder>") {
		t.Fatalf("usage lacks migrate: %s", usageMessage("emhcl"))
	}
}

func TestRun_MigratePrintsOrWritesTheMigratedModel(t *testing.T) {
	path := writeModel(t, "model.em.hcl", v04Model)

	printed := runCLI(t, "migrate", path)
	if printed.exitCode != 0 {
		t.Fatalf("migrate failed: %+v", printed)
	}
	if !strings.HasPrefix(printed.stdout, "board {\n  workflows = [\n    workflow.register_pet,\n  ]\n}\n") || strings.Contains(printed.stdout, "# ==>") {
		t.Fatalf("stdout:\n%s", printed.stdout)
	}
	if !strings.Contains(printed.stderr, "types: wrote 2 explicit field types\n") {
		t.Fatalf("stderr lacks the report: %s", printed.stderr)
	}
	if source, _ := os.ReadFile(path); string(source) != v04Model {
		t.Fatal("migrate without -w changed the file")
	}

	written := runCLI(t, "migrate", "-w", path)
	if written.exitCode != 0 || written.stdout != "" {
		t.Fatalf("migrate -w: %+v", written)
	}
	if source, _ := os.ReadFile(path); string(source) != printed.stdout {
		t.Fatalf("written file differs from printed output:\n%s", source)
	}
}

func TestRun_MigrateFolderWritesOnlyChangedMembers(t *testing.T) {
	dir := t.TempDir()
	catalog := filepath.Join(dir, "a.em.hcl")
	flows := filepath.Join(dir, "b.em.hcl")
	untouched := filepath.Join(dir, "c.em.hcl")
	sources := map[string]string{
		catalog:   "bounded_context \"clinic\" {\n  title = \"Clinic\"\n\n  event \"pet_registered\" {\n    title = \"Pet registered\"\n  }\n}\n",
		flows:     "state_view \"directory\" {\n  readmodel \"pets\" {\n    question = \"Which pets?\"\n    from     = [event.clinic.pet_registered]\n  }\n}\n",
		untouched: "hotspot \"open\" {\n  question = \"Who pays?\"\n}\n",
	}
	for path, source := range sources {
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Unix(1, 0)
	if err := os.Chtimes(untouched, old, old); err != nil {
		t.Fatal(err)
	}

	printed := runCLI(t, "migrate", dir)
	if printed.exitCode != 0 {
		t.Fatalf("migrate failed: %+v", printed)
	}
	for _, path := range []string{catalog, flows, untouched} {
		if !strings.Contains(printed.stdout, "# ==> "+path+" <==\n") {
			t.Fatalf("stdout lacks the %s header:\n%s", path, printed.stdout)
		}
	}

	written := runCLI(t, "migrate", "-w", dir)
	if written.exitCode != 0 {
		t.Fatalf("migrate -w failed: %+v", written)
	}
	source, _ := os.ReadFile(catalog)
	if !strings.HasPrefix(string(source), "board {\n  workflows = [\n    workflow.directory,\n  ]\n}\n") {
		t.Fatalf("catalog:\n%s", source)
	}
	if info, err := os.Stat(untouched); err != nil || !info.ModTime().Equal(old) {
		t.Fatalf("unchanged member was rewritten: %v", err)
	}
}

func TestRun_MigrateErrorWritesNothing(t *testing.T) {
	source := "board {\n  workflows = []\n}\n"
	path := writeModel(t, "model.em.hcl", source)
	result := runCLI(t, "migrate", "-w", path)
	if result.exitCode != 1 || result.stdout != "" || !strings.Contains(result.stderr, "Existing board block") {
		t.Fatalf("result = %+v", result)
	}
	if written, _ := os.ReadFile(path); string(written) != source {
		t.Fatal("migrate wrote a file after an error")
	}
}
