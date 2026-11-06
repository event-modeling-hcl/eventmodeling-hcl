// Package formatter canonicalizes Event Modeling HCL source formatting.
package formatter

import (
	"bytes"
	"cmp"
	"slices"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// Format applies HCL whitespace formatting and canonical attribute ordering
// while preserving source block order and raw attribute expression tokens.
func Format(filename string, source []byte) ([]byte, hcl.Diagnostics) {
	whitespaceFormatted := hclwrite.Format(source)
	file, diagnostics := hclwrite.ParseConfig(whitespaceFormatted, filename, hcl.Pos{Line: 1, Column: 1})
	if diagnostics.HasErrors() {
		return nil, diagnostics
	}
	output := hclwrite.NewEmptyFile()
	rebuildBody(output.Body(), file.Body(), "")
	formatted := hclwrite.Format(output.Bytes())
	return append(leadingCommentTrivia(whitespaceFormatted), formatted...), diagnostics
}

func leadingCommentTrivia(source []byte) []byte {
	var prefix []byte
	remaining := source
	seenComment := false
	for len(remaining) > 0 {
		line, rest, found := bytes.Cut(remaining, []byte{'\n'})
		withNewline := line
		if found {
			withNewline = append(append([]byte(nil), line...), '\n')
		}
		trimmed := bytes.TrimSpace(line)
		if bytes.HasPrefix(trimmed, []byte("#")) || bytes.HasPrefix(trimmed, []byte("//")) {
			prefix = append(prefix, withNewline...)
			seenComment = true
		} else if len(trimmed) == 0 && seenComment {
			prefix = append(prefix, withNewline...)
		} else {
			break
		}
		if !found {
			break
		}
		remaining = rest
	}
	return prefix
}

// rebuildBody copies source into destination in canonical attribute order.
// blockType is the type of the block that owns source, or "" for the file.
func rebuildBody(destination, source *hclwrite.Body, blockType string) {
	attributes := source.Attributes()
	names := make([]string, 0, len(attributes))
	for name := range attributes {
		names = append(names, name)
	}
	slices.SortFunc(names, func(left, right string) int { return compareAttributeNames(blockType, left, right) })
	for _, name := range names {
		destination.SetAttributeRaw(name, attributes[name].Expr().BuildTokens(nil))
	}
	if len(names) > 0 && len(source.Blocks()) > 0 {
		destination.AppendNewline()
	}
	for index, block := range source.Blocks() {
		copy := hclwrite.NewBlock(block.Type(), block.Labels())
		rebuildBody(copy.Body(), block.Body(), block.Type())
		destination.AppendBlock(copy)
		if index < len(source.Blocks())-1 {
			destination.AppendNewline()
		}
	}
}

func compareAttributeNames(blockType, left, right string) int {
	if rank := cmp.Compare(attributeRank(blockType, left), attributeRank(blockType, right)); rank != 0 {
		return rank
	}
	return cmp.Compare(sortName(left), sortName(right))
}

// sortName orders attributes inside one rank. The key flags of a field sort
// directly after id_attribute: correlation_key, then deduplication_key.
func sortName(name string) string {
	switch name {
	case "correlation_key":
		return "id_attribute\x01"
	case "deduplication_key":
		return "id_attribute\x02"
	}
	return name
}

func attributeRank(blockType, name string) int {
	switch name {
	case "title":
		return 10
	case "description":
		return 20
	case "group_id":
		return 30
	case "tags":
		return 40
	case "owner":
		return 50
	case "status":
		return 60
	case "external":
		return 70
	case "kind":
		return 90
	case "case":
		return 91
	case "empty_history":
		return 92
	case "category":
		return 95
	case "question", "type", "cardinality", "mapping", "schema", "identity", "end", "path":
		return 100
	case "options":
		return 101
	case "concerns":
		return 102
	case "aggregate":
		return 110
	case "aggregate_dependencies":
		return 120
	case "api_endpoint":
		return 130
	case "service":
		return 140
	case "creates_aggregate", "external_trigger", "triggers":
		return 150
	case "actor":
		return 160
	case "optional", "technical_attribute", "generated", "id_attribute", "pii", "correlation_key", "deduplication_key":
		return 170
	case "sketched", "prototype", "list_element", "url", "examples", "expect_empty_list", "expect_no_dispatch":
		return 180
	case "workflows":
		return 190
	case "on":
		// Evidence and decisions state their semantic configuration before
		// their relationships, so on sorts last there.
		if blockType == "evidence" || blockType == "decision" {
			return 920
		}
		return 190
	case "fields":
		return 195
	case "signal":
		return 501
	case "statement":
		return 502
	case "reason":
		return 503
	case "from":
		return 900
	case "before", "source":
		return 901
	case "after":
		return 902
	case "to":
		return 910
	default:
		return 500
	}
}
