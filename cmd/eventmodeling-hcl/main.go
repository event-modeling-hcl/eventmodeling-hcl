// Package main implements the eventmodeling-hcl command-line validator for
// the native HCL Event Modeling Specification.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/formatter"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/model"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/renderer"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/validator"
	"github.com/hashicorp/hcl/v2"
)

var version = "dev"

const usageMessage = "usage: eventmodeling-hcl <validate [--profile workshop|valid|strict] | fmt [-w] | diagram> <model.em.hcl> [diagram: -o <file>]"

// commandKind names which subcommand a parsed command line requested.
type commandKind uint8

const (
	validateCommand commandKind = iota
	formatCommand
	diagramCommand
	versionCommand
)

// cliCommand is one fully parsed command-line invocation, ready to be
// executed by run.
type cliCommand struct {
	kind    commandKind
	path    string
	profile validator.Profile
	write   bool
	output  string
}

// main runs the CLI against the process's real arguments and standard
// streams, and exits with the resulting status code.
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes one command-line invocation and returns the process exit
// code: 0 for success, 1 when validation finds problems in the model, and
// 2 when the command line itself could not be understood.
func run(args []string, stdout, stderr io.Writer) int {
	command, err := parseCommand(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	if command.kind == versionCommand {
		fmt.Fprintf(stdout, "eventmodeling-hcl %s\n", version)
		return 0
	}
	if command.kind == formatCommand {
		return formatFile(command, stdout, stderr)
	}
	if command.kind == diagramCommand {
		return diagramFile(command, stdout, stderr)
	}
	diagnostics := validator.ValidateFileWithProfile(command.path, command.profile)
	if len(diagnostics) > 0 {
		writeDiagnostics(stderr, diagnostics)
	}
	if diagnostics.HasErrors() {
		return 1
	}

	fmt.Fprintf(stdout, "%s valid\n", command.path)
	return 0
}

// parseCommand turns raw command-line arguments into a cliCommand, or
// reports an error describing why the arguments are not a valid
// invocation.
func parseCommand(args []string) (cliCommand, error) {
	if len(args) == 1 && args[0] == "version" {
		return cliCommand{kind: versionCommand}, nil
	}
	if len(args) == 2 && args[0] == "validate" {
		return validateCLICommand(args[1], validator.Valid)
	}
	if len(args) == 4 && args[0] == "validate" && args[1] == "--profile" {
		profile, ok := validator.ParseProfile(args[2])
		if !ok {
			return cliCommand{}, errors.New("profile must be workshop, valid, or strict")
		}
		return validateCLICommand(args[3], profile)
	}
	if len(args) == 2 && args[0] == "fmt" {
		return formatCLICommand(args[1], false)
	}
	if len(args) == 3 && args[0] == "fmt" && args[1] == "-w" {
		return formatCLICommand(args[2], true)
	}
	if len(args) >= 1 && args[0] == "diagram" {
		return diagramCLICommand(args[1:])
	}
	return cliCommand{}, errors.New(usageMessage)
}

func validateCLICommand(path string, profile validator.Profile) (cliCommand, error) {
	if !strings.HasSuffix(path, ".em.hcl") {
		return cliCommand{}, errors.New("model file must use the .em.hcl extension")
	}
	return cliCommand{kind: validateCommand, path: path, profile: profile}, nil
}

func formatCLICommand(path string, write bool) (cliCommand, error) {
	if !strings.HasSuffix(path, ".em.hcl") {
		return cliCommand{}, errors.New("model file must use the .em.hcl extension")
	}
	return cliCommand{kind: formatCommand, path: path, write: write}, nil
}

func diagramCLICommand(args []string) (cliCommand, error) {
	var path string
	var output string
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "-o", "--output":
			if output != "" || index+1 >= len(args) {
				return cliCommand{}, errors.New(usageMessage)
			}
			output = args[index+1]
			index++
		default:
			if path != "" {
				return cliCommand{}, errors.New(usageMessage)
			}
			path = args[index]
		}
	}
	if path == "" {
		return cliCommand{}, errors.New(usageMessage)
	}
	if !strings.HasSuffix(path, ".em.hcl") {
		return cliCommand{}, errors.New("model file must use the .em.hcl extension")
	}
	return cliCommand{kind: diagramCommand, path: path, output: output}, nil
}

func formatFile(command cliCommand, stdout, stderr io.Writer) int {
	source, err := os.ReadFile(command.path)
	if err != nil {
		fmt.Fprintf(stderr, "failed to read %s: %v\n", command.path, err)
		return 1
	}
	formatted, diagnostics := formatter.Format(command.path, source)
	if len(diagnostics) > 0 {
		writeDiagnostics(stderr, diagnostics)
	}
	if diagnostics.HasErrors() {
		return 1
	}
	if !command.write {
		_, _ = stdout.Write(formatted)
		return 0
	}
	info, err := os.Stat(command.path)
	if err != nil {
		fmt.Fprintf(stderr, "failed to stat %s: %v\n", command.path, err)
		return 1
	}
	if err := os.WriteFile(command.path, formatted, info.Mode().Perm()); err != nil {
		fmt.Fprintf(stderr, "failed to write %s: %v\n", command.path, err)
		return 1
	}
	return 0
}

func diagramFile(command cliCommand, stdout, stderr io.Writer) int {
	source, err := os.ReadFile(command.path)
	if err != nil {
		diagnostics := hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Extra:    validator.DiagnosticCodeExtra("EM001"),
			Summary:  "Failed to read file",
			Detail:   fmt.Sprintf("The configuration file %q could not be read.", command.path),
		}}
		writeDiagnostics(stderr, diagnostics)
		return 1
	}
	loaded, diagnostics := model.Load(command.path, source, model.Valid)
	if len(diagnostics) > 0 {
		writeDiagnostics(stderr, diagnostics)
	}
	if diagnostics.HasErrors() {
		return 1
	}
	html, err := renderer.Render(command.path, loaded)
	if err != nil {
		fmt.Fprintf(stderr, "failed to render %s: %v\n", command.path, err)
		return 1
	}
	if command.output == "" {
		_, _ = io.WriteString(stdout, html)
		return 0
	}
	if err := os.WriteFile(command.output, []byte(html), 0o644); err != nil {
		fmt.Fprintf(stderr, "failed to write %s: %v\n", command.output, err)
		return 1
	}
	fmt.Fprintf(stderr, "wrote %s\n", command.output)
	return 0
}

// writeDiagnostics renders diagnostics in the CLI's text format and writes
// the result to w.
func writeDiagnostics(w io.Writer, diagnostics hcl.Diagnostics) {
	_, _ = io.WriteString(w, formatDiagnostics(diagnostics))
}

// formatDiagnostics renders every diagnostic with formatDiagnostic and
// concatenates the results in order.
func formatDiagnostics(diagnostics hcl.Diagnostics) string {
	return strings.Join(mapSlice(diagnostics, formatDiagnostic), "")
}

// mapSlice applies transform to each element of items, in order, and
// returns the resulting slice.
func mapSlice[T, U any](items []T, transform func(T) U) []U {
	result := make([]U, len(items))
	for index, item := range items {
		result[index] = transform(item)
	}
	return result
}

// formatDiagnostic renders one diagnostic as a single line of text, in the
// form "file:line:column: Severity: Summary: Detail\n". Diagnostics that
// have no source location (for example, a failure to even read the model
// file, which happens before any source text exists to point at) omit the
// "file:line:column:" prefix.
func formatDiagnostic(diagnostic *hcl.Diagnostic) string {
	severity := ""
	switch diagnostic.Severity {
	case hcl.DiagInvalid:
		severity = "Info"
	case hcl.DiagError:
		severity = "Error"
	case hcl.DiagWarning:
		severity = "Warning"
	}
	message := fmt.Sprintf("%s %s: %s", severity, validator.DiagnosticCode(diagnostic), diagnostic.Summary)
	if diagnostic.Detail != "" {
		message += ": " + diagnostic.Detail
	}
	if diagnostic.Subject == nil {
		return fmt.Sprintf("%s\n", message)
	}
	return fmt.Sprintf("%s:%d:%d: %s\n", diagnostic.Subject.Filename, diagnostic.Subject.Start.Line, diagnostic.Subject.Start.Column, message)
}
