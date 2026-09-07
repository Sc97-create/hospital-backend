package admins

import (
	"errors"
	"hospital-backend/internal/admins/dto"
	"hospital-backend/pkg/middleware"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type OrganisationScheduleServicer interface {
	Create(log *zap.Logger, reqModel dto.OrgScheduleReq) error
	GetScheduleByOrganisationID(log *zap.Logger, organisationID string) (dto.GetResponse, error)
}

type IOrgSchedController struct {
	OrgSchedService OrganisationScheduleServicer
}

type IOrgSched interface {
	CreateOrgSched(c *fiber.Ctx) error
}

func NewOrgSchedController(OrgSchedService OrganisationScheduleServicer) IOrgSchedController {
	return IOrgSchedController{OrgSchedService: OrgSchedService}
}

func (O *IOrgSchedController) Create(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)

	payload, err := params.New(c)
	if err != nil {
		logger.Warn("organisation schedule create request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	var orgSched dto.OrgScheduleReq
	orgSched.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil || strings.TrimSpace(orgSched.OrganisationID) == "" {
		logger.Warn("organisation schedule create request invalid", zap.String("field", "organisation_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	orgSched.StartTime, err = payload.Getstring("start_time")
	if err != nil || strings.TrimSpace(orgSched.StartTime) == "" {
		logger.Warn("organisation schedule create request invalid", zap.String("field", "start_time"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	orgSched.EndTime, err = payload.Getstring("end_time")
	if err != nil || strings.TrimSpace(orgSched.EndTime) == "" {
		logger.Warn("organisation schedule create request invalid", zap.String("field", "end_time"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	orgSched.SlotDuration, err = payload.Getfloat("time_slot")
	if err != nil || orgSched.SlotDuration <= 0 {
		logger.Warn("organisation schedule create request invalid", zap.String("field", "time_slot"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	orgSched.BreakStartTime, err = payload.Getstring("break_start_time")
	if err != nil || strings.TrimSpace(orgSched.BreakStartTime) == "" {
		logger.Warn("organisation schedule create request invalid", zap.String("field", "break_start_time"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	orgSched.BreakEndTime, err = payload.Getstring("break_end_time")
	if err != nil || strings.TrimSpace(orgSched.BreakEndTime) == "" {
		logger.Warn("organisation schedule create request invalid", zap.String("field", "break_end_time"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	orgSched.WeekDays, err = payload.GetStringArray("week_offs")
	if err != nil {
		logger.Warn("organisation schedule create request invalid", zap.String("field", "week_offs"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	orgSched.IsClosed, _ = payload.GetBool("is_closed")

	logger.Info("organisation schedule create attempt",
		zap.String("organisation_id", orgSched.OrganisationID),
		zap.Int("slot_duration", int(orgSched.SlotDuration)),
		zap.Int("week_off_count", len(orgSched.WeekDays)),
		zap.Bool("is_closed", orgSched.IsClosed),
	)

	err = O.OrgSchedService.Create(logger, orgSched)
	if err != nil {
		return wrapCreateError(c, err)
	}
	var responseModel dto.FormatResponse
	responseModel.Code = StatusOk
	responseModel.Message = OrgSchedCreated
	return c.Status(fiber.StatusOK).JSON(responseModel)
}

func wrapCreateError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, c, fiber.StatusBadRequest)
	default:
		return wrapError.Wrap(wrapError.ErrOrgScheduleCreateFailed, c, fiber.StatusInternalServerError)
	}
}
