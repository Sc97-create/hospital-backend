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
	GetPatientByID(c *fiber.Ctx) (err error)
	UpdatePatient(c *fiber.Ctx) (err error)
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
	payloadModel.Name, err = params.Getstring("name")
	if err != nil {
		return
	}
	payloadModel.BloodGroup, err = params.Getstring("blood_group")
	if err != nil {
		return
	}
	payloadModel.Address, err = params.Getstring("address")
	if err != nil {
		return
	}

	payloadModel.Age, err = params.Getstring("age")
	if err != nil {
		return
	}
	payloadModel.UserID, err = params.Getstring("user_id")
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
	payloadModel.OrganisationID, err = params.Getstring("organisation_id")
	if err != nil {
		return
	}
	payloadModel.EmailID, err = params.Getstring("email_id")
	if err != nil {
		return
	}
	payloadModel.MobileNumber, err = params.Getstring("mobile_number")
	if err != nil {
		return
	}
	//payloadModel.ActiveCondition, _ = params.Getstring("active_condition")
	return
}
func (p *PatientController) GetPatientByID(c *fiber.Ctx) (err error) {
	patientID := c.Params("patientID")
	patient, err := p.PatientService.FindOne(patientID)
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}
	res := make(map[string]interface{})
	res["data"] = patient
	res["code"] = 200
	err = c.Status(200).JSON(&res)
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
	patient, total, err := p.PatientService.FindMany(limit, pageNo, organisationID)
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	var response dto.PatientListResponse
	response.Data = patient
	response.Total = total
	response.Code = 200
	err = c.Status(200).JSON(&response)
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	return

}

// UpdatePatient handles the PUT request to update patient information
func (p *PatientController) UpdatePatient(c *fiber.Ctx) (err error) {

	var payloadModel dto.UpdatePatientInfo
	params, err := params.New(c)
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}

	payloadModel.Name, _ = params.Getstring("name")

	payloadModel.Age, _ = params.Getstring("age")

	payloadModel.Weight, _ = params.Getstring("weight")

	payloadModel.Gender, _ = params.Getstring("gender")

	payloadModel.EmailID, _ = params.Getstring("email_id")

	payloadModel.MobileNumber, _ = params.Getstring("mobile_number")

	payloadModel.OrganisationID, _ = params.Getstring("organisation_id")
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}
	payloadModel.PatientID, err = params.Getstring("patient_id")
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}

	// Call service to update patient
	err = p.PatientService.UpdatePatientSrv(c.Context(), payloadModel)
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}

	res := make(map[string]interface{})
	res["message"] = "patient updated successfully"
	res["patient_id"] = payloadModel.PatientID
	res["code"] = 200
	return c.Status(200).JSON(res)
}
