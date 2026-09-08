// Package playground provides small build-time helpers for the browser editor.
package playground

import "encoding/json"

// SeedScript serializes source as the editor seed without maintaining a second
// hand-written copy of the example in JavaScript.
func SeedScript(source []byte) ([]byte, error) {
	literal, err := json.Marshal(string(source))
	if err != nil {
		return nil, err
	}
	return append(append([]byte("window.EVENT_MODELING_SEED = "), literal...), []byte(";\n")...), nil
}
