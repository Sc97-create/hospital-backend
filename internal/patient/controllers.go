package patient

import (
	"errors"
	"hospital-backend/internal/patient/dto"
	"hospital-backend/pkg/middleware"
	"hospital-backend/shared/params"

	errwrap "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type IPatientController interface {
	AddGeneralInfoHandler(c *fiber.Ctx) (err error)
	Find(c *fiber.Ctx) (err error)
	GetPatientByID(c *fiber.Ctx) (err error)
}

type PatientServicer interface {
	CreatePatientSrv(log *zap.Logger, payload dto.PatientInfo) (string, error)
	FindOne(log *zap.Logger, id string) (dto.PatientResponse, error)
	FindMany(log *zap.Logger, req dto.PatientListReq) ([]dto.PatientResponse, int64, error)
}

type PatientController struct {
	PatientService PatientServicer
}

func NewPatientControllerInterface(service PatientServicer) IPatientController {
	return &PatientController{PatientService: service}
}

func (p *PatientController) AddGeneralInfoHandler(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)

	params, err := params.New(c)
	if err != nil {
		logger.Warn("patient create request invalid", zap.Error(err))
		return errwrap.Wrap(errwrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	var payloadModel dto.PatientInfo
	var field string
	payloadModel, field, err = p.ToPatientModel(params)
	if err != nil {
		logger.Warn("patient create request invalid", zap.Error(err), zap.String("field", field))
		return errwrap.Wrap(errwrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("patient create attempt",
		zap.String("organisation_id", payloadModel.OrganisationID),
		zap.String("created_by", payloadModel.UserID),
	)

	id, err := p.PatientService.CreatePatientSrv(logger, payloadModel)
	if err != nil {
		return p.wrapCreateError(c, err)
	}

	res := make(map[string]interface{})
	res["message"] = "general info added"
	res["patient_id"] = id
	res["code"] = 200
	return c.Status(fiber.StatusOK).JSON(res)
}

func (p *PatientController) wrapCreateError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, errwrap.ErrOrganisationNotFound):
		return errwrap.Wrap(err, c, fiber.StatusNotFound)
	case errors.Is(err, errwrap.ErrPatientAlreadyExists):
		return errwrap.Wrap(err, c, fiber.StatusConflict)
	case errors.Is(err, errwrap.ErrPatientCreateFailed):
		return errwrap.Wrap(err, c, fiber.StatusInternalServerError)
	default:
		var ve *validationError
		if errors.As(err, &ve) {
			return errwrap.Wrap(err, c, fiber.StatusBadRequest)
		}
		return errwrap.Wrap(errwrap.ErrPatientCreateFailed, c, fiber.StatusInternalServerError)
	}
}

func (p *PatientController) ToPatientModel(params *params.Payload) (payloadModel dto.PatientInfo, field string, err error) {
	payloadModel.Name, err = params.Getstring("name")
	if err != nil {
		return payloadModel, "name", err
	}
	payloadModel.BloodGroup, err = params.Getstring("blood_group")
	if err != nil {
		return payloadModel, "blood_group", err
	}
	payloadModel.Address, err = params.Getstring("address")
	if err != nil {
		return payloadModel, "address", err
	}
	payloadModel.Age, err = params.Getstring("age")
	if err != nil {
		return payloadModel, "age", err
	}
	payloadModel.UserID, err = params.Getstring("user_id")
	if err != nil {
		return payloadModel, "user_id", err
	}
	payloadModel.Weight, err = params.Getstring("weight")
	if err != nil {
		return payloadModel, "weight", err
	}
	payloadModel.Gender, err = params.Getstring("gender")
	if err != nil {
		return payloadModel, "gender", err
	}
	payloadModel.OrganisationID, err = params.Getstring("organisation_id")
	if err != nil {
		return payloadModel, "organisation_id", err
	}
	payloadModel.EmailID, err = params.Getstring("email_id")
	if err != nil {
		return payloadModel, "email_id", err
	}
	payloadModel.MobileNumber, err = params.Getstring("mobile_number")
	if err != nil {
		return payloadModel, "mobile_number", err
	}
	return payloadModel, "", nil
}

func (p *PatientController) GetPatientByID(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)
	patientID := c.Params("patientID")
	if patientID == "" {
		logger.Warn("patient get request invalid", zap.String("reason", "missing_patient_id"))
		return errwrap.Wrap(errwrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("patient get attempt", zap.String("patient_id", patientID))

	patient, err := p.PatientService.FindOne(logger, patientID)
	if err != nil {
		if errors.Is(err, errwrap.ErrPatientNotFound) {
			return errwrap.Wrap(err, c, fiber.StatusNotFound)
		}
		return errwrap.Wrap(err, c, fiber.StatusInternalServerError)
	}

	res := make(map[string]interface{})
	res["data"] = patient
	res["code"] = 200
	if err = c.Status(fiber.StatusOK).JSON(&res); err != nil {
		logger.Error("patient get response failed", zap.String("patient_id", patientID), zap.Error(err))
		return errwrap.Wrap(err, c, fiber.StatusInternalServerError)
	}
	return
}

func (p *PatientController) Find(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)
	payload, err := params.New(c)
	if err != nil {
		logger.Warn("patient list request invalid", zap.Error(err))
		return errwrap.Wrap(errwrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	var req dto.PatientListReq
	req.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil || req.OrganisationID == "" {
		logger.Warn("patient list request invalid", zap.String("field", "organisation_id"))
		return errwrap.Wrap(errwrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	req.Limit, err = payload.Getfloat("limit")
	if err != nil {
		logger.Warn("patient list request invalid", zap.String("field", "limit"))
		return errwrap.Wrap(errwrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	req.PageNo, err = payload.Getfloat("page_no")
	if err != nil {
		logger.Warn("patient list request invalid", zap.String("field", "page_no"))
		return errwrap.Wrap(errwrap.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	req.Search, _ = payload.Getstring("search")

	logger.Info("patient list attempt",
		zap.String("organisation_id", req.OrganisationID),
		zap.Float64("limit", req.Limit),
		zap.Float64("page_no", req.PageNo),
		zap.Bool("has_search", req.Search != ""),
	)

	patient, total, err := p.PatientService.FindMany(logger, req)
	if err != nil {
		return errwrap.Wrap(err, c, fiber.StatusInternalServerError)
	}

	var response dto.PatientListResponse
	response.Data = patient
	response.Total = total
	response.Code = 200
	if err = c.Status(fiber.StatusOK).JSON(&response); err != nil {
		logger.Error("patient list response failed",
			zap.String("organisation_id", req.OrganisationID),
			zap.Error(err),
		)
		return errwrap.Wrap(err, c, fiber.StatusInternalServerError)
	}
	return
}
