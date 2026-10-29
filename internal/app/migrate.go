package app

import "github.com/event-modeling-hcl/eventmodeling-hcl/internal/migrate"

// MigratePath reads the v0.4.0 model at path, a file or a folder, rewrites it
// for the current language, and validates the rewritten files under the valid
// profile. When migration fails or the rewritten model has a core error, the
// Result is empty, nothing counts as migrated, and the core errors are
// returned; they point into the rewritten source, not the files on disk.
// Judgment signals do not block migration and are not returned.
func MigratePath(path string) (migrate.Result, Diagnostics) {
	files, diagnostics := ReadModel(path)
	if diagnostics.HasErrors() {
		return migrate.Result{}, diagnostics
	}
	result, migrateDiagnostics := migrate.Migrate(files)
	diagnostics = toDiagnostics(migrateDiagnostics)
	if diagnostics.HasErrors() {
		return migrate.Result{}, diagnostics
	}
	validation := ValidateFiles(path, result.Files, Valid)
	if !validation.HasCoreErrors() {
		return result, diagnostics
	}
	for _, diagnostic := range validation {
		if diagnostic.Severity == "Error" && !diagnostic.Judgment {
			diagnostics = append(diagnostics, diagnostic)
		}
	}
	return migrate.Result{}, diagnostics
}
