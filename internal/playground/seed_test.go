package playground

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSeedScriptEmbedsSourceVerbatim(t *testing.T) {
	source := []byte("event \"pet_added\" {\n  title = \"Pet Added\"\n}\n")

	script, err := SeedScript(source)
	if err != nil {
		t.Fatalf("SeedScript: %v", err)
	}

	literal := strings.TrimSuffix(strings.TrimPrefix(string(script), "window.EVENT_MODELING_SEED = "), ";\n")
	var got string
	if err := json.Unmarshal([]byte(literal), &got); err != nil {
		t.Fatalf("seed script does not contain a JSON string: %v", err)
	}
	if got != string(source) {
		t.Errorf("seed differs from source:\n got %q\nwant %q", got, source)
	}
}
