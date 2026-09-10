package renderer

import (
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/model"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/version"
)

//go:embed assets/viewer.shell.html
var viewerShell string

//go:embed assets/viewer.css
var viewerCSS string

//go:embed assets/viewer.js
var viewerJS string

const (
	stylesMarker      = "__STYLES__"
	scriptMarker      = "__SCRIPT__"
	modelMarker       = "__MODEL_JSON__"
	specVersionMarker = "__SPEC_VERSION__"
)

func RenderHTML(view *ViewModel) (string, error) {
	encoded, err := json.Marshal(view)
	if err != nil {
		return "", err
	}
	doc := strings.Replace(viewerShell, stylesMarker, viewerCSS, 1)
	doc = strings.Replace(doc, scriptMarker, viewerJS, 1)
	doc = strings.Replace(doc, specVersionMarker, version.Spec, 1)
	return strings.Replace(doc, modelMarker, string(encoded), 1), nil
}

func Render(filename string, source *model.Model) (string, error) {
	return RenderHTML(BuildViewModel(filename, source))
}
