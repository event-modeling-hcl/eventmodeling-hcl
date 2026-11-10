package renderer

import (
	"os/exec"
	"strings"
	"testing"
)

func TestChapterInputsRemainVisibleWithoutMovingFullBoardEvents(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is required to exercise chapter filtering")
	}
	prefix, _, found := strings.Cut(viewerJS, "/* --------------------------- constants")
	if !found {
		t.Fatal("viewer initialization boundary not found")
	}
	fixture := `{
	  "slices":[
	    {"id":"produce","type":"state_change","stageCount":2,"elements":[
	      {"id":"command","kind":"command","stage":0},
	      {"id":"event","kind":"event","title":"Same label","ctx":"catalog","agg":"item","stage":1},
	      {"id":"scenario-event","kind":"event","title":"Same label","stage":1},
	      {"id":"unrelated-event","kind":"event","stage":1}]},
	    {"id":"view","type":"state_view","stageCount":2,"elements":[
	      {"id":"read","kind":"readmodel","stage":0},{"id":"screen","kind":"screen","stage":1}],
	      "scenarios":[{"given":[{"refKind":"event","refId":"scenario-event","ref":"Same label"}]}]},
	    {"id":"automate","type":"automation","stageCount":2,"elements":[
	      {"id":"processor","kind":"processor","stage":0},{"id":"dispatch","kind":"command","stage":1}]},
	    {"id":"other","type":"state_view","stageCount":1,"elements":[{"id":"other-read","kind":"readmodel","stage":0}]}
	  ],
	  "chapters":[{"id":"producer","slices":["produce"]},{"id":"consumer","slices":["view","automate"]},{"id":"automation","slices":["automate"]}],
	  "edges":[{"from":"command","to":"event"},{"from":"event","to":"read"},{"from":"read","to":"screen"},{"from":"event","to":"processor"},{"from":"processor","to":"dispatch"},{"from":"unrelated-event","to":"other-read"}],
	  "hotspots":[{"onId":"event"},{"onId":"other-read"}]
	}`
	cases := []struct {
		name       string
		hash       string
		assertions string
	}{
		{
			name: "consumer chapter",
			hash: "#chapter/consumer/model",
			assertions: `
assert.deepStrictEqual(MODEL.slices.map(s=>s.id), ["view","automate"]);
const ids = MODEL.slices.flatMap(s=>s.elements.map(e=>e.id));
assert.strictEqual(new Set(ids).size, ids.length);
assert(!ids.includes("command") && !ids.includes("unrelated-event") && !ids.includes("other-read"));
const input = MODEL.slices[0].elements.find(e=>e.id==="event");
assert(input.given);
assert.strictEqual(input.stage, 0);
assert.strictEqual(input.ctx, "catalog");
assert.strictEqual(input.agg, "item");
assert(MODEL.slices[0].elements.find(e=>e.id==="scenario-event").given);
assert.strictEqual(MODEL.slices[0].elements.find(e=>e.id==="read").stage, 1);
assert.strictEqual(MODEL.slices[0].elements.find(e=>e.id==="screen").stage, 2);
assert.strictEqual(MODEL.slices[0].stageCount, 3);
assert(MODEL.edges.some(e=>e.from==="event" && e.to==="read"));
assert(MODEL.edges.some(e=>e.from==="event" && e.to==="processor"));
assert(!MODEL.edges.some(e=>e.from==="command" || e.to==="other-read"));
assert.deepStrictEqual(MODEL.hotspots, [{onId:"event"}]);
assert.strictEqual(FULL_MODEL.slices[0].elements.find(e=>e.id==="event").given, undefined);
assert.strictEqual(FULL_MODEL.slices[0].elements.find(e=>e.id==="event").stage, 1);
assert.strictEqual(FULL_MODEL.slices[1].elements.find(e=>e.id==="read").stage, 0);
assert.strictEqual(FULL_MODEL.slices[1].elements.length, 2);
assert.strictEqual(MODEL.slices[1], FULL_MODEL.slices[2]);`,
		},
		{
			name: "automation chapter",
			hash: "#chapter/automation/storming",
			assertions: `
assert.deepStrictEqual(MODEL.slices.map(s=>s.id), ["automate"]);
assert(MODEL.slices[0].elements.find(e=>e.id==="event").given);
assert.strictEqual(MODEL.slices[0].elements.find(e=>e.id==="processor").stage, 1);
assert(MODEL.edges.some(e=>e.from==="event" && e.to==="processor"));`,
		},
		{
			name: "producer chapter",
			hash: "#chapter/producer/model",
			assertions: `
assert.strictEqual(MODEL.slices[0], FULL_MODEL.slices[0]);
assert.strictEqual(MODEL.slices[0].elements.filter(e=>e.id==="event").length, 1);
assert.strictEqual(MODEL.slices[0].elements.find(e=>e.id==="event").given, undefined);`,
		},
		{
			name:       "full board",
			hash:       "#model",
			assertions: `assert.strictEqual(MODEL, FULL_MODEL);`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			script := "const assert = require('node:assert/strict');\nconst location = {hash:" + "\"" + tc.hash + "\"" + "};\n" + strings.Replace(prefix, modelMarker, fixture, 1) + tc.assertions
			cmd := exec.Command(node, "-")
			cmd.Stdin = strings.NewReader(script)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("chapter filtering: %v\n%s", err, output)
			}
		})
	}
}
