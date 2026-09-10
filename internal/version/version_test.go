package version_test

import (
	"regexp"
	"testing"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/version"
)

func TestSpec_MatchesSemverPattern(t *testing.T) {
	pattern := regexp.MustCompile(`^v\d+\.\d+\.\d+$`)
	if !pattern.MatchString(version.Spec) {
		t.Fatalf("version.Spec = %q, want match for %s", version.Spec, pattern)
	}
}
