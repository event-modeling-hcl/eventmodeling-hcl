package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
)

func TestParseCommand_RecognizesValidateInvocation(t *testing.T) {
	// Given a validate invocation with an Event Modeling model path.
	args := []string{"validate", "model.em.hcl"}

	// When its command is parsed.
	command, err := parseCommand(args)

	// Then it produces a validate command without an error.
	if err != nil {
		t.Fatalf("err = %q, want nil", err)
	}
	if got, want := command, (cliCommand{kind: validateCommand, path: "model.em.hcl"}); got != want {
		t.Fatalf("command = %#v, want %#v", got, want)
	}
}

func TestParseCommand_RejectsUnsupportedExtension(t *testing.T) {
	// Given a validate invocation with a non-Event-Modeling extension.
	args := []string{"validate", "model.hcl"}

	// When its command is parsed.
	_, err := parseCommand(args)

	// Then it produces the extension error.
	if got, want := err, "model file must use the .em.hcl extension"; got == nil || got.Error() != want {
		t.Fatalf("err = %v, want %q", got, want)
	}
}

func TestFormatDiagnostics_FormatsSourceLocatedError(t *testing.T) {
	// Given one source-located validation error.
	diagnostics := hcl.Diagnostics{&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Invalid attribute value",
		Detail:   "title must be a string.",
		Subject:  &hcl.Range{Filename: "model.em.hcl", Start: hcl.Pos{Line: 3, Column: 11}},
	}}

	// When its diagnostics are formatted.
	got := formatDiagnostics(diagnostics)

	// Then it returns the CLI rendering without writing to a stream.
	const want = "model.em.hcl:3:11: Error: Invalid attribute value: title must be a string.\n"
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

func TestRun_ValidateValidModelPrintsSuccess(t *testing.T) {
	// Given a valid Event Modeling model.
	modelPath := writeModel(t, "valid.em.hcl", `slice "example" {
  title      = "Example"
  slice_type = "STATE_CHANGE"
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

func TestRun_ValidateRejectsNonEventModelExtension(t *testing.T) {
	// Given a valid model whose file name does not end in .em.hcl.
	modelPath := writeModel(t, "valid.hcl", `slice "example" {
  title      = "Example"
  slice_type = "STATE_CHANGE"
}
`)

	// When the validate command runs.
	result := runCLI(t, "validate", modelPath)

	// Then it returns the usage error for an unsupported extension.
	if result.exitCode != 2 {
		t.Fatalf("exit code = %d, want 2; stderr = %q", result.exitCode, result.stderr)
	}
	if !strings.Contains(result.stderr, "must use the .em.hcl extension") {
		t.Fatalf("stderr = %q, want extension diagnostic", result.stderr)
	}
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
			if got, want := result.stderr, "usage: eventmodeling-hcl validate <model.em.hcl>\n"; got != want {
				t.Fatalf("stderr = %q, want %q", got, want)
			}
		})
	}
}

func TestRun_ValidateSyntaxErrorIncludesSourceLocation(t *testing.T) {
	// Given a model with an unclosed block.
	modelPath := writeModel(t, "invalid.em.hcl", "slice \"example\" {\n")

	// When the validate command runs.
	result := runCLI(t, "validate", modelPath)

	// Then it returns a source-located error without standard output.
	if result.exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", result.exitCode, result.stderr)
	}
	location := regexp.MustCompile(regexp.QuoteMeta(modelPath) + `:\d+:\d+: Error:`)
	if !location.MatchString(result.stderr) {
		t.Fatalf("stderr = %q, want source-located error", result.stderr)
	}
	if result.stdout != "" {
		t.Fatalf("stdout = %q, want empty", result.stdout)
	}
}

func TestRun_VersionPrintsBuildVersion(t *testing.T) {
	// Given the validator's development build version.

	// When the version command runs.
	result := runCLI(t, "version")

	// Then it prints the exact version without diagnostics.
	if result.exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %q", result.exitCode, result.stderr)
	}
	if got, want := result.stdout, "eventmodeling-hcl dev\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if result.stderr != "" {
		t.Fatalf("stderr = %q, want empty", result.stderr)
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
		exitCode: run(args, &stdout, &stderr),
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
	paths, err := filepath.Glob(filepath.Join(pattern, "*.em.hcl"))
	if err != nil {
		t.Fatalf("find models: %v", err)
	}
	if len(paths) == 0 {
		t.Fatalf("no Event Modeling files found at %s", pattern)
	}
	return paths
}
