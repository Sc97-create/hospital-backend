package email

import "hospital-backend/internal/email/utils"

type EmailService struct {
	EmpName    string
	OrgName    string
	Department string
	Role       string
	EmailID    string
	Password   string
	LoginUrl   string
	Appname    string
}

func SendNotification(employeeName string, OrganisationName string, Dept string, Role string, EmailID string, Password string, LoginUrl string, Appname string) (err error) {
	var e EmailService
	e.EmpName = employeeName
	e.OrgName = OrganisationName
	e.Department = Dept
	e.Role = Role
	e.Appname = Appname
	e.LoginUrl = LoginUrl
	e.Password = Password

	formatstring, err := utils.EmailTemplate(e, "employee-template")
	if err != nil {
		return
	}
	err = utils.SendEmail(EmailID, formatstring)
	if err != nil {
		return
	}

	return
}
