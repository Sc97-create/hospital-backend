package appointments

import (
	"fmt"
	"hospital-backend/internal/appointments/dto"
	"hospital-backend/shared/params"

	errWrap "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
)

type IAppointment interface {
	CreateAppointment(c *fiber.Ctx) (err error)
	GetSlots(c *fiber.Ctx) (err error)
	FindManyByOrganisationID(c *fiber.Ctx) (err error)
	FindAppointmentsPreview(c *fiber.Ctx) (err error)
	UpdateStatus(c *fiber.Ctx) (err error)
	GetAppointmentByPatientID(c *fiber.Ctx) (err error)
}

type AppointmentController struct {
	AppntmentService *AppointmentService
}

func NewAppointmentController(appointmentSrv *AppointmentService) AppointmentController {
	return AppointmentController{AppntmentService: appointmentSrv}
}

/*
testing
creating internal api for organisation schedule tomorrow
*/

func (A *AppointmentController) CreateAppointment(c *fiber.Ctx) error {
	payload, err := params.New(c)
	if err != nil {
		return errWrap.Wrap(err, c, 500)
	}
	var requestModel dto.NewApptmnt
	requestModel.PatientID, err = payload.Getstring("patient_id")
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	requestModel.UserID, err = payload.Getstring("user_id")
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	requestModel.Notes, _ = payload.Getstring("notes")

	requestModel.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	requestModel.DoctorID, err = payload.Getstring("doctor_id")
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	requestModel.StartTime, err = payload.Getstring("start_time")
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	requestModel.EndTime, err = payload.Getstring("end_time")
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	requestModel.AppointmentDate, err = payload.Getstring("appointment_date")
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	requestModel.ReasonForVisit, err = payload.Getstring("visit_type")
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	requestModel.SeriesID, _ = payload.Getstring("series_id")
	resp, err := A.AppntmentService.CreateApptmnt(requestModel)
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	return c.Status(200).JSON(resp)
}
func (A *AppointmentController) GetSlots(c *fiber.Ctx) error {
	doctorID := c.Query("doctor_id")
	organisationID := c.Query("organisation_id")
	date := c.Query("date")
	err := A.validateIDs(doctorID, organisationID)
	if err != nil {
		return errWrap.Wrap(err, c, 400)
	}
	slotResponse, err := A.AppntmentService.GetSlots(doctorID, organisationID, date)
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	return c.Status(200).JSON(slotResponse)
}
func (c *AppointmentController) validateIDs(args ...any) (err error) {
	if len(args) == 0 {
		return nil
	}
	for _, each := range args {
		switch each.(type) {
		case string:
			if each == "" {
				return fmt.Errorf("%s is not passed, please pass the required query value", each)
			}
		}
	}
	return nil
}
func (A *AppointmentController) FindManyByOrganisationID(c *fiber.Ctx) (err error) {
	payload, err := params.New(c)
	if err != nil {
		return errWrap.Wrap(err, c, 500)
	}
	var reqModel dto.GetDataReq
	reqModel.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	reqModel.Limit, err = payload.Getfloat("limit")
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	reqModel.PageNo, err = payload.Getfloat("page_no")
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	reqModel.DoctorID, _ = payload.Getstring("doctor_id")
	reqModel.Date, _ = payload.Getstring("date")
	reqModel.Status, _ = payload.Getstring("status")
	reqModel.VisitType, _ = payload.Getstring("visit_type")
	err = A.validateIDs(reqModel.OrganisationID)
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	resp, total, err := A.AppntmentService.GetAppointmentsByOrgID(reqModel)
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	var response dto.Response
	response.Data = resp
	response.Total = total
	response.Code = StatusOk
	response.Message = AppointmentData
	return c.Status(200).JSON(response)
}
func (A *AppointmentController) FindAppointmentsPreview(c *fiber.Ctx) (err error) {
	organisationID := c.Query("organisation_id")
	appointmentID := c.Query("appointment_id")
	err = A.validateIDs(organisationID, appointmentID)
	if err != nil {
		return errWrap.Wrap(err, c, 400)
	}
	appointmentDetails, err := A.AppntmentService.GetAppointmentPreview(organisationID, appointmentID)
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	return c.Status(200).JSON(dto.Response{
		Data:    appointmentDetails,
		Message: "appointment preview data retrieved successfully",
		Code:    StatusOk,
	})
}
func (A *AppointmentController) UpdateStatus(c *fiber.Ctx) (err error) {
	payload, err := params.New(c)
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	var updateReq dto.UpdateStatus
	updateReq.AppointmentID, err = payload.Getstring("appointment_id")
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	updateReq.Status, err = payload.Getstring("status")
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	err = A.AppntmentService.UpdateStatus(updateReq)
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	var resp dto.Response
	resp.Code = "200"
	resp.Message = "updated status"
	return c.Status(200).JSON(resp)
}
func (A *AppointmentController) GetAppointmentByPatientID(c *fiber.Ctx) (err error) {
	payload, err := params.New(c)
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	var requestModel dto.PatientAppntment
	requestModel.PatientID, err = payload.Getstring("patient_id")
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	requestModel.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	requestModel.Limit, err = payload.Getfloat("limit")
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	requestModel.Pageno, err = payload.Getfloat("page_no")
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	requestModel.Status, _ = payload.Getstring("status")
	responses, err := A.AppntmentService.GetAppointmentByPatientID(requestModel)
	if err != nil {
		return errWrap.Wrap(err, c, 409)
	}
	return c.Status(200).JSON(responses)
}
