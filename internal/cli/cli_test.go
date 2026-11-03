package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/app"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/version"
)

func TestParseCommand_RecognizesValidateInvocation(t *testing.T) {
	// Given a validate invocation with an Event Modeling model path.
	args := []string{"validate", "model.em.hcl"}

	// When its command is parsed.
	command, err := parseCommand("emhcl", args)

	// Then it produces a validate command without an error.
	if err != nil {
		t.Fatalf("err = %q, want nil", err)
	}
	if got, want := command, (cliCommand{kind: validateCommand, path: "model.em.hcl", profile: app.Valid}); got != want {
		t.Fatalf("command = %#v, want %#v", got, want)
	}
}

func TestParseCommand_RecognizesValidationProfile(t *testing.T) {
	command, err := parseCommand("emhcl", []string{"validate", "--profile", "strict", "model.em.hcl"})

	if err != nil {
		t.Fatalf("err = %q, want nil", err)
	}
	if got, want := command, (cliCommand{kind: validateCommand, path: "model.em.hcl", profile: app.Strict}); got != want {
		t.Fatalf("command = %#v, want %#v", got, want)
	}
}

func TestParseCommand_RecognizesFormatterInvocations(t *testing.T) {
	for _, test := range []struct {
		args  []string
		write bool
	}{
		{args: []string{"fmt", "model.em.hcl"}},
		{args: []string{"fmt", "-w", "model.em.hcl"}, write: true},
	} {
		command, err := parseCommand("emhcl", test.args)
		if err != nil {
			t.Fatalf("parseCommand(%q): %v", test.args, err)
		}
		if command.kind != formatCommand || command.path != "model.em.hcl" || command.write != test.write {
			t.Fatalf("command = %#v", command)
		}
	}
}

func TestParseCommand_RecognizesDiagramInvocations(t *testing.T) {
	tests := []struct {
		args   []string
		output string
	}{
		{args: []string{"diagram", "model.em.hcl"}},
		{args: []string{"diagram", "model.em.hcl", "-o", "model.html"}, output: "model.html"},
		{args: []string{"diagram", "--output", "model.html", "model.em.hcl"}, output: "model.html"},
	}
	for _, test := range tests {
		command, err := parseCommand("emhcl", test.args)
		if err != nil {
			t.Fatalf("parseCommand(%q): %v", test.args, err)
		}
		if command.kind != diagramCommand || command.path != "model.em.hcl" || command.output != test.output {
			t.Fatalf("command = %#v", command)
		}
	}
}

func TestParseCommand_RecognizesServeInvocation(t *testing.T) {
	// Given a bare serve invocation with just a model path.
	args := []string{"serve", "model.em.hcl"}

	// When its command is parsed.
	command, err := parseCommand("emhcl", args)

	// Then it produces a serve command with the documented defaults.
	if err != nil {
		t.Fatalf("err = %q, want nil", err)
	}
	want := cliCommand{kind: serveCommand, path: "model.em.hcl", profile: app.Valid, addr: defaultServeAddr, port: defaultServePort}
	if command != want {
		t.Fatalf("command = %#v, want %#v", command, want)
	}
}

func TestParseCommand_RejectsServePortsOutsideTheTCPRange(t *testing.T) {
	for _, port := range []string{"-1", "65536"} {
		_, err := parseCommand("emhcl", []string{"serve", "--port", port, "model.em.hcl"})
		if err == nil || !strings.Contains(err.Error(), "between 0 and 65535") {
			t.Errorf("port %s error = %v, want TCP range error", port, err)
		}
	}
}

func TestParseCommand_RecognizesServeFlags(t *testing.T) {
	// Given a serve invocation overriding every optional flag, in any order.
	args := []string{"serve", "--addr", "127.0.0.1", "--port", "9090", "--profile", "strict", "model.em.hcl"}

	// When its command is parsed.
	command, err := parseCommand("emhcl", args)

	// Then every flag is reflected in the resulting command.
	if err != nil {
		t.Fatalf("err = %q, want nil", err)
	}
	want := cliCommand{kind: serveCommand, path: "model.em.hcl", profile: app.Strict, addr: "127.0.0.1", port: 9090}
	if command != want {
		t.Fatalf("command = %#v, want %#v", command, want)
	}
}

func TestParseCommand_RejectsServeWithoutModelPath(t *testing.T) {
	_, err := parseCommand("emhcl", []string{"serve"})
	if err == nil {
		t.Fatal("expected an error for a serve invocation with no model path")
	}
}

func TestParseCommand_RejectsServeNonNumericPort(t *testing.T) {
	_, err := parseCommand("emhcl", []string{"serve", "--port", "notanumber", "model.em.hcl"})
	if err == nil {
		t.Fatal("expected an error for a non-numeric --port")
	}
}

func TestParseCommand_RejectsServeInvalidProfile(t *testing.T) {
	_, err := parseCommand("emhcl", []string{"serve", "--profile", "bogus", "model.em.hcl"})
	if got, want := err, "profile must be workshop, valid, or strict"; got == nil || got.Error() != want {
		t.Fatalf("err = %v, want %q", got, want)
	}
}

func TestParseCommand_RejectsServeUnsupportedExtension(t *testing.T) {
	_, err := parseCommand("emhcl", []string{"serve", "model.hcl"})
	if got, want := err, "model path must be a .em.hcl file or a folder"; got == nil || got.Error() != want {
		t.Fatalf("err = %v, want %q", got, want)
	}
}

func TestParseCommand_RejectsUnsupportedExtension(t *testing.T) {
	// Given a validate invocation with a non-Event-Modeling extension.
	args := []string{"validate", "model.hcl"}

	// When its command is parsed.
	_, err := parseCommand("emhcl", args)

	// Then it produces the extension error.
	if got, want := err, "model path must be a .em.hcl file or a folder"; got == nil || got.Error() != want {
		t.Fatalf("err = %v, want %q", got, want)
	}
}

func TestFormatDiagnostics_FormatsSourceLocatedError(t *testing.T) {
	// Given one source-located validation error.
	diagnostics := []app.Diagnostic{{
		Code:     "EM007",
		Severity: "Error",
		Summary:  "Invalid attribute value",
		Detail:   "title must be a string.",
		Filename: "model.em.hcl",
		Line:     3,
		Column:   11,
	}}

	// When its diagnostics are formatted.
	got := formatDiagnostics(diagnostics)

	// Then it returns the CLI rendering without writing to a stream.
	const want = "model.em.hcl:3:11: Error EM007: Invalid attribute value: title must be a string.\n"
	if got != want {
		t.Fatalf("formatted diagnostics = %q, want %q", got, want)
	}
}

func TestRun_ValidatesShippedExample(t *testing.T) {
	paths := modelPaths(t, "examples")
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			// Given one shipped Event Modeling example.

			// When the validate command runs for that file.
			result := runCLI(t, "validate", path)

			// Then the example validates successfully.
			if result.exitCode != 0 {
				t.Fatalf("exit code = %d, stderr = %q", result.exitCode, result.stderr)
			}
		})
	}
}

func TestRun_ValidatesValidFixture(t *testing.T) {
	paths := modelPaths(t, "testdata", "valid")
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			// Given one valid structural fixture.

			// When the validate command runs for that file.
			result := runCLI(t, "validate", path)

			// Then the fixture validates successfully.
			if result.exitCode != 0 {
				t.Fatalf("exit code = %d, stderr = %q", result.exitCode, result.stderr)
			}
		})
	}
}

func TestRun_InvalidFixtureFailsValidation(t *testing.T) {
	paths := modelPaths(t, "testdata", "invalid")
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			// Given one intentionally invalid fixture.

			// When the validate command runs for that file.
			result := runCLI(t, "validate", path)

			// Then validation fails.
			if result.exitCode == 0 {
				t.Fatal("exit code = 0, want validation failure")
			}
		})
	}
}

func TestRun_CanonicalInvalidFixturesReportExpectedCodes(t *testing.T) {
	tests := []struct {
		path string
		code string
	}{
		{path: filepath.Join("..", "..", "testdata", "invalid", "reverse-flow.em.hcl"), code: "EM201"},
		{path: filepath.Join("..", "..", "testdata", "invalid", "state-view-when.em.hcl"), code: "EM301"},
	}

	for _, test := range tests {
		t.Run(filepath.Base(test.path), func(t *testing.T) {
			result := runCLI(t, "validate", test.path)
			if result.exitCode != 1 {
				t.Fatalf("exit code = %d, want 1; stderr = %q", result.exitCode, result.stderr)
			}
			if !strings.Contains(result.stderr, test.code) {
				t.Fatalf("stderr = %q, want %s", result.stderr, test.code)
			}
		})
	}
}

func TestRun_ValidateValidModelPrintsSuccess(t *testing.T) {
	// Given a valid Event Modeling model.
	modelPath := writeModel(t, "valid.em.hcl", `bounded_context "example" {
  title = "Example"
}

state_change "example" {
  title = "Example"
}

board {
  workflows = [workflow.example]
}
`)

	// When the validate command runs.
	result := runCLI(t, "validate", modelPath)

	// Then it returns success and reports the validated path.
	if result.exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %q", result.exitCode, result.stderr)
	}
	if got, want := result.stdout, modelPath+" valid\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if result.stderr != "" {
		t.Fatalf("stderr = %q, want empty", result.stderr)
	}
}

func TestRun_ValidatePrintsWarningsWithoutFailing(t *testing.T) {
	modelPath := writeModel(t, "warning.em.hcl", `state_change "example" {
  title = "Example"
  command "submit" { title = "Submit" }
}
board { workflows = [workflow.example] }`)

	result := runCLI(t, "validate", modelPath)

	if result.exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %q", result.exitCode, result.stderr)
	}
	if !strings.Contains(result.stderr, "Warning EM404: Every command has a reason") {
		t.Fatalf("stderr = %q, want command-reason warning", result.stderr)
	}
}

func TestRun_ValidateStrictProfileEscalatesCommandReason(t *testing.T) {
	modelPath := writeModel(t, "warning.em.hcl", `state_change "example" {
  command "submit" {}
}
board { workflows = [workflow.example] }`)

	result := runCLI(t, "validate", "--profile", "strict", modelPath)

	if result.exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", result.exitCode, result.stderr)
	}
	if !strings.Contains(result.stderr, "Error EM404: Every command has a reason") {
		t.Fatalf("stderr = %q, want strict command-reason error", result.stderr)
	}
}

func TestRun_FormatWritesToStdoutOrInPlace(t *testing.T) {
	source := `state_change "example" {
  command "submit" {
    to = [event.example.submitted]
    title = "Submit"
  }
}`

	stdoutPath := writeModel(t, "stdout.em.hcl", source)
	stdoutResult := runCLI(t, "fmt", stdoutPath)
	if stdoutResult.exitCode != 0 || stdoutResult.stderr != "" {
		t.Fatalf("stdout format result = %#v", stdoutResult)
	}
	if !strings.Contains(stdoutResult.stdout, "title = \"Submit\"") {
		t.Fatalf("stdout = %q, want formatted source", stdoutResult.stdout)
	}

	writePath := writeModel(t, "write.em.hcl", source)
	writeResult := runCLI(t, "fmt", "-w", writePath)
	if writeResult.exitCode != 0 || writeResult.stdout != "" || writeResult.stderr != "" {
		t.Fatalf("write format result = %#v", writeResult)
	}
	written, err := os.ReadFile(writePath)
	if err != nil {
		t.Fatalf("read formatted model: %v", err)
	}
	if !strings.Contains(string(written), "title = \"Submit\"") {
		t.Fatalf("written = %q, want formatted source", written)
	}
}

func TestRun_DiagramWritesHTMLToStdoutOrFile(t *testing.T) {
	modelPath := writeModel(t, "valid.em.hcl", "state_change \"example\" {}\nboard { workflows = [workflow.example] }\n")

	stdoutResult := runCLI(t, "diagram", modelPath)
	if stdoutResult.exitCode != 0 || stdoutResult.stderr != "" {
		t.Fatalf("stdout diagram result = %#v", stdoutResult)
	}
	if !strings.Contains(stdoutResult.stdout, "<!doctype html>") || !strings.Contains(stdoutResult.stdout, "const MODEL =") {
		t.Fatalf("stdout does not contain rendered HTML")
	}

	outputPath := filepath.Join(t.TempDir(), "model.html")
	fileResult := runCLI(t, "diagram", "--output", outputPath, modelPath)
	if fileResult.exitCode != 0 || fileResult.stdout != "" {
		t.Fatalf("file diagram result = %#v", fileResult)
	}
	written, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read diagram: %v", err)
	}
	if !strings.Contains(string(written), "<!doctype html>") {
		t.Fatal("output file does not contain rendered HTML")
	}
}

func TestRun_DiagramRejectsInvalidModelWithoutOverwritingOutput(t *testing.T) {
	modelPath := writeModel(t, "invalid.em.hcl", `state_change "example" {
  command "submit" { to = [event.example.missing] }
}
board { workflows = [workflow.example] }`)
	outputPath := filepath.Join(t.TempDir(), "existing.html")
	const existing = "keep this"
	if err := os.WriteFile(outputPath, []byte(existing), 0o600); err != nil {
		t.Fatalf("write existing output: %v", err)
	}

	result := runCLI(t, "diagram", modelPath, "-o", outputPath)
	if result.exitCode != 1 || result.stdout != "" || !strings.Contains(result.stderr, "EM102") {
		t.Fatalf("diagram result = %#v", result)
	}
	written, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read existing output: %v", err)
	}
	if got := string(written); got != existing {
		t.Fatalf("output = %q, want existing content", got)
	}
}

func TestRun_DiagramReportsWarningsAndStillRenders(t *testing.T) {
	modelPath := writeModel(t, "warning.em.hcl", `state_change "example" {
  command "submit" {}
}
board { workflows = [workflow.example] }`)

	result := runCLI(t, "diagram", modelPath)
	if result.exitCode != 0 || !strings.Contains(result.stdout, "<!doctype html>") {
		t.Fatalf("diagram result = %#v", result)
	}
	if !strings.Contains(result.stderr, "Warning EM404") {
		t.Fatalf("stderr = %q, want warning", result.stderr)
	}
}

func TestRun_ValidateRejectsNonEventModelExtension(t *testing.T) {
	// Given a valid model whose file name does not end in .em.hcl.
	modelPath := writeModel(t, "valid.hcl", `bounded_context "example" {
  title = "Example"
}

state_change "example" {
  title = "Example"
}
`)

	// When the validate command runs.
	result := runCLI(t, "validate", modelPath)

	// Then it returns the usage error for an unsupported extension.
	if result.exitCode != 2 {
		t.Fatalf("exit code = %d, want 2; stderr = %q", result.exitCode, result.stderr)
	}
	if !strings.Contains(result.stderr, "model path must be a .em.hcl file or a folder") {
		t.Fatalf("stderr = %q, want extension diagnostic", result.stderr)
	}
}

// writeFolderModel writes the minimal example split over two files that
// reference each other and returns the folder path.
func writeFolderModel(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"a-catalog.em.hcl": `bounded_context "pet_management" {
  title = "Pet Management"

  aggregate "pet" {
  }

  field_type "pet_id" {
    type         = "Int"
    id_attribute = true
    example      = 5
  }

  event "pet_added" {
    title     = "Pet Added"
    aggregate = aggregate.pet

    field "pet_id" {
      type = field_type.pet_id
    }
  }
}
`,
		"b-slice.em.hcl": `state_change "add_pet" {
  title = "Add Pet"

  screen "add_pet_form" {
    title = "Add Pet Form"
    to    = [command.add_pet_command]
  }

  command "add_pet_command" {
    title     = "Add Pet"
    aggregate = aggregate.pet_management.pet
    to        = [event.pet_management.pet_added]

    field "pet_id" {
      type = field_type.pet_management.pet_id
    }
  }
}

board {
  workflows = [workflow.add_pet]
}
`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

func TestRun_ValidateFolderWhoseFilesReferenceEachOther(t *testing.T) {
	dir := writeFolderModel(t)

	result := runCLI(t, "validate", dir)

	if result.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", result.exitCode, result.stderr)
	}
	if got, want := result.stdout, dir+" valid\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestRun_DiagramAndExportAcceptAFolder(t *testing.T) {
	dir := writeFolderModel(t)

	for _, command := range []string{"diagram", "export"} {
		result := runCLI(t, command, dir)

		if result.exitCode != 0 {
			t.Fatalf("%s: exit code = %d, want 0; stderr = %q", command, result.exitCode, result.stderr)
		}
		if result.stdout == "" {
			t.Fatalf("%s: stdout is empty, want the rendered or exported model", command)
		}
	}
}

func TestRun_FolderDiagnosticsNameTheMemberFile(t *testing.T) {
	dir := writeFolderModel(t)
	broken := filepath.Join(dir, "c-broken.em.hcl")
	if err := os.WriteFile(broken, []byte("bounded_context \"example\" {\n"), 0o600); err != nil {
		t.Fatalf("write broken file: %v", err)
	}

	result := runCLI(t, "validate", dir)

	if result.exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", result.exitCode, result.stderr)
	}
	location := regexp.MustCompile(regexp.QuoteMeta(broken) + `:\d+:\d+: Error EM000:`)
	if !location.MatchString(result.stderr) {
		t.Fatalf("stderr = %q, want a diagnostic located in %s", result.stderr, broken)
	}
	if result.stdout != "" {
		t.Fatalf("stdout = %q, want empty", result.stdout)
	}
}

func TestRun_RejectsModelPathsThatAreNeitherAFileNorAFolder(t *testing.T) {
	existing := filepath.Join(t.TempDir(), "x.txt")
	if err := os.WriteFile(existing, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	missing := filepath.Join(t.TempDir(), "missing")

	for _, command := range []string{"validate", "diagram", "export", "serve", "fmt"} {
		for _, path := range []string{existing, missing, "x.txt"} {
			result := runCLI(t, command, path)

			if result.exitCode != 2 {
				t.Fatalf("%s %s: exit code = %d, want 2; stderr = %q", command, path, result.exitCode, result.stderr)
			}
			if got, want := result.stderr, "model path must be a .em.hcl file or a folder\n"; got != want {
				t.Fatalf("%s %s: stderr = %q, want %q", command, path, got, want)
			}
		}
	}
}

func TestRun_FormatFolderWritesAndChecksOnlyMemberFiles(t *testing.T) {
	dir := writeFolderModel(t)
	nested := filepath.Join(dir, "nested")
	if err := os.Mkdir(nested, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	ignored := filepath.Join(nested, "ignored.em.hcl")
	const unformatted = "actor \"x\" {\nauth_required=true\n}\n"
	if err := os.WriteFile(ignored, []byte(unformatted), 0o600); err != nil {
		t.Fatalf("write nested file: %v", err)
	}

	if result := runCLI(t, "fmt", "-w", dir); result.exitCode != 0 {
		t.Fatalf("fmt -w: %#v", result)
	}
	if result := runCLI(t, "fmt", "--check", dir); result.exitCode != 0 || result.stdout != "" {
		t.Fatalf("fmt --check after fmt -w: %#v", result)
	}
	printed := runCLI(t, "fmt", dir)
	catalog := filepath.Join(dir, "a-catalog.em.hcl")
	if printed.exitCode != 0 || !strings.HasPrefix(printed.stdout, "# ==> "+catalog+" <==\n") {
		t.Fatalf("fmt <folder> = %#v, want each member after a header line", printed)
	}

	source, err := os.ReadFile(catalog)
	if err != nil {
		t.Fatalf("read catalog: %v", err)
	}
	changed := strings.Replace(string(source), `title = "Pet Management"`, `title="Pet Management"`, 1)
	if err := os.WriteFile(catalog, []byte(changed), 0o600); err != nil {
		t.Fatalf("write catalog: %v", err)
	}
	check := runCLI(t, "fmt", "--check", dir)
	if check.exitCode != 1 || check.stdout != catalog+"\n" {
		t.Fatalf("fmt --check = %#v, want exit 1 and only %s", check, catalog)
	}
	if after, _ := os.ReadFile(catalog); string(after) != changed {
		t.Fatal("fmt --check changed a file")
	}
	if after, _ := os.ReadFile(ignored); string(after) != unformatted {
		t.Fatal("fmt -w changed a file in a subfolder")
	}
}

func TestRun_FormatFolderWithSyntaxErrorChangesNoFile(t *testing.T) {
	dir := writeFolderModel(t)
	catalog := filepath.Join(dir, "a-catalog.em.hcl")
	before, err := os.ReadFile(catalog)
	if err != nil {
		t.Fatalf("read catalog: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "c-broken.em.hcl"), []byte("bounded_context \"broken\" {\n"), 0o600); err != nil {
		t.Fatalf("write broken file: %v", err)
	}

	result := runCLI(t, "fmt", "-w", dir)

	if result.exitCode != 1 || !strings.Contains(result.stderr, "c-broken.em.hcl") {
		t.Fatalf("fmt -w = %#v, want exit 1 with a diagnostic in c-broken.em.hcl", result)
	}
	if after, _ := os.ReadFile(catalog); string(after) != string(before) {
		t.Fatal("fmt -w changed a valid member although another member has a syntax error")
	}
}

// validate --format json carries the same diagnostics as the text output, in
// the same order, for every shipped model and fixture under every profile.
func TestRun_ValidateJSONMatchesTextOutput(t *testing.T) {
	paths := modelPaths(t, "examples")
	for _, directory := range []string{"valid", "invalid", "valid-folders", "invalid-folders"} {
		paths = append(paths, modelPaths(t, "testdata", directory)...)
	}
	for _, path := range paths {
		for _, profile := range []string{"workshop", "valid", "strict"} {
			text := runCLI(t, "validate", "--profile", profile, path)
			data := runCLI(t, "validate", "--profile", profile, "--format", "json", path)

			report := decodeValidateReport(t, data)
			if data.exitCode != text.exitCode || report.HasErrors != (data.exitCode == 1) {
				t.Fatalf("%s %s: json exit %d, has_errors %v; text exit %d", path, profile, data.exitCode, report.HasErrors, text.exitCode)
			}
			if got := formatDiagnostics(report.Diagnostics); got != text.stderr {
				t.Fatalf("%s %s: json diagnostics\n%s\nwant text diagnostics\n%s", path, profile, got, text.stderr)
			}
		}
	}
}

func TestRun_ValidateJSONReportsJudgmentAndAddress(t *testing.T) {
	modelPath := writeModel(t, "warning.em.hcl", `state_change "example" {
  command "submit" {}
}
board { workflows = [workflow.example] }`)

	result := runCLI(t, "validate", "--profile", "strict", "--format", "json", modelPath)

	report := decodeValidateReport(t, result)
	if result.exitCode != 1 || !report.HasErrors || report.HasCoreErrors || report.Profile != "strict" {
		t.Fatalf("result = %#v, report = %#v; want a policy rejection without core errors", result, report)
	}
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Code == "EM404" {
			if !diagnostic.Judgment || diagnostic.Severity != "Error" || diagnostic.Address != "command.example.submit" || diagnostic.Filename != modelPath {
				t.Fatalf("EM404 = %#v", diagnostic)
			}
			return
		}
	}
	t.Fatalf("diagnostics = %#v, want EM404", report.Diagnostics)
}

func TestRun_DiffIgnoresFileLayoutAndListsDeclarationChanges(t *testing.T) {
	base := writeFolderModel(t)
	var joined strings.Builder
	for _, name := range []string{"a-catalog.em.hcl", "b-slice.em.hcl"} {
		source, err := os.ReadFile(filepath.Join(base, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		joined.Write(source)
	}
	moved := writeModel(t, "joined.em.hcl", strings.ReplaceAll(joined.String(), "  ", "    "))

	if result := runCLI(t, "diff", base, moved); result.exitCode != 0 || result.stdout != "" {
		t.Fatalf("diff of the same model in another layout = %#v, want no change", result)
	}

	edited := strings.NewReplacer(
		"example      = 5", "example      = 6",
		`title     = "Add Pet"`, `title     = "Register Pet"`,
	).Replace(joined.String()) + "\nstate_change \"list_pets\" {}\n"
	head := writeModel(t, "edited.em.hcl", edited)

	result := runCLI(t, "diff", base, head)

	want := "changed field_type.pet_management.pet_id\n" +
		"changed workflow.add_pet\n" +
		"  changed command.add_pet.add_pet_command\n" +
		"added workflow.list_pets\n"
	if result.exitCode != 0 || result.stdout != want {
		t.Fatalf("diff = %#v, want stdout\n%s", result, want)
	}

	data := runCLI(t, "diff", "--format", "json", base, head)
	var report struct {
		Changes []struct {
			Change, Kind, Address string
			Children              []struct{ Address string }
		}
	}
	if err := json.Unmarshal([]byte(data.stdout), &report); err != nil || data.exitCode != 0 {
		t.Fatalf("diff --format json = %#v, decode error %v", data, err)
	}
	if len(report.Changes) != 3 || report.Changes[1].Children[0].Address != "command.add_pet.add_pet_command" {
		t.Fatalf("json changes = %#v", report.Changes)
	}
	if same := runCLI(t, "diff", "--format", "json", base, moved); !strings.Contains(same.stdout, `"changes": []`) {
		t.Fatalf("diff --format json of equal models = %q, want an empty changes list", same.stdout)
	}
}

func TestRun_DiffRefusesAModelWithCoreErrors(t *testing.T) {
	base := writeFolderModel(t)
	head := writeModel(t, "broken.em.hcl", "state_change \"example\" {\n  command \"submit\" { to = [event.example.missing] }\n}\nboard { workflows = [workflow.example] }\n")

	text := runCLI(t, "diff", base, head)
	if text.exitCode != 1 || text.stdout != "" || !strings.Contains(text.stderr, head+":") {
		t.Fatalf("diff = %#v, want exit 1 and a diagnostic in %s", text, head)
	}
	data := runCLI(t, "diff", "--format", "json", base, head)
	if data.exitCode != 1 || strings.Contains(data.stdout, `"changes"`) || !strings.Contains(data.stdout, `"diagnostics"`) {
		t.Fatalf("diff --format json = %#v, want diagnostics and no changes", data)
	}
}

func TestRun_DiagramTitle(t *testing.T) {
	modelPath := writeModel(t, "valid.em.hcl", "state_change \"example\" {}\nboard { workflows = [workflow.example] }\n")

	titled := runCLI(t, "diagram", "--title", "shop main @ 1a2b3c4", modelPath)
	if titled.exitCode != 0 || !strings.Contains(titled.stdout, `"title":"shop main @ 1a2b3c4"`) {
		t.Fatalf("diagram --title: exit %d, stderr %q, want the title in the view model", titled.exitCode, titled.stderr)
	}
	if plain := runCLI(t, "diagram", modelPath); !strings.Contains(plain.stdout, `"title":"valid"`) {
		t.Fatal("diagram without --title does not use the file name as the title")
	}
	if empty := runCLI(t, "diagram", "--title", "", modelPath); empty.exitCode != 2 || empty.stderr != "title must not be empty\n" {
		t.Fatalf("diagram --title \"\" = %#v, want a usage error", empty)
	}
}

func TestRun_VersionJSONNamesToolAndSpecification(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run("emhcl", "v0.6.0", []string{"version", "--format", "json"}, &stdout, &stderr)

	var versions struct{ Tool, Spec string }
	if err := json.Unmarshal(stdout.Bytes(), &versions); err != nil || exitCode != 0 {
		t.Fatalf("version --format json: exit %d, stdout %q, error %v", exitCode, stdout.String(), err)
	}
	if versions.Tool != "v0.6.0" || versions.Spec != version.Spec {
		t.Fatalf("versions = %#v, want tool v0.6.0 and spec %s", versions, version.Spec)
	}
}

type validateJSON struct {
	Profile       string           `json:"profile"`
	HasErrors     bool             `json:"has_errors"`
	HasCoreErrors bool             `json:"has_core_errors"`
	Diagnostics   []app.Diagnostic `json:"diagnostics"`
}

func decodeValidateReport(t *testing.T, result cliResult) validateJSON {
	t.Helper()
	if result.stderr != "" {
		t.Fatalf("stderr = %q, want empty with --format json", result.stderr)
	}
	var report validateJSON
	if err := json.Unmarshal([]byte(result.stdout), &report); err != nil {
		t.Fatalf("decode %q: %v", result.stdout, err)
	}
	return report
}

func TestRun_RejectsInvalidArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"missing subcommand", nil},
		{"missing model path", []string{"validate"}},
		{"multiple model paths", []string{"validate", "first.em.hcl", "second.em.hcl"}},
		{"unknown subcommand", []string{"format", "model.em.hcl"}},
		{"diagram missing model path", []string{"diagram"}},
		{"diagram missing output value", []string{"diagram", "model.em.hcl", "-o"}},
		{"diagram multiple models", []string{"diagram", "first.em.hcl", "second.em.hcl"}},
		{"serve missing port value", []string{"serve", "model.em.hcl", "--port"}},
		{"serve multiple models", []string{"serve", "first.em.hcl", "second.em.hcl"}},
		{"fmt with -w and --check", []string{"fmt", "-w", "--check", "model.em.hcl"}},
		{"diff with one model", []string{"diff", "model.em.hcl"}},
		{"title for export", []string{"export", "--title", "x", "model.em.hcl"}},
		{"version with an argument", []string{"version", "json"}},
		{"validate with a repeated profile", []string{"validate", "--profile", "strict", "--profile", "valid", "model.em.hcl"}},
		{"validate with a repeated format", []string{"validate", "--format", "json", "--format", "text", "model.em.hcl"}},
		{"diff with a repeated format", []string{"diff", "--format", "json", "--format", "text", "a.em.hcl", "b.em.hcl"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given an invalid command-line invocation.

			// When the command runs.
			result := runCLI(t, test.args...)

			// Then it returns the documented usage error.
			if result.exitCode != 2 {
				t.Fatalf("exit code = %d, want 2; stderr = %q", result.exitCode, result.stderr)
			}
			if result.stdout != "" || !strings.Contains(result.stderr, "usage: emhcl") {
				t.Fatalf("expected usage on stderr only, got %+v", result)
			}
		})
	}
}

func TestRun_ValidateSyntaxErrorIncludesSourceLocation(t *testing.T) {
	// Given a model with an unclosed block.
	modelPath := writeModel(t, "invalid.em.hcl", "bounded_context \"example\" {\n")

	// When the validate command runs.
	result := runCLI(t, "validate", modelPath)

	// Then it returns a source-located error without standard output.
	if result.exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", result.exitCode, result.stderr)
	}
	location := regexp.MustCompile(regexp.QuoteMeta(modelPath) + `:\d+:\d+: Error EM000:`)
	if !location.MatchString(result.stderr) {
		t.Fatalf("stderr = %q, want source-located error", result.stderr)
	}
	if result.stdout != "" {
		t.Fatalf("stdout = %q, want empty", result.stdout)
	}
}

func TestRun_VersionPrintsBuildVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := Run("emhcl", "v0.6.0", []string{"version"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %q", exitCode, stderr.String())
	}
	if got, want := stdout.String(), "emhcl v0.6.0\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

type cliResult struct {
	exitCode int
	stdout   string
	stderr   string
}

func runCLI(t *testing.T, args ...string) cliResult {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	return cliResult{
		exitCode: Run("emhcl", "dev", args, &stdout, &stderr),
		stdout:   stdout.String(),
		stderr:   stderr.String(),
	}
}

func writeModel(t *testing.T, name, model string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(model), 0o600); err != nil {
		t.Fatalf("write model: %v", err)
	}
	return path
}

func modelPaths(t *testing.T, directories ...string) []string {
	t.Helper()
	pattern := filepath.Join(append([]string{"..", ".."}, directories...)...)

	// Get files
	paths, err := filepath.Glob(filepath.Join(pattern, "*.em.hcl"))
	if err != nil {
		t.Fatalf("find models: %v", err)
	}

	// Get folders
	entries, err := os.ReadDir(pattern)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			paths = append(paths, filepath.Join(pattern, entry.Name()))
		}
	}

	if len(paths) == 0 {
		t.Fatalf("no Event Modeling files found at %s", pattern)
	}
	return paths
}
