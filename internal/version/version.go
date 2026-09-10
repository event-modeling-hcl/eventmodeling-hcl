// Package version holds the single source of truth for the Event Modeling
// HCL Specification version that the renderer and its embedded assets
// reference. It is unrelated to the CLI tool's own version, which is
// injected separately via the main.version ldflag.
package version

// Spec is the version of the Event Modeling HCL Specification that this
// renderer targets.
const Spec = "v0.3.0"
