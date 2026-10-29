// Package migrate rewrites a language v0.4.0 Event Modeling model as a v0.5.0
// model (RFC 0004 §3). It is the only place that still reads v0.4.0 rules:
// the effective board order, typeless field inference, derived titles, and the
// translation keyword. It works on raw HCL so it does not depend on the v0.5.0
// validator or model.
package migrate

import (
	"bytes"
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// Result is a migrated model: every member file under its original name, in
// the input order, plus the human-readable report. Edited files keep their
// comments and get whitespace-only formatting; other files are unchanged.
type Result struct {
	Files  []syntax.File
	Report []string
}

// workflowKinds are the v0.4.0 workflow block types.
var workflowKinds = []string{"state_change", "state_view", "automation", "translation"}

// elementKinds are the workflow children that v0.4.0 gave an effective title.
var elementKinds = []string{"command", "readmodel", "screen", "processor", "screen_image", "table"}

// member is one parsed input file.
type member struct {
	name   string
	source []byte
	body   *hclsyntax.Body
}

type workflow struct {
	id    string
	kind  string
	block *hclsyntax.Block
	file  string
}

// edits are the rewrites planned on the original syntax tree, keyed by block.
type edits struct {
	titles  map[*hclsyntax.Block]string
	types   map[*hclsyntax.Block]hcl.Traversal
	renames map[*hclsyntax.Block]bool
}

// index holds the document-wide v0.4.0 facts the rewrite needs.
type index struct {
	external   map[string]bool
	fieldTypes map[string][]string
	workflows  []workflow
	chapters   [][]string
}

// Migrate reads files as one v0.4.0 model and returns them rewritten for
// v0.5.0. Any error diagnostic stops the migration and returns an empty
// Result.
func Migrate(files []syntax.File) (Result, hcl.Diagnostics) {
	members, diagnostics := parse(files)
	if diagnostics.HasErrors() {
		return Result{}, diagnostics
	}
	ordered := slices.Clone(members)
	slices.SortStableFunc(ordered, func(left, right member) int { return cmp.Compare(left.name, right.name) })

	idx, indexDiagnostics := buildIndex(ordered)
	diagnostics = append(diagnostics, indexDiagnostics...)
	diagnostics = append(diagnostics, idx.checkChapters(ordered)...)
	plan := edits{titles: map[*hclsyntax.Block]string{}, types: map[*hclsyntax.Block]hcl.Traversal{}, renames: map[*hclsyntax.Block]bool{}}
	for _, file := range ordered {
		diagnostics = append(diagnostics, idx.planFields(file.body, &plan)...)
	}
	diagnostics = append(diagnostics, idx.checkExternality()...)
	if diagnostics.HasErrors() {
		return Result{}, diagnostics
	}
	for _, file := range ordered {
		planTitles(file.body, &plan)
	}
	var translations []string
	for _, flow := range idx.workflows {
		if flow.kind == "translation" {
			plan.renames[flow.block] = true
			translations = append(translations, fmt.Sprintf("translation: %s:%d: workflow %q rewritten as automation", flow.file, flow.block.DefRange().Start.Line, flow.id))
		}
	}

	board := idx.boardOrder(len(members))
	boardFile := ordered[0].name
	result := Result{Files: make([]syntax.File, 0, len(members))}
	for _, file := range members {
		var boardBlock []byte
		if file.name == boardFile {
			boardBlock = boardSource(board)
		}
		output, outputDiagnostics := rewrite(file, plan, boardBlock)
		diagnostics = append(diagnostics, outputDiagnostics...)
		if outputDiagnostics.HasErrors() {
			return Result{}, diagnostics
		}
		result.Files = append(result.Files, syntax.File{Name: file.name, Source: output})
	}
	result.Report = append([]string{
		fmt.Sprintf("board: inserted at the start of %s with %d workflows", boardFile, len(board)),
		fmt.Sprintf("types: wrote %d explicit field types", len(plan.types)),
		fmt.Sprintf("titles: wrote %d explicit titles", len(plan.titles)),
	}, translations...)
	return result, diagnostics
}

func parse(files []syntax.File) ([]member, hcl.Diagnostics) {
	if len(files) == 0 {
		return nil, hcl.Diagnostics{{Severity: hcl.DiagError, Summary: "Empty model", Detail: "migrate needs at least one member file."}}
	}
	var diagnostics hcl.Diagnostics
	members := make([]member, 0, len(files))
	for _, file := range files {
		parsed, fileDiagnostics := hclsyntax.ParseConfig(file.Source, file.Name, hcl.Pos{Line: 1, Column: 1})
		diagnostics = append(diagnostics, fileDiagnostics...)
		if fileDiagnostics.HasErrors() {
			continue
		}
		members = append(members, member{name: file.Name, source: file.Source, body: parsed.Body.(*hclsyntax.Body)})
	}
	return members, diagnostics
}

func buildIndex(ordered []member) (*index, hcl.Diagnostics) {
	idx := &index{external: map[string]bool{}, fieldTypes: map[string][]string{}}
	var diagnostics hcl.Diagnostics
	for _, file := range ordered {
		for _, block := range file.body.Blocks {
			if block.Type == "board" {
				diagnostics = append(diagnostics, errorAt(block.DefRange(), "Existing board block", "the model already has a board block; migrate reads only v0.4.0 models, which have none."))
				continue
			}
			if len(block.Labels) != 1 {
				continue
			}
			id := block.Labels[0]
			switch {
			case block.Type == "bounded_context":
				idx.external[id] = idx.external[id] || literalTrue(block.Body.Attributes["external"])
				for _, child := range block.Body.Blocks {
					if child.Type == "field_type" && len(child.Labels) == 1 && !slices.Contains(idx.fieldTypes[child.Labels[0]], id) {
						idx.fieldTypes[child.Labels[0]] = append(idx.fieldTypes[child.Labels[0]], id)
					}
				}
			case block.Type == "chapter":
				idx.chapters = append(idx.chapters, workflowReferences(block.Body.Attributes["workflows"]))
			case slices.Contains(workflowKinds, block.Type):
				idx.workflows = append(idx.workflows, workflow{id: id, kind: block.Type, block: block, file: file.name})
			}
		}
	}
	return idx, diagnostics
}

// checkChapters preserves the v0.4.0 acceptance boundary. The new validator
// cannot enforce EM006/EM013 after rewriting because v0.5.0 retired them.
func (idx *index) checkChapters(files []member) hcl.Diagnostics {
	positions := make(map[string]int, len(idx.workflows))
	for position, flow := range idx.workflows {
		positions[flow.id] = position
	}
	var diagnostics hcl.Diagnostics
	chapterFile := ""
	for _, file := range files {
		for _, block := range file.body.Blocks {
			if block.Type != "chapter" {
				continue
			}
			if len(files) > 1 {
				if chapterFile == "" {
					chapterFile = file.name
				} else if chapterFile != file.name {
					diagnostics = append(diagnostics, errorAt(block.DefRange(), "Chapters in several files", "v0.4.0 requires all chapters in one member file (EM013)."))
				}
				continue
			}
			previous := -1
			attribute := block.Body.Attributes["workflows"]
			for _, id := range workflowReferences(attribute) {
				position, exists := positions[id]
				if !exists {
					continue // The v0.5.0 reference check reports it after rewriting.
				}
				if previous >= 0 && position != previous+1 {
					diagnostics = append(diagnostics, errorAt(attribute.Expr.Range(), "Invalid chapter range", "v0.4.0 requires chapter workflows to be contiguous and listed in source order (EM006)."))
					break
				}
				previous = position
			}
		}
	}
	return diagnostics
}

// planFields records the explicit type of every typeless field and subfield.
// A field inside a bounded_context resolves to that context's field_type of
// the same name; any other field resolves to the unique field_type of that
// name in the whole model.
func (idx *index) planFields(body *hclsyntax.Body, plan *edits) hcl.Diagnostics {
	var diagnostics hcl.Diagnostics
	for _, block := range body.Blocks {
		if len(block.Labels) != 1 {
			continue
		}
		switch {
		case block.Type == "bounded_context":
			diagnostics = append(diagnostics, idx.planNestedFields(block.Body, block.Labels[0], plan)...)
		case slices.Contains(workflowKinds, block.Type):
			diagnostics = append(diagnostics, idx.planNestedFields(block.Body, "", plan)...)
		}
	}
	return diagnostics
}

func (idx *index) planNestedFields(body *hclsyntax.Body, contextID string, plan *edits) hcl.Diagnostics {
	var diagnostics hcl.Diagnostics
	for _, block := range body.Blocks {
		if (block.Type == "field" || block.Type == "subfield") && len(block.Labels) == 1 && block.Body.Attributes["type"] == nil {
			owner, diagnostic := idx.resolveField(block, contextID)
			if diagnostic != nil {
				diagnostics = append(diagnostics, diagnostic)
			} else {
				plan.types[block] = hcl.Traversal{hcl.TraverseRoot{Name: "field_type"}, hcl.TraverseAttr{Name: owner}, hcl.TraverseAttr{Name: block.Labels[0]}}
			}
		}
		diagnostics = append(diagnostics, idx.planNestedFields(block.Body, contextID, plan)...)
	}
	return diagnostics
}

func (idx *index) resolveField(block *hclsyntax.Block, contextID string) (string, *hcl.Diagnostic) {
	name := block.Labels[0]
	contexts := idx.fieldTypes[name]
	if contextID != "" {
		if !slices.Contains(contexts, contextID) {
			return "", errorAt(block.DefRange(), "Unresolved field type", fmt.Sprintf("field %q has no type and bounded_context %q declares no field_type %q.", name, contextID, name))
		}
		return contextID, nil
	}
	switch len(contexts) {
	case 1:
		return contexts[0], nil
	case 0:
		return "", errorAt(block.DefRange(), "Unresolved field type", fmt.Sprintf("field %q has no type and no field_type %q is declared in any bounded_context.", name, name))
	default:
		return "", errorAt(block.DefRange(), "Ambiguous field type", fmt.Sprintf("field %q has no type and field_type %q is declared in multiple bounded_contexts (%s); add an explicit type.", name, name, strings.Join(contexts, ", ")))
	}
}

// checkExternality applies the v0.4.0 EM011/EM012 rules: a translation must
// consume an external event and an automation must not.
func (idx *index) checkExternality() hcl.Diagnostics {
	var diagnostics hcl.Diagnostics
	for _, flow := range idx.workflows {
		if flow.kind != "translation" && flow.kind != "automation" {
			continue
		}
		external := idx.consumesExternal(flow.block)
		if flow.kind == "translation" && !external {
			diagnostics = append(diagnostics, errorAt(flow.block.DefRange(), "Invalid translation", fmt.Sprintf("translation %q consumes no event from an external bounded_context; v0.4.0 rejects it (EM011).", flow.id)))
		}
		if flow.kind == "automation" && external {
			diagnostics = append(diagnostics, errorAt(flow.block.DefRange(), "Invalid automation", fmt.Sprintf("automation %q consumes an event from an external bounded_context; v0.4.0 rejects it (EM012).", flow.id)))
		}
	}
	return diagnostics
}

func (idx *index) consumesExternal(block *hclsyntax.Block) bool {
	for _, child := range block.Body.Blocks {
		for _, parts := range traversalList(child.Body.Attributes["from"]) {
			if len(parts) == 3 && parts[0] == "event" && idx.external[parts[1]] {
				return true
			}
		}
	}
	return false
}

// boardOrder returns the v0.4.0 effective workflow order. A one-file model
// uses source order. A model of several files uses chapter order, then each
// chapter's list order, then every remaining workflow in model order.
func (idx *index) boardOrder(fileCount int) []string {
	known := map[string]bool{}
	for _, flow := range idx.workflows {
		known[flow.id] = true
	}
	placed := map[string]bool{}
	var order []string
	place := func(id string) {
		if known[id] && !placed[id] {
			placed[id] = true
			order = append(order, id)
		}
	}
	if fileCount > 1 {
		for _, chapter := range idx.chapters {
			for _, id := range chapter {
				place(id)
			}
		}
	}
	for _, flow := range idx.workflows {
		place(flow.id)
	}
	return order
}

// planTitles records the v0.4.0 effective title of every construct that had
// one and declares none.
func planTitles(body *hclsyntax.Body, plan *edits) {
	for _, block := range body.Blocks {
		switch {
		case slices.Contains([]string{"actor", "team", "system", "chapter"}, block.Type):
			planTitle(block, plan)
		case block.Type == "bounded_context":
			planTitle(block, plan)
			for _, child := range block.Body.Blocks {
				if child.Type == "aggregate" || child.Type == "event" {
					planTitle(child, plan)
				}
			}
		case slices.Contains(workflowKinds, block.Type):
			planTitle(block, plan)
			for _, child := range block.Body.Blocks {
				if child.Type == "scenario" || slices.Contains(elementKinds, child.Type) {
					planTitle(child, plan)
				}
			}
		}
	}
}

func planTitle(block *hclsyntax.Block, plan *edits) {
	if len(block.Labels) == 1 && block.Body.Attributes["title"] == nil {
		plan.titles[block] = humanize(block.Labels[0])
	}
}

// humanize is the v0.4.0 derived title of a label: words split on
// underscores, each capitalized ("todo_list" becomes "Todo List").
func humanize(label string) string {
	words := strings.Split(label, "_")
	for index, word := range words {
		if word == "" {
			continue
		}
		words[index] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

// rewrite applies plan to one member file with hclwrite, keeping every
// comment and the existing attribute order, and inserts boardBlock after the
// file's comment header. An edited file gets whitespace-only formatting
// (hclwrite.Format); a file without edits is returned byte for byte. Edits
// are keyed by the blocks of the hclsyntax parse; both parsers read the same
// block tree, so the two trees are walked in parallel.
func rewrite(file member, plan edits, boardBlock []byte) ([]byte, hcl.Diagnostics) {
	written, diagnostics := hclwrite.ParseConfig(file.source, file.name, hcl.Pos{Line: 1, Column: 1})
	if diagnostics.HasErrors() {
		return nil, diagnostics
	}
	applied, ok := applyEdits(file.body.Blocks, written.Body().Blocks(), plan)
	if !ok {
		return nil, hcl.Diagnostics{{Severity: hcl.DiagError, Summary: "Migration failed", Detail: fmt.Sprintf("the HCL writer read a different block structure in %s.", file.name)}}
	}
	if applied == 0 && boardBlock == nil {
		return file.source, nil
	}
	output := written.Bytes()
	if boardBlock != nil {
		header := commentHeader(output)
		output = slices.Concat(output[:len(header)], boardBlock, []byte("\n"), output[len(header):])
	}
	return hclwrite.Format(output), nil
}

// applyEdits walks the original and the writable block trees in parallel and
// applies the planned edits, children before parents: inserting attributes
// replaces a body with unstructured tokens, which hides its nested blocks from
// later edits. It returns the number of edited blocks, and false when the two
// trees differ.
func applyEdits(original hclsyntax.Blocks, written []*hclwrite.Block, plan edits) (int, bool) {
	if len(original) != len(written) {
		return 0, false
	}
	applied := 0
	for index, block := range original {
		target := written[index]
		nested, ok := applyEdits(block.Body.Blocks, target.Body().Blocks(), plan)
		if !ok {
			return 0, false
		}
		applied += nested
		if plan.renames[block] {
			target.SetType("automation")
			applied++
		}
		inserted := hclwrite.NewEmptyFile().Body()
		if title, ok := plan.titles[block]; ok {
			inserted.SetAttributeValue("title", cty.StringVal(title))
		}
		if traversal, ok := plan.types[block]; ok {
			inserted.SetAttributeTraversal("type", traversal)
		}
		if len(inserted.Attributes()) > 0 {
			prependAttributes(target.Body(), inserted)
			applied++
		}
	}
	return applied, true
}

// prependAttributes inserts the attributes of inserted at the top of body,
// after the line that holds the opening brace, keeping every existing token
// and comment. A single-line body is expanded to a multi-line one first.
func prependAttributes(body, inserted *hclwrite.Body) {
	attributes := inserted.BuildTokens(nil)
	if len(body.Attributes()) == 0 && len(body.Blocks()) > 0 {
		attributes = append(attributes, &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")})
	}
	existing := body.BuildTokens(nil)
	newline := hclwrite.Tokens{{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")}}
	var tokens hclwrite.Tokens
	split := slices.IndexFunc(existing, func(token *hclwrite.Token) bool { return bytes.HasSuffix(token.Bytes, []byte("\n")) })
	if split < 0 {
		tokens = slices.Concat(newline, attributes, existing)
		if len(existing) > 0 {
			tokens = append(tokens, newline...)
		}
	} else {
		tokens = slices.Concat(existing[:split+1], attributes, existing[split+1:])
	}
	body.Clear()
	body.AppendUnstructuredTokens(tokens)
}

// boardSource is the board block listing workflows one per line.
func boardSource(workflows []string) []byte {
	var builder strings.Builder
	builder.WriteString("board {\n")
	if len(workflows) == 0 {
		builder.WriteString("workflows = []\n")
	} else {
		builder.WriteString("workflows = [\n")
		for _, id := range workflows {
			builder.WriteString("workflow." + id + ",\n")
		}
		builder.WriteString("]\n")
	}
	builder.WriteString("}\n")
	return []byte(builder.String())
}

// commentHeader returns the file's leading comment header: the leading run of
// line comments and blank lines, up to its last blank line. Comments directly
// above the first block stay attached to it, so the board goes before them.
func commentHeader(source []byte) []byte {
	end, header := 0, 0
	for end < len(source) {
		line, _, found := bytes.Cut(source[end:], []byte{'\n'})
		trimmed := bytes.TrimSpace(line)
		blank := len(trimmed) == 0
		if !blank && !bytes.HasPrefix(trimmed, []byte("#")) && !bytes.HasPrefix(trimmed, []byte("//")) {
			break
		}
		end += len(line)
		if found {
			end++
		}
		if blank {
			header = end
		}
	}
	return source[:header]
}

func workflowReferences(attribute *hclsyntax.Attribute) []string {
	var ids []string
	for _, parts := range traversalList(attribute) {
		if len(parts) == 2 && parts[0] == "workflow" {
			ids = append(ids, parts[1])
		}
	}
	return ids
}

// traversalList returns the segments of each absolute traversal in a list
// attribute, skipping entries that are not plain traversals.
func traversalList(attribute *hclsyntax.Attribute) [][]string {
	if attribute == nil {
		return nil
	}
	expressions, diagnostics := hcl.ExprList(attribute.Expr)
	if diagnostics.HasErrors() {
		return nil
	}
	var lists [][]string
	for _, expression := range expressions {
		traversal, traversalDiagnostics := hcl.AbsTraversalForExpr(expression)
		if traversalDiagnostics.HasErrors() {
			continue
		}
		parts := []string{traversal.RootName()}
		for _, step := range traversal[1:] {
			attr, ok := step.(hcl.TraverseAttr)
			if !ok {
				parts = nil
				break
			}
			parts = append(parts, attr.Name)
		}
		if parts != nil {
			lists = append(lists, parts)
		}
	}
	return lists
}

func literalTrue(attribute *hclsyntax.Attribute) bool {
	if attribute == nil {
		return false
	}
	value, diagnostics := attribute.Expr.Value(nil)
	return !diagnostics.HasErrors() && value.Type() == cty.Bool && value.IsKnown() && !value.IsNull() && value.True()
}

func errorAt(subject hcl.Range, summary, detail string) *hcl.Diagnostic {
	return &hcl.Diagnostic{Severity: hcl.DiagError, Summary: summary, Detail: detail, Subject: subject.Ptr()}
}
