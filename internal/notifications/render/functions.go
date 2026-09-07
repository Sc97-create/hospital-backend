package render

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"path"
	"strings"
)

// templateFS embeds all notification templates into the binary so the
// application does not depend on OS-specific filesystem paths at runtime.
//
//go:embed templates/*.tmpl
var templateFS embed.FS

const (
	templateDir = "templates"
	layoutFile  = "templates/layout.tmpl"
)

type TemplateConfig struct {
	Subject string
}

type HTMLRenderer struct {
	templates  map[string]*template.Template
	subjects   map[string]string
	withLayout map[string]bool
}

func NewHTMLRenderer(subjects map[string]string) (*HTMLRenderer, error) {

	r := &HTMLRenderer{
		templates:  make(map[string]*template.Template),
		subjects:   subjects,
		withLayout: make(map[string]bool),
	}

	layoutTemplates := map[string]bool{
		"appointment_created":         true,
		"patient_created":             true,
		"prescription_created":        true,
		"payment_recieved":            true,
		"employee_created_welcome":    true,
		"employee_created_onboarding": true,
		"employee_created_compact":    true,
	}

	entries, err := fs.Glob(templateFS, templateDir+"/*.tmpl")
	if err != nil {
		return nil, err
	}

	for _, file := range entries {
		key := strings.TrimSuffix(path.Base(file), ".tmpl")
		if key == "layout" {
			continue
		}

		var (
			tmpl       *template.Template
			usesLayout bool
		)

		if layoutTemplates[key] {
			tmpl, err = template.ParseFS(templateFS, layoutFile, file)
			usesLayout = true
		} else {
			tmpl, err = template.ParseFS(templateFS, file)
		}
		if err != nil {
			return nil, err
		}

		r.templates[key] = tmpl
		if usesLayout {
			r.withLayout[key] = true
		}
	}

	return r, nil
}
func (r *HTMLRenderer) Render(notificationType string, data any) (string, error) {

	tmpl, ok := r.templates[notificationType]
	if !ok {
		return "", fmt.Errorf(
			"template not found: %s",
			notificationType,
		)
	}

	var buf bytes.Buffer

	var err error
	if r.withLayout[notificationType] {
		err = tmpl.ExecuteTemplate(&buf, "layout", data)
	} else {
		err = tmpl.Execute(&buf, data)
	}
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
