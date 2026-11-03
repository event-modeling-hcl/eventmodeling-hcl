// Package cli implements the emhcl command-line adapter.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/app"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/compare"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/serve"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/version"
)

func usageMessage(name string) string {
	return "usage: " + name + " <validate [--profile workshop|valid|strict] [--format text|json] | diagram [--title <text>] | serve | export> <model.em.hcl | folder> | diff [--format text|json] <base> <head> | fmt [-w | --check] <model.em.hcl | folder> | migrate [-w] <model.em.hcl | folder> | import <model.json> | version [--format text|json] [diagram/export/import: -o <file>] [serve: --addr <host> --port <n> --profile <p>]"
}

// defaultServeAddr and defaultServePort are serve's defaults when a
// command line omits --addr / --port: loopback only, so a model isn't
// reachable from the rest of the network unless --addr is passed
// explicitly, on a port unlikely to already be in use by another local dev
// server. Pass --addr 0.0.0.0 (or a specific interface) to share it.
const (
	defaultServeAddr = "127.0.0.1"
	defaultServePort = 8080
)

// commandKind names which subcommand a parsed command line requested.
type commandKind uint8

const (
	validateCommand commandKind = iota
	formatCommand
	diagramCommand
	serveCommand
	exportCommand
	importCommand
	versionCommand
	migrateCommand
	diffCommand
)

// outputFormat selects how validate, diff, and version encode their result:
// text for people (the default) or one JSON object for programs.
type outputFormat uint8

const (
	textFormat outputFormat = iota
	jsonFormat
)

// parseFormat is the one parser of the --format option.
func parseFormat(value string) (outputFormat, error) {
	switch value {
	case "text":
		return textFormat, nil
	case "json":
		return jsonFormat, nil
	}
	return textFormat, errors.New("format must be text or json")
}

// cliCommand is one fully parsed command-line invocation, ready to be
// executed by Run. For diff, path is the base model and head the head model.
type cliCommand struct {
	kind    commandKind
	path    string
	head    string
	profile app.Profile
	format  outputFormat
	write   bool
	check   bool
	title   string
	output  string
	addr    string
	port    int
}

// Run executes one command-line invocation and returns the process exit code:
// 0 for success, 1 when validation finds problems in the model, and 2 when
// the command line itself could not be understood.
func Run(name, toolVersion string, args []string, stdout, stderr io.Writer) int {
	command, err := parseCommand(name, args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	if command.kind == versionCommand {
		return writeVersion(command, name, toolVersion, stdout)
	}
	if command.kind == formatCommand {
		return formatModel(command, stdout, stderr)
	}
	if command.kind == diagramCommand {
		return diagramFile(command, stdout, stderr)
	}
	if command.kind == exportCommand {
		return exportFile(command, stdout, stderr)
	}
	if command.kind == importCommand {
		return importFile(command, stdout, stderr)
	}
	if command.kind == serveCommand {
		return serveFile(command, stderr)
	}
	if command.kind == migrateCommand {
		return migrateModel(command, stdout, stderr)
	}
	if command.kind == diffCommand {
		return diffModels(command, stdout, stderr)
	}
	return validateModel(command, stdout, stderr)
}

// validateReport is the JSON object of `validate --format json`.
type validateReport struct {
	Path          string          `json:"path"`
	Profile       string          `json:"profile"`
	HasErrors     bool            `json:"has_errors"`
	HasCoreErrors bool            `json:"has_core_errors"`
	Diagnostics   app.Diagnostics `json:"diagnostics"`
}

// validateModel validates the model at command.path. Text output prints the
// diagnostics to stderr and "<path> valid" to stdout on success. JSON output
// writes one validateReport to stdout. The exit code is the same for both.
func validateModel(command cliCommand, stdout, stderr io.Writer) int {
	diagnostics := app.ValidatePath(command.path, command.profile)
	exitCode := 0
	if diagnostics.HasErrors() {
		exitCode = 1
	}
	if command.format == jsonFormat {
		writeJSON(stdout, validateReport{
			Path:          command.path,
			Profile:       command.profile.String(),
			HasErrors:     diagnostics.HasErrors(),
			HasCoreErrors: diagnostics.HasCoreErrors(),
			Diagnostics:   diagnostics,
		})
		return exitCode
	}
	if len(diagnostics) > 0 {
		writeDiagnostics(stderr, diagnostics)
	}
	if exitCode == 0 {
		fmt.Fprintf(stdout, "%s valid\n", command.path)
	}
	return exitCode
}

// parseCommand turns raw command-line arguments into a cliCommand, or reports
// an error describing why the arguments are not a valid invocation.
func parseCommand(name string, args []string) (cliCommand, error) {
	usage := usageMessage(name)
	if len(args) >= 1 && args[0] == "version" {
		return versionCLICommand(args[1:], usage)
	}
	if len(args) >= 1 && args[0] == "validate" {
		return validateCLICommand(args[1:], usage)
	}
	if len(args) >= 1 && args[0] == "fmt" {
		return formatCLICommand(args[1:], usage)
	}
	if len(args) >= 1 && args[0] == "diff" {
		return diffCLICommand(args[1:], usage)
	}
	if len(args) == 2 && args[0] == "migrate" {
		return migrateCLICommand(args[1], false)
	}
	if len(args) == 3 && args[0] == "migrate" && args[1] == "-w" {
		return migrateCLICommand(args[2], true)
	}
	if len(args) >= 1 && args[0] == "diagram" {
		return outputCLICommand(diagramCommand, args[1:], usage)
	}
	if len(args) >= 1 && args[0] == "export" {
		return outputCLICommand(exportCommand, args[1:], usage)
	}
	if len(args) >= 1 && args[0] == "import" {
		return outputCLICommand(importCommand, args[1:], usage)
	}
	if len(args) >= 1 && args[0] == "serve" {
		return serveCLICommand(args[1:], usage)
	}
	return cliCommand{}, errors.New(usage)
}

// modelPathError is the usage error for a model path that is neither a
// .em.hcl file nor a folder.
const modelPathError = "model path must be a .em.hcl file or a folder"

// isModelPath reports whether path names a model: a path ending in .em.hcl,
// or an existing folder. Whether the file or folder can actually be read is
// the model loader's concern.
func isModelPath(path string) bool {
	return strings.HasSuffix(path, ".em.hcl") || isDirectory(path)
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// validateCLICommand parses "validate [--profile p] [--format f] <path>",
// with the options in any order. Each option can appear one time.
func validateCLICommand(args []string, usage string) (cliCommand, error) {
	command := cliCommand{kind: validateCommand, profile: app.Valid}
	seen := map[string]bool{}
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--profile":
			if seen["--profile"] || index+1 >= len(args) {
				return cliCommand{}, errors.New(usage)
			}
			seen["--profile"] = true
			profile, ok := app.ParseProfile(args[index+1])
			if !ok {
				return cliCommand{}, errors.New("profile must be workshop, valid, or strict")
			}
			command.profile = profile
			index++
		case "--format":
			if seen["--format"] || index+1 >= len(args) {
				return cliCommand{}, errors.New(usage)
			}
			seen["--format"] = true
			format, err := parseFormat(args[index+1])
			if err != nil {
				return cliCommand{}, err
			}
			command.format = format
			index++
		default:
			if command.path != "" {
				return cliCommand{}, errors.New(usage)
			}
			command.path = args[index]
		}
	}
	if command.path == "" {
		return cliCommand{}, errors.New(usage)
	}
	if !isModelPath(command.path) {
		return cliCommand{}, errors.New(modelPathError)
	}
	return command, nil
}

// formatCLICommand parses "fmt [-w | --check] <path>". -w and --check
// exclude each other.
func formatCLICommand(args []string, usage string) (cliCommand, error) {
	command := cliCommand{kind: formatCommand}
	for _, arg := range args {
		switch arg {
		case "-w", "--check":
			if command.write || command.check {
				return cliCommand{}, errors.New(usage)
			}
			command.write = arg == "-w"
			command.check = arg == "--check"
		default:
			if command.path != "" {
				return cliCommand{}, errors.New(usage)
			}
			command.path = arg
		}
	}
	if command.path == "" {
		return cliCommand{}, errors.New(usage)
	}
	if !isModelPath(command.path) {
		return cliCommand{}, errors.New(modelPathError)
	}
	return command, nil
}

// diffCLICommand parses "diff [--format f] <base> <head>". --format can
// appear one time.
func diffCLICommand(args []string, usage string) (cliCommand, error) {
	command := cliCommand{kind: diffCommand}
	var paths []string
	formatSeen := false
	for index := 0; index < len(args); index++ {
		if args[index] != "--format" {
			paths = append(paths, args[index])
			continue
		}
		if formatSeen || index+1 >= len(args) {
			return cliCommand{}, errors.New(usage)
		}
		formatSeen = true
		format, err := parseFormat(args[index+1])
		if err != nil {
			return cliCommand{}, err
		}
		command.format = format
		index++
	}
	if len(paths) != 2 {
		return cliCommand{}, errors.New(usage)
	}
	for _, path := range paths {
		if !isModelPath(path) {
			return cliCommand{}, errors.New(modelPathError)
		}
	}
	command.path, command.head = paths[0], paths[1]
	return command, nil
}

// versionCLICommand parses "version [--format f]".
func versionCLICommand(args []string, usage string) (cliCommand, error) {
	command := cliCommand{kind: versionCommand}
	if len(args) == 0 {
		return command, nil
	}
	if len(args) != 2 || args[0] != "--format" {
		return cliCommand{}, errors.New(usage)
	}
	format, err := parseFormat(args[1])
	if err != nil {
		return cliCommand{}, err
	}
	command.format = format
	return command, nil
}

func migrateCLICommand(path string, write bool) (cliCommand, error) {
	if !isModelPath(path) {
		return cliCommand{}, errors.New(modelPathError)
	}
	return cliCommand{kind: migrateCommand, path: path, write: write}, nil
}

// outputCLICommand parses "<path> [-o <file>]" in either order for the
// commands that read one input and write one output. The input is a model
// path for diagram and export, and a JSON file for import. diagram also
// accepts "--title <text>".
func outputCLICommand(kind commandKind, args []string, usage string) (cliCommand, error) {
	var path string
	var output string
	var title string
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "-o", "--output":
			if output != "" || index+1 >= len(args) {
				return cliCommand{}, errors.New(usage)
			}
			output = args[index+1]
			index++
		case "--title":
			if kind != diagramCommand || title != "" || index+1 >= len(args) {
				return cliCommand{}, errors.New(usage)
			}
			if args[index+1] == "" {
				return cliCommand{}, errors.New("title must not be empty")
			}
			title = args[index+1]
			index++
		default:
			if path != "" {
				return cliCommand{}, errors.New(usage)
			}
			path = args[index]
		}
	}
	if path == "" {
		return cliCommand{}, errors.New(usage)
	}
	if kind == importCommand {
		if !strings.HasSuffix(path, ".json") {
			return cliCommand{}, errors.New("event model JSON file must use the .json extension")
		}
	} else if !isModelPath(path) {
		return cliCommand{}, errors.New(modelPathError)
	}
	if kind == importCommand && output != "" && !strings.HasSuffix(output, ".em.hcl") {
		return cliCommand{}, errors.New("import output must use the .em.hcl extension")
	}
	if kind == exportCommand && output != "" {
		inputPath, inputErr := filepath.Abs(path)
		outputPath, outputErr := filepath.Abs(output)
		samePath := inputErr == nil && outputErr == nil && inputPath == outputPath
		inputInfo, inputStatErr := os.Stat(path)
		outputInfo, outputStatErr := os.Stat(output)
		sameFile := inputStatErr == nil && outputStatErr == nil && os.SameFile(inputInfo, outputInfo)
		if samePath || sameFile || strings.HasSuffix(output, ".em.hcl") || (isDirectory(path) && overwritesFolderMember(path, output)) {
			return cliCommand{}, errors.New("export output must not overwrite the source or use the .em.hcl extension; use -o <model.json> or omit -o to write JSON to stdout")
		}
	}
	return cliCommand{kind: kind, path: path, output: output, title: title}, nil
}

// overwritesFolderMember reports whether writing output would replace a member
// file of the folder model at folder. It matches a member by cleaned absolute
// path, and by file identity when output exists, which also catches symlinks
// and hard links in either direction because os.Stat follows symlinks. When the
// members cannot be listed it reports false and leaves the read error to the
// export itself.
func overwritesFolderMember(folder, output string) bool {
	files, diagnostics := app.ReadModel(folder)
	if diagnostics.HasErrors() {
		return false
	}
	outputPath, outputErr := filepath.Abs(output)
	outputInfo, outputStatErr := os.Stat(output)
	for _, file := range files {
		if memberPath, err := filepath.Abs(file.Name); err == nil && outputErr == nil && memberPath == outputPath {
			return true
		}
		if outputStatErr != nil {
			continue
		}
		if memberInfo, err := os.Stat(file.Name); err == nil && os.SameFile(memberInfo, outputInfo) {
			return true
		}
	}
	return false
}

// serveCLICommand parses the arguments following "serve": exactly one
// model path, plus any of --addr, --port, and --profile in any order.
func serveCLICommand(args []string, usage string) (cliCommand, error) {
	path := ""
	addr := defaultServeAddr
	port := defaultServePort
	profile := app.Valid

	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--addr":
			if index+1 >= len(args) {
				return cliCommand{}, errors.New(usage)
			}
			addr = args[index+1]
			index++
		case "--port":
			if index+1 >= len(args) {
				return cliCommand{}, errors.New(usage)
			}
			parsedPort, err := strconv.Atoi(args[index+1])
			if err != nil {
				return cliCommand{}, errors.New("port must be a number")
			}
			if parsedPort < 0 || parsedPort > 65535 {
				return cliCommand{}, errors.New("port must be between 0 and 65535")
			}
			port = parsedPort
			index++
		case "--profile":
			if index+1 >= len(args) {
				return cliCommand{}, errors.New(usage)
			}
			parsedProfile, ok := app.ParseProfile(args[index+1])
			if !ok {
				return cliCommand{}, errors.New("profile must be workshop, valid, or strict")
			}
			profile = parsedProfile
			index++
		default:
			if path != "" {
				return cliCommand{}, errors.New(usage)
			}
			path = args[index]
		}
	}
	if path == "" {
		return cliCommand{}, errors.New(usage)
	}
	if !isModelPath(path) {
		return cliCommand{}, errors.New(modelPathError)
	}
	return cliCommand{kind: serveCommand, path: path, profile: profile, addr: addr, port: port}, nil
}

// formatModel formats the member files of the model at command.path, a file
// or a folder. Without a flag it prints each file, and a folder prints a
// "# ==> <file> <==" line before each member. With -w it rewrites only the
// files whose layout changes. With --check it writes no file, prints the name
// of each member whose layout differs, and exits 1 when at least one differs.
// A syntax error in any member changes no file.
func formatModel(command cliCommand, stdout, stderr io.Writer) int {
	result := app.FormatPath(command.path)
	if len(result.Diagnostics) > 0 {
		writeDiagnostics(stderr, result.Diagnostics)
	}
	if result.Diagnostics.HasErrors() {
		return 1
	}
	if command.check {
		exitCode := 0
		for _, file := range result.Files {
			if file.Changed {
				fmt.Fprintln(stdout, file.Name)
				exitCode = 1
			}
		}
		return exitCode
	}
	files := make([]syntax.File, len(result.Files))
	for index, file := range result.Files {
		files[index] = file.File
	}
	if command.write {
		return rewriteFiles(files, stderr)
	}
	printFiles(isDirectory(command.path), files, stdout)
	return 0
}

// printFiles writes each file to stdout. A folder model gets a
// "# ==> <file> <==" line before each member.
func printFiles(folder bool, files []syntax.File, stdout io.Writer) {
	for _, file := range files {
		if folder {
			fmt.Fprintf(stdout, "# ==> %s <==\n", file.Name)
		}
		_, _ = stdout.Write(file.Source)
	}
}

// rewriteFiles overwrites each file whose content differs from the file on
// disk, and keeps its permissions.
func rewriteFiles(files []syntax.File, stderr io.Writer) int {
	for _, file := range files {
		current, err := os.ReadFile(file.Name)
		if err == nil && string(current) == string(file.Source) {
			continue
		}
		info, err := os.Stat(file.Name)
		if err != nil {
			fmt.Fprintf(stderr, "failed to stat %s: %v\n", file.Name, err)
			return 1
		}
		if err := os.WriteFile(file.Name, file.Source, info.Mode().Perm()); err != nil {
			fmt.Fprintf(stderr, "failed to write %s: %v\n", file.Name, err)
			return 1
		}
	}
	return 0
}

// migrateModel rewrites the v0.4.0 model at command.path for the current
// language. Report lines go to stderr. Without -w the migrated source goes to
// stdout, each folder member preceded by a "# ==> <path> <==" line; with -w
// only members whose content changed are overwritten. Any error writes
// nothing.
func migrateModel(command cliCommand, stdout, stderr io.Writer) int {
	result, diagnostics := app.MigratePath(command.path)
	if len(diagnostics) > 0 {
		writeDiagnostics(stderr, diagnostics)
	}
	if diagnostics.HasErrors() {
		return 1
	}
	for _, line := range result.Report {
		fmt.Fprintln(stderr, line)
	}
	if !command.write {
		printFiles(isDirectory(command.path), result.Files, stdout)
		return 0
	}
	return rewriteFiles(result.Files, stderr)
}

// serveFile runs a local live-reload server for command.path until it is
// interrupted (Ctrl-C) or fails to start. All behavior — rendering,
// file-watching, HTTP, browser-opening — is internal/serve's; this is
// deliberately thin, blocking glue with nothing of its own worth unit
// testing (internal/serve's own test suite already covers every part of it
// that is testable without actually binding a live socket).
func serveFile(command cliCommand, stderr io.Writer) int {
	if err := serve.Start(command.path, command.addr, command.port, command.profile); err != nil {
		fmt.Fprintf(stderr, "failed to serve %s: %v\n", command.path, err)
		return 1
	}
	return 0
}

// diagramFile writes the diagram unless the model has core errors. A policy
// rejection still writes it, and the exit status reports the rejection.
// --title replaces the default title, which the renderer derives from the path.
func diagramFile(command cliCommand, stdout, stderr io.Writer) int {
	var result app.RenderResult
	if command.title != "" {
		result = app.RenderPathTitled(command.path, command.title, app.Valid)
	} else {
		result = app.RenderPath(command.path, app.Valid)
	}
	if len(result.Diagnostics) > 0 {
		writeDiagnostics(stderr, result.Diagnostics)
	}
	if result.Diagnostics.HasCoreErrors() {
		return 1
	}
	if code := writeOutput(command.output, result.HTML, stdout, stderr); code != 0 {
		return code
	}
	if result.Diagnostics.HasErrors() {
		return 1
	}
	return 0
}

// diffReport is the JSON object of `diff --format json`. It has changes when
// both models load, and diagnostics when a model has a core error.
type diffReport struct {
	Base        string            `json:"base"`
	Head        string            `json:"head"`
	Changes     *[]compare.Change `json:"changes,omitempty"`
	Diagnostics app.Diagnostics   `json:"diagnostics,omitempty"`
}

// diffModels compares the base and head models. It exits 0 when both load,
// also when they differ, and 1 when a model has a core error.
func diffModels(command cliCommand, stdout, stderr io.Writer) int {
	result := app.DiffPaths(command.path, command.head)
	refused := result.Changes == nil
	if command.format == jsonFormat {
		report := diffReport{Base: command.path, Head: command.head, Diagnostics: result.Diagnostics}
		if !refused {
			report.Changes = &result.Changes
		}
		writeJSON(stdout, report)
	} else if refused {
		writeDiagnostics(stderr, result.Diagnostics)
	} else {
		writeChanges(stdout, result.Changes, "")
	}
	if refused {
		return 1
	}
	return 0
}

// writeChanges writes one line for each change, "<change> <label>", with the
// children of a workflow indented by two spaces.
func writeChanges(w io.Writer, changes []compare.Change, indent string) {
	for _, change := range changes {
		fmt.Fprintf(w, "%s%s %s\n", indent, change.Change, change.Label())
		writeChanges(w, change.Children, indent+"  ")
	}
}

// writeVersion prints the tool version. JSON output adds the version of the
// specification that the tool implements.
func writeVersion(command cliCommand, name, toolVersion string, stdout io.Writer) int {
	if command.format == jsonFormat {
		writeJSON(stdout, struct {
			Tool string `json:"tool"`
			Spec string `json:"spec"`
		}{Tool: toolVersion, Spec: version.Spec})
		return 0
	}
	fmt.Fprintf(stdout, "%s %s\n", name, toolVersion)
	return 0
}

// writeJSON writes value as one indented JSON object and a newline. It does
// not escape <, >, and &, so diagnostic text stays readable.
func writeJSON(w io.Writer, value any) {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(value)
}

// exportFile writes the JSON unless the model has core errors. A policy
// rejection still writes it, and the exit status reports the rejection.
func exportFile(command cliCommand, stdout, stderr io.Writer) int {
	result := app.ExportPath(command.path)
	if len(result.Diagnostics) > 0 {
		writeDiagnostics(stderr, result.Diagnostics)
	}
	if result.Diagnostics.HasCoreErrors() {
		return 1
	}
	writeWarnings(stderr, result.Warnings)
	if code := writeOutput(command.output, result.JSON, stdout, stderr); code != 0 {
		return code
	}
	if result.Diagnostics.HasErrors() {
		return 1
	}
	return 0
}

// importFile writes the converted model even when validating it reports
// errors, so the generated source can be repaired by hand; the exit code
// still signals the failure.
func importFile(command cliCommand, stdout, stderr io.Writer) int {
	outputName := command.output
	if outputName == "" {
		outputName = "<stdout>"
	}
	result := app.ImportFile(command.path, outputName)
	writeWarnings(stderr, result.Warnings)
	if len(result.Diagnostics) > 0 {
		writeDiagnostics(stderr, result.Diagnostics)
	}
	if result.Source == "" && result.Diagnostics.HasErrors() {
		return 1
	}
	if code := writeOutput(command.output, result.Source, stdout, stderr); code != 0 {
		return code
	}
	if result.Diagnostics.HasErrors() {
		return 1
	}
	return 0
}

func writeWarnings(w io.Writer, warnings []string) {
	for _, warning := range warnings {
		fmt.Fprintf(w, "Warning: %s\n", warning)
	}
}

// writeOutput writes content to path, or to stdout when path is empty.
func writeOutput(path, content string, stdout, stderr io.Writer) int {
	if path == "" {
		_, _ = io.WriteString(stdout, content)
		return 0
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fmt.Fprintf(stderr, "failed to write %s: %v\n", path, err)
		return 1
	}
	fmt.Fprintf(stderr, "wrote %s\n", path)
	return 0
}

// writeDiagnostics renders diagnostics in the CLI's text format and writes
// the result to w.
func writeDiagnostics(w io.Writer, diagnostics []app.Diagnostic) {
	_, _ = io.WriteString(w, formatDiagnostics(diagnostics))
}

// formatDiagnostics renders every diagnostic with formatDiagnostic and
// concatenates the results in order.
func formatDiagnostics(diagnostics []app.Diagnostic) string {
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
// form "file:line:column: Severity EMxxx: Summary: Detail\n". Diagnostics
// that have no source location (for example, a failure to even read the
// model file, which happens before any source text exists to point at) omit
// the "file:line:column:" prefix — recognized here by an empty Filename,
// which app.Diagnostic only ever leaves unset when the underlying
// hcl.Diagnostic had no Subject.
func formatDiagnostic(diagnostic app.Diagnostic) string {
	label := strings.TrimSpace(diagnostic.Severity + " " + diagnostic.Code)
	message := fmt.Sprintf("%s: %s", label, diagnostic.Summary)
	if diagnostic.Detail != "" {
		message += ": " + diagnostic.Detail
	}
	if diagnostic.Filename == "" {
		return fmt.Sprintf("%s\n", message)
	}
	return fmt.Sprintf("%s:%d:%d: %s\n", diagnostic.Filename, diagnostic.Line, diagnostic.Column, message)
}
