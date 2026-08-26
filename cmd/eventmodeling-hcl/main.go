// Package main implements the eventmodeling-hcl command-line validator for
// the native HCL v1 Event Modeling Specification.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dclimber/event-modeling-hcl/internal/validator"
	"github.com/hashicorp/hcl/v2"
)

var version = "dev"

const usageMessage = "usage: eventmodeling-hcl validate <model.em.hcl>"

// commandKind names which subcommand a parsed command line requested.
type commandKind uint8

const (
	validateCommand commandKind = iota
	versionCommand
)

// cliCommand is one fully parsed command-line invocation, ready to be
// executed by run.
type cliCommand struct {
	kind commandKind
	path string
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
	if diagnostics := validator.ValidateFile(command.path); diagnostics.HasErrors() {
		writeDiagnostics(stderr, diagnostics)
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
	if len(args) != 2 || args[0] != "validate" {
		return cliCommand{}, errors.New(usageMessage)
	}
	if !strings.HasSuffix(args[1], ".em.hcl") {
		return cliCommand{}, errors.New("model file must use the .em.hcl extension")
	}
	return cliCommand{kind: validateCommand, path: args[1]}, nil
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
	severity := "Warning"
	if diagnostic.Severity == hcl.DiagError {
		severity = "Error"
	}
	message := diagnostic.Summary
	if diagnostic.Detail != "" {
		message += ": " + diagnostic.Detail
	}
	if diagnostic.Subject == nil {
		return fmt.Sprintf("%s: %s\n", severity, message)
	}
	return fmt.Sprintf("%s:%d:%d: %s: %s\n", diagnostic.Subject.Filename, diagnostic.Subject.Start.Line, diagnostic.Subject.Start.Column, severity, message)
}
