package utils

import (
	"bytes"
	"fmt"
	"text/template"

	"gopkg.in/gomail.v2"
)

func EmailTemplate(data any, templatetype string) (format string, err error) {
	path := fmt.Sprintf("templates/%s.html", templatetype)
	tmp, err := template.ParseFS(templates, path)
	if err != nil {
		return
	}
	buf := bytes.Buffer{}
	err = tmp.Execute(&buf, data)
	if err != nil {
		return
	}
	return buf.String(), nil
}
func SendEmail(emailID string, body string) (err error) {
	m := gomail.NewMessage()
	m.SetHeader("From", "sachinchate34@gmail.com")
	m.SetHeader("To", emailID)
	m.SetHeader("Subject", "Verify License and Get Started!!!")
	m.SetBody("text/html", body)
	d := gomail.NewDialer("smtp.gmail.com", 587, "sachinchate34@gmail.com", "affd ccib fwcn nijr")
	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return
}
