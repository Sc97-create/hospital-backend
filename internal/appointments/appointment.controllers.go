package appointments

import (
	"errors"
	"fmt"
	"hospital-backend/internal/appointments/dto"
	"hospital-backend/pkg/middleware"
	"hospital-backend/shared/params"

	errWrap "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
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

func (A *AppointmentController) CreateAppointment(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)

	payload, err := params.New(c)
	if err != nil {
		logger.Warn("appointment create request invalid", zap.Error(err))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	var requestModel dto.NewApptmnt
	var field string
	requestModel, field, err = A.toCreateRequest(payload)
	if err != nil {
		logger.Warn("appointment create request invalid", zap.Error(err), zap.String("field", field))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("appointment create attempt",
		zap.String("organisation_id", requestModel.OrganisationID),
		zap.String("patient_id", requestModel.PatientID),
		zap.String("doctor_id", requestModel.DoctorID),
		zap.String("created_by", requestModel.UserID),
		zap.String("appointment_date", requestModel.AppointmentDate),
		zap.String("visit_type", requestModel.ReasonForVisit),
		zap.Bool("has_series_id", requestModel.SeriesID != ""),
	)

	resp, err := A.AppntmentService.CreateApptmnt(logger, requestModel)
	if err != nil {
		return A.wrapCreateError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

func (A *AppointmentController) toCreateRequest(payload *params.Payload) (requestModel dto.NewApptmnt, field string, err error) {
	requestModel.PatientID, err = payload.Getstring("patient_id")
	if err != nil {
		return requestModel, "patient_id", err
	}
	requestModel.UserID, err = payload.Getstring("user_id")
	if err != nil {
		return requestModel, "user_id", err
	}
	requestModel.Notes, _ = payload.Getstring("notes")

	requestModel.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		return requestModel, "organisation_id", err
	}
	requestModel.DoctorID, err = payload.Getstring("doctor_id")
	if err != nil {
		return requestModel, "doctor_id", err
	}
	requestModel.StartTime, err = payload.Getstring("start_time")
	if err != nil {
		return requestModel, "start_time", err
	}
	requestModel.EndTime, err = payload.Getstring("end_time")
	if err != nil {
		return requestModel, "end_time", err
	}
	requestModel.AppointmentDate, err = payload.Getstring("appointment_date")
	if err != nil {
		return requestModel, "appointment_date", err
	}
	requestModel.ReasonForVisit, err = payload.Getstring("visit_type")
	if err != nil {
		return requestModel, "visit_type", err
	}
	requestModel.SeriesID, _ = payload.Getstring("series_id")
	return requestModel, "", nil
}

func (A *AppointmentController) wrapCreateError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, errWrap.ErrOrgScheduleNotFound):
		return errWrap.Wrap(err, c, fiber.StatusNotFound)
	case errors.Is(err, errWrap.ErrAppointmentCreateFailed):
		return errWrap.Wrap(err, c, fiber.StatusInternalServerError)
	default:
		var ve *validationError
		if errors.As(err, &ve) {
			return errWrap.Wrap(err, c, fiber.StatusBadRequest)
		}
		return errWrap.Wrap(errWrap.ErrAppointmentCreateFailed, c, fiber.StatusInternalServerError)
	}
}

func (A *AppointmentController) GetSlots(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	doctorID := c.Query("doctor_id")
	organisationID := c.Query("organisation_id")
	date := c.Query("date")

	if doctorID == "" {
		logger.Warn("appointment slots request invalid", zap.String("reason", "missing_doctor_id"))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	if organisationID == "" {
		logger.Warn("appointment slots request invalid", zap.String("reason", "missing_organisation_id"))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("appointment slots attempt",
		zap.String("doctor_id", doctorID),
		zap.String("organisation_id", organisationID),
		zap.String("date", date),
	)

	slotResponse, err := A.AppntmentService.GetSlots(logger, doctorID, organisationID, date)
	if err != nil {
		if errors.Is(err, errWrap.ErrOrgScheduleNotFound) {
			return errWrap.Wrap(err, c, fiber.StatusNotFound)
		}
		return errWrap.Wrap(err, c, fiber.StatusInternalServerError)
	}
	return c.Status(fiber.StatusOK).JSON(slotResponse)
}

func (A *AppointmentController) FindManyByOrganisationID(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)

	payload, err := params.New(c)
	if err != nil {
		logger.Warn("appointment list request invalid", zap.Error(err))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	var reqModel dto.GetDataReq
	reqModel.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		logger.Warn("appointment list request invalid", zap.Error(err), zap.String("field", "organisation_id"))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	reqModel.Limit, err = payload.Getfloat("limit")
	if err != nil {
		logger.Warn("appointment list request invalid", zap.Error(err), zap.String("field", "limit"))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	reqModel.PageNo, err = payload.Getfloat("page_no")
	if err != nil {
		logger.Warn("appointment list request invalid", zap.Error(err), zap.String("field", "page_no"))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	reqModel.DoctorID, _ = payload.Getstring("doctor_id")
	reqModel.Date, _ = payload.Getstring("date")
	reqModel.Status, _ = payload.Getstring("status")
	reqModel.VisitType, _ = payload.Getstring("visit_type")
	reqModel.Search, _ = payload.Getstring("search")

	if reqModel.OrganisationID == "" {
		logger.Warn("appointment list request invalid", zap.String("reason", "missing_organisation_id"))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("appointment list attempt",
		zap.String("organisation_id", reqModel.OrganisationID),
		zap.Float64("limit", reqModel.Limit),
		zap.Float64("page_no", reqModel.PageNo),
		zap.String("doctor_id", reqModel.DoctorID),
		zap.String("date", reqModel.Date),
		zap.String("status", reqModel.Status),
		zap.String("visit_type", reqModel.VisitType),
		zap.Bool("has_search", reqModel.Search != ""),
	)

	resp, total, err := A.AppntmentService.GetAppointmentsByOrgID(logger, reqModel)
	if err != nil {
		return errWrap.Wrap(err, c, fiber.StatusInternalServerError)
	}
	var response dto.Response
	response.Data = resp
	response.Total = total
	response.Code = StatusOk
	response.Message = AppointmentData
	return c.Status(fiber.StatusOK).JSON(response)
}

func (A *AppointmentController) FindAppointmentsPreview(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)
	organisationID := c.Query("organisation_id")
	appointmentID := c.Query("appointment_id")

	if organisationID == "" {
		logger.Warn("appointment preview request invalid", zap.String("reason", "missing_organisation_id"))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	if appointmentID == "" {
		logger.Warn("appointment preview request invalid", zap.String("reason", "missing_appointment_id"))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("appointment preview attempt",
		zap.String("organisation_id", organisationID),
		zap.String("appointment_id", appointmentID),
	)

	appointmentDetails, err := A.AppntmentService.GetAppointmentPreview(logger, organisationID, appointmentID)
	if err != nil {
		if errors.Is(err, errWrap.ErrAppointmentNotFound) {
			return errWrap.Wrap(err, c, fiber.StatusNotFound)
		}
		return errWrap.Wrap(err, c, fiber.StatusInternalServerError)
	}
	return c.Status(fiber.StatusOK).JSON(dto.Response{
		Data:    appointmentDetails,
		Message: "appointment preview data retrieved successfully",
		Code:    StatusOk,
	})
}

func (A *AppointmentController) UpdateStatus(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)

	payload, err := params.New(c)
	if err != nil {
		logger.Warn("appointment status update request invalid", zap.Error(err))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	var updateReq dto.UpdateStatus
	updateReq.AppointmentID, err = payload.Getstring("appointment_id")
	if err != nil {
		logger.Warn("appointment status update request invalid", zap.Error(err), zap.String("field", "appointment_id"))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	updateReq.Status, err = payload.Getstring("status")
	if err != nil {
		logger.Warn("appointment status update request invalid", zap.Error(err), zap.String("field", "status"))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("appointment status update attempt",
		zap.String("appointment_id", updateReq.AppointmentID),
		zap.String("status", updateReq.Status),
	)

	err = A.AppntmentService.UpdateStatus(logger, updateReq)
	if err != nil {
		if errors.Is(err, errWrap.ErrInvalidRequest) {
			return errWrap.Wrap(err, c, fiber.StatusBadRequest)
		}
		return errWrap.Wrap(err, c, fiber.StatusInternalServerError)
	}
	var resp dto.Response
	resp.Code = "200"
	resp.Message = "updated status"
	return c.Status(fiber.StatusOK).JSON(resp)
}

func (A *AppointmentController) GetAppointmentByPatientID(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)

	payload, err := params.New(c)
	if err != nil {
		logger.Warn("appointment patient list request invalid", zap.Error(err))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	var requestModel dto.PatientAppntment
	requestModel.PatientID, err = payload.Getstring("patient_id")
	if err != nil {
		logger.Warn("appointment patient list request invalid", zap.Error(err), zap.String("field", "patient_id"))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	requestModel.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		logger.Warn("appointment patient list request invalid", zap.Error(err), zap.String("field", "organisation_id"))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	requestModel.Limit, err = payload.Getfloat("limit")
	if err != nil {
		logger.Warn("appointment patient list request invalid", zap.Error(err), zap.String("field", "limit"))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	requestModel.Pageno, err = payload.Getfloat("page_no")
	if err != nil {
		logger.Warn("appointment patient list request invalid", zap.Error(err), zap.String("field", "page_no"))
		return errWrap.Wrap(errWrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	requestModel.Status, _ = payload.Getstring("status")

	logger.Info("appointment patient list attempt",
		zap.String("patient_id", requestModel.PatientID),
		zap.String("organisation_id", requestModel.OrganisationID),
		zap.Float64("limit", requestModel.Limit),
		zap.Float64("page_no", requestModel.Pageno),
		zap.String("status", requestModel.Status),
	)

	responses, err := A.AppntmentService.GetAppointmentByPatientID(logger, requestModel)
	if err != nil {
		return errWrap.Wrap(err, c, fiber.StatusInternalServerError)
	}
	return c.Status(fiber.StatusOK).JSON(responses)
}

// validateIDs kept for any remaining callers; prefer explicit checks in handlers.
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
