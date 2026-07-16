package render

import (
	"bytes"
	"fmt"
	"hospital-backend/config"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

type TemplateConfig struct {
	Subject string
}

type HTMLRenderer struct {
	templates  map[string]*template.Template
	subjects   map[string]string
	withLayout map[string]bool
}

func NewHTMLRenderer(templatePath config.NotificationTemplateFilepath, subjects map[string]string) (*HTMLRenderer, error) {

	r := &HTMLRenderer{
		templates:  make(map[string]*template.Template),
		subjects:   subjects,
		withLayout: make(map[string]bool),
	}
	files := createFilepath(templatePath)

	layoutTemplates := map[string]bool{
		"appointment_created":  true,
		"patient_created":      true,
		"prescription_created": true,
	}

	for key, file := range files {
		var (
			tmpl       *template.Template
			err        error
			usesLayout bool
		)

		if layoutTemplates[key] {
			layoutPath := filepath.Join(filepath.Dir(file), "layout.tmpl")
			tmpl, err = template.ParseFiles(layoutPath, file)
			usesLayout = true
		} else {
			tmpl, err = template.ParseFiles(file)
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

func createFilepath(templatePath config.NotificationTemplateFilepath) map[string]string {
	filemap := make(map[string]string)
	dir, err := os.Getwd()
	if err != nil {
		return nil
	}
	getlastkey := func(path string) string {
		normalizedPath := filepath.FromSlash(path)
		return strings.TrimSuffix(filepath.Base(normalizedPath), ".tmpl")
	}

	filemap[getlastkey(templatePath.Appointmentcreated)] = filepath.Join(dir, templatePath.Appointmentcreated)
	//filemap[getlastkey(templatePath.AppointmentUpdated)] = filepath.Join(dir, templatePath.AppointmentUpdated)
	filemap[getlastkey(templatePath.Patientcreated)] = filepath.Join(dir, templatePath.Patientcreated)
	filemap[getlastkey(templatePath.PatientUpdated)] = filepath.Join(dir, templatePath.PatientUpdated)
	filemap[getlastkey(templatePath.PrescriptionCreated)] = filepath.Join(dir, templatePath.PrescriptionCreated)
	filemap[getlastkey(templatePath.MedicationAdherence)] = filepath.Join(dir, templatePath.MedicationAdherence)
	filemap[getlastkey(templatePath.FollowUpReminder)] = filepath.Join(dir, templatePath.FollowUpReminder)
	filemap[getlastkey(templatePath.PaymentLinkGenerated)] = filepath.Join(dir, templatePath.PaymentLinkGenerated)
	//filemap[getlastkey(templatePath.PaymentRecieved)] = filepath.Join(dir, templatePath.PaymentRecieved)

	return filemap
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
