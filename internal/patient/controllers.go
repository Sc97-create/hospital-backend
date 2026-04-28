package patient

import (
	"hospital-backend/internal/patient/dto"
	"hospital-backend/shared/params"

	errwrap "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
)

type IPatientController interface {
	AddGeneralInfoHandler(c *fiber.Ctx) (err error)
	Find(c *fiber.Ctx) (err error)
}
type PatientController struct {
	PatientService *PatientService
}

func NewPatientControllerInterface(service *PatientService) IPatientController {
	return &PatientController{PatientService: service}
}
func (p *PatientController) AddGeneralInfoHandler(c *fiber.Ctx) (err error) {
	params, err := params.New(c)
	if err != nil {
		return errwrap.Wrap(err, c, 409)

	}
	var payloadModel dto.PatientInfo
	payloadModel, err = p.ToPatientModel(params)
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}
	id, err := p.PatientService.CreatePatientSrv(payloadModel)
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}
	res := make(map[string]interface{})
	res["message"] = "general info added"
	res["patient_id"] = id
	res["code"] = 200
	return c.Status(200).JSON(res)
}
func (p *PatientController) ToPatientModel(params *params.Payload) (payloadModel dto.PatientInfo, err error) {
	// payloadModel.UserID, err = params.Getstring("user_id")
	// if err != nil {
	// 	return
	// } ==> once userid is saved for current user then we can start sending
	payloadModel.FirstName, err = params.Getstring("first_name")
	if err != nil {
		return
	}
	payloadModel.LastName, err = params.Getstring("last_name")
	if err != nil {
		return
	}

	payloadModel.Age, err = params.Getstring("age")
	if err != nil {
		return
	}
	payloadModel.Weight, err = params.Getstring("weight")
	if err != nil {
		return
	}
	payloadModel.Gender, err = params.Getstring("gender")
	if err != nil {
		return
	}
	payloadModel.OrganisationID = "dce26168-eb8d-4723-a21e-2c33ad3ce39c"
	payloadModel.EmailID, _ = params.Getstring("email_id")
	payloadModel.MobileNumber, _ = params.Getstring("mobile_number")
	payloadModel.DoctorID, _ = params.Getstring("doctor_id")
	payloadModel.Symptoms, _ = params.GetStringArray("symptoms")
	//payloadModel.ActiveCondition, _ = params.Getstring("active_condition")
	return
}
func (p *PatientController) GetPatientByID(c *fiber.Ctx) (err error) {
	patientID := c.Query("patient_id")
	patient, err := p.PatientService.FindOne(patientID)
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}
	return c.Status(200).JSON(patient)
}

func PreviewAppointment(c *fiber.Ctx) (err error) {
	//patientID := c.Query("patient_id")
	//patientModel := PatientRepo{}
	//service := NewPatientService(&patientModel)
	//patient, err := service.FindOne(patientID)
	response := make(map[string]interface{})
	response["data"] = "patient"
	response["code"] = 200
	err = c.JSON(&response)
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	return
}
func (p *PatientController) Find(c *fiber.Ctx) (err error) {
	limit := c.Query("limit")
	pageNo := c.Query("page_no")
	organisationID := c.Query("organisation_id")
	patient, err := p.PatientService.FindMany(limit, pageNo, organisationID)
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	response := make(map[string]interface{})
	response["data"] = patient
	response["code"] = 200
	err = c.Status(200).JSON(&response)
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	return

}
