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
	templates map[string]*template.Template
	subjects  map[string]string
}

func NewHTMLRenderer(templatePath config.NotificationTemplateFilepath, subjects map[string]string) (*HTMLRenderer, error) {

	r := &HTMLRenderer{
		templates: make(map[string]*template.Template),
		subjects:  subjects,
	}
	files := createFilepath(templatePath)

	for key, file := range files {

		// 	if file.IsDir() {
		// 		continue
		// 	}

		// 	if filepath.Ext(file.Name()) != ".tmpl" {
		// 		continue
		// 	}

		// 	fullPath := filepath.Join(
		// 		templatePath,
		// 		file.Name(),
		// 	)

		tmpl, err := template.ParseFiles(file)
		if err != nil {
			return nil, err
		}
		r.templates[key] = tmpl
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
		value := strings.Split(path, "\\")
		return strings.TrimSuffix(value[len(value)-1], ".tmpl")
	}

	filemap[getlastkey(templatePath.Appointmentcreated)] = filepath.Join(dir, templatePath.Appointmentcreated)
	//filemap[getlastkey(templatePath.AppointmentUpdated)] = filepath.Join(dir, templatePath.AppointmentUpdated)
	filemap[getlastkey(templatePath.Patientcreated)] = filepath.Join(dir, templatePath.Patientcreated)
	filemap[getlastkey(templatePath.PatientUpdated)] = filepath.Join(dir, templatePath.PatientUpdated)
	filemap[getlastkey(templatePath.PrescriptionCreated)] = filepath.Join(dir, templatePath.PrescriptionCreated)
	filemap[getlastkey(templatePath.MedicationAdherence)] = filepath.Join(dir, templatePath.MedicationAdherence)
	filemap[getlastkey(templatePath.FollowUpReminder)] = filepath.Join(dir, templatePath.FollowUpReminder)
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

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
