package patient

import (
	"hospital-backend/internal/patient/dto"
	"hospital-backend/shared/params"
	"log"

	errwrap "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func AddGeneralInfoHandler(c *fiber.Ctx, service *PatientService) (err error) {
	params, err := params.New(c)
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadModel := dto.PatientInfo{}
	payloadModel.UserID, err = params.Getstring("user_id")
	if err != nil {
		errwrap.Wrap(err, c, 409)
	}
	payloadModel.FirstName, err = params.Getstring("first_name")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadModel.LastName, err = params.Getstring("last_name")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}

	payloadModel.Age, err = params.Getstring("age")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadModel.Weight, err = params.Getstring("weight")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadModel.Gender, err = params.Getstring("gender")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadModel.OrganisationID = uuid.New().String()
	payloadModel.EmailID, _ = params.Getstring("email_id")
	payloadModel.MobileNumber, _ = params.Getstring("mobile_number")
	payloadModel.DoctorID, _ = params.Getstring("doctor_id")
	payloadModel.Symptoms, _ = params.GetStringArray("symptoms")
	payloadModel.ActiveCondition, _ = params.Getstring("active_condition")

	id, err := service.CreatePatientSrv(payloadModel)
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}
	res := make(map[string]interface{})
	res["message"] = "general info added"
	res["patient_id"] = id
	res["code"] = 200
	err = c.JSON(res)
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	return
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
func PatientHandler(c *fiber.Ctx, service *PatientService) (err error) {
	param, err := params.New(c)
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	limit, err := param.Getint("limit")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	pageno, err := param.Getint("page_no")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	log.Println("pageno", pageno, "limit", limit)
	// patient, err := service.FindMany(limit, pageno)
	// if err != nil {
	// 	errwrap.Wrap(err, c, 409)
	// 	return
	// }
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
