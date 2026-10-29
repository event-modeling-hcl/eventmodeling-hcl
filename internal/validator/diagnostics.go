package validator

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
)

type diagnosticCode string

const (
	codeUnclassified diagnosticCode = "EM000"

	codeReadFile                  diagnosticCode = "EM001"
	codeDuplicateID               diagnosticCode = "EM002"
	codeInvalidBlockLabel         diagnosticCode = "EM003"
	codeInvalidWorkflowChild      diagnosticCode = "EM004"
	codeInvalidChapter            diagnosticCode = "EM005"
	codeInvalidAttribute          diagnosticCode = "EM007"
	codeInvalidEnum               diagnosticCode = "EM008"
	codeInvalidFieldType          diagnosticCode = "EM009"
	codeInvalidExample            diagnosticCode = "EM010"
	codeWorkflowInSeveralChapters diagnosticCode = "EM014"
	codeInvalidBoard              diagnosticCode = "EM015"
	codeBoardRepeatsWorkflow      diagnosticCode = "EM016"

	codeInvalidReference    diagnosticCode = "EM101"
	codeUnresolvedReference diagnosticCode = "EM102"

	codeInvalidFlowReference diagnosticCode = "EM201"

	codeInvalidScenario       diagnosticCode = "EM301"
	codeInvalidScenarioStep   diagnosticCode = "EM302"
	codeInvalidScenarioTarget diagnosticCode = "EM303"

	codeBedAntiPattern        diagnosticCode = "EM401"
	codeLeftChairAntiPattern  diagnosticCode = "EM402"
	codeRightChairAntiPattern diagnosticCode = "EM403"
	codeCommandWithoutReason  diagnosticCode = "EM404"
	codeShelfAntiPattern      diagnosticCode = "EM405"
	codeOpenHotspot           diagnosticCode = "EM406"
)

// isJudgmentCode reports whether code is a judgment signal: the EM4xx family
// (EM401-EM406). Every other code is a core diagnostic, whose severity no
// profile changes.
func isJudgmentCode(code diagnosticCode) bool {
	return strings.HasPrefix(string(code), "EM4")
}

// judgmentSeverity is the severity profile assigns to the judgment signal code:
// workshop reports every signal as information, valid as a warning, and strict
// rejects commands without a reason and open hotspots.
func judgmentSeverity(code diagnosticCode, profile Profile) hcl.DiagnosticSeverity {
	switch profile {
	case Workshop:
		return hcl.DiagInvalid
	case Valid:
		return hcl.DiagWarning
	case Strict:
		if code == codeCommandWithoutReason || code == codeOpenHotspot {
			return hcl.DiagError
		}
	}
	return hcl.DiagWarning
}

// Profile controls the severity assigned to Event Modeling judgment diagnostics.
type Profile uint8

const (
	Workshop Profile = iota
	Valid
	Strict
)

func (p Profile) String() string {
	switch p {
	case Workshop:
		return "workshop"
	case Valid:
		return "valid"
	case Strict:
		return "strict"
	default:
		return "valid"
	}
}

// ParseProfile converts a CLI profile name into its typed representation.
func ParseProfile(value string) (Profile, bool) {
	switch value {
	case "workshop":
		return Workshop, true
	case "valid":
		return Valid, true
	case "strict":
		return Strict, true
	default:
		return Valid, false
	}
}

func diag(code diagnosticCode, severity hcl.DiagnosticSeverity, subject hcl.Range, summary, detail string) *hcl.Diagnostic {
	return &hcl.Diagnostic{Severity: severity, Summary: summary, Detail: detail, Subject: &subject, Extra: code}
}

func errorDiagnostic(code diagnosticCode, subject hcl.Range, summary, detail string) *hcl.Diagnostic {
	return diag(code, hcl.DiagError, subject, summary, detail)
}

// judgmentSignal creates a judgment signal. Its severity is a placeholder:
// applyProfile assigns the severity of the selected profile.
func judgmentSignal(code diagnosticCode, subject hcl.Range, summary, detail string) *hcl.Diagnostic {
	return diag(code, hcl.DiagWarning, subject, summary, detail)
}

// DiagnosticCode returns a stable Event Modeling code, or EM000 for diagnostics
// produced by HCL before Event Modeling validation begins.
func DiagnosticCode(diagnostic *hcl.Diagnostic) string {
	if code, ok := diagnostic.Extra.(diagnosticCode); ok {
		return string(code)
	}
	return string(codeUnclassified)
}

// IsJudgment reports whether diagnostic is a judgment signal (EM401-EM406)
// rather than a core diagnostic. An error-severity judgment diagnostic is a
// policy rejection, not a core failure.
func IsJudgment(diagnostic *hcl.Diagnostic) bool {
	code, ok := diagnostic.Extra.(diagnosticCode)
	return ok && isJudgmentCode(code)
}

// DiagnosticCodeExtra provides an HCL diagnostic Extra value with an Event
// Modeling code for adapters that construct diagnostics outside this package.
func DiagnosticCodeExtra(code string) interface{} {
	return diagnosticCode(code)
}

// hasCoreErrors reports whether any core diagnostic is an error.
func hasCoreErrors(diagnostics hcl.Diagnostics) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == hcl.DiagError && !IsJudgment(diagnostic) {
			return true
		}
	}
	return false
}

// applyProfile is the policy evaluator: it assigns every judgment signal its
// severity under profile and leaves core diagnostics unchanged.
func applyProfile(diagnostics hcl.Diagnostics, profile Profile) hcl.Diagnostics {
	for _, diagnostic := range diagnostics {
		if code, ok := diagnostic.Extra.(diagnosticCode); ok && isJudgmentCode(code) {
			diagnostic.Severity = judgmentSeverity(code, profile)
		}
	}
	return diagnostics
}
