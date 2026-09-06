package renderer

import (
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/model"
)

//go:embed assets/viewer.html.tmpl
var viewerTemplate string

const modelMarker = "__MODEL_JSON__"

func RenderHTML(view *ViewModel) (string, error) {
	encoded, err := json.Marshal(view)
	if err != nil {
		return "", err
	}
	return strings.Replace(viewerTemplate, modelMarker, string(encoded), 1), nil
}

func Render(filename string, source *model.Model) (string, error) {
	return RenderHTML(BuildViewModel(filename, source))
}
