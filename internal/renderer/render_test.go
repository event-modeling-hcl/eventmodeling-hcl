package renderer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/model"
)

func TestRenderHTML_InjectsEscapedModelIntoSelfContainedPage(t *testing.T) {
	view := &ViewModel{
		Title:    `</script><script>alert("x")</script>`,
		Version:  specVersion,
		Actors:   map[string]Actor{},
		Contexts: map[string]Context{},
		Slices:   []Slice{{ID: "example", Type: "state_change", Title: "Example"}},
	}

	html, err := RenderHTML(view)
	if err != nil {
		t.Fatalf("render HTML: %v", err)
	}
	for _, expected := range []string{"<!doctype html>", "const MODEL =", `\u003c/script\u003e`, "image-preview-fallback"} {
		if !strings.Contains(html, expected) {
			t.Errorf("HTML missing %q", expected)
		}
	}
	if strings.Contains(html, modelMarker) {
		t.Fatal("HTML still contains model marker")
	}
	if strings.Contains(html, `</script><script>alert`) {
		t.Fatal("model title escaped from script data")
	}
	if strings.Contains(html, "https://") {
		t.Fatal("template contains an external application dependency")
	}
}

func TestRender_RendersEveryCurrentExample(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "examples", "*.em.hcl"))
	if err != nil {
		t.Fatalf("find examples: %v", err)
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			source, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("read example: %v", readErr)
			}
			loaded, diagnostics := model.Load(path, source, model.Valid)
			if diagnostics.HasErrors() {
				t.Fatalf("load example: %s", diagnostics.Error())
			}

			html, renderErr := Render(path, loaded)
			if renderErr != nil {
				t.Fatalf("render example: %v", renderErr)
			}
			if !strings.Contains(html, "<!doctype html>") || !strings.Contains(html, loaded.Workflows[0].Title) {
				t.Fatal("rendered document is missing page or workflow content")
			}
		})
	}
}

func TestRenderHTML_IncludesAdaptiveTimelineLayout(t *testing.T) {
	view := &ViewModel{
		Title:    "Layout",
		Version:  specVersion,
		Actors:   map[string]Actor{"operator": {Title: "Operator"}},
		Contexts: map[string]Context{"partner": {Title: "Partner", External: true}},
		Slices: []Slice{{
			ID: "translate", Type: "translation", Title: "Translate", StageCount: 4,
			Actors:   []SliceActor{{ID: "operator", Title: "Operator", Stage: 0, ScreenIDs: []string{"screen"}}},
			Elements: []Element{{ID: "event", Kind: "event", Title: "Received", External: true}},
		}},
	}

	html, err := RenderHTML(view)
	if err != nil {
		t.Fatalf("render HTML: %v", err)
	}
	for _, expected := range []string{
		"--external-event-fill",
		`<span class="sub">model <b id="m-title">—</b> · HCL Spec <span class="mono" id="m-version">—</span></span>`,
		`{key:"actors"`,
		`{key:"processors"`,
		"function sliceWidth",
		"event-strip",
		"actor-card",
	} {
		if !strings.Contains(html, expected) {
			t.Errorf("HTML missing adaptive layout hook %q", expected)
		}
	}
}
