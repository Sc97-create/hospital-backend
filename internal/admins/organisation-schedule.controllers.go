package admins

import (
	"hospital-backend/internal/admins/dto"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
)

type IOrgSchedController struct {
	OrgSchedService *OrganisationScheduleService
}

type IOrgSched interface {
	CreateOrgSched(c *fiber.Ctx) error
}

func NewOrgSchedController(OrgSchedService *OrganisationScheduleService) IOrgSchedController {
	return IOrgSchedController{OrgSchedService: OrgSchedService}
}

func (O *IOrgSchedController) Create(c *fiber.Ctx) error {
	payload, err := params.New(c)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	var orgSched dto.OrgScheduleReq
	orgSched.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	orgSched.StartTime, err = payload.Getstring("start_time")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	orgSched.EndTime, err = payload.Getstring("end_time")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	orgSched.SlotDuration, err = payload.Getfloat("time_slot")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	orgSched.BreakStartTime, err = payload.Getstring("break_start_time")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	orgSched.BreakEndTime, err = payload.Getstring("break_end_time")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	orgSched.WeekDays, err = payload.GetStringArray("week_offs")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	orgSched.IsClosed, _ = payload.GetBool("is_closed")
	err = O.OrgSchedService.Create(orgSched)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	var responseModel dto.FormatResponse
	responseModel.Code = StatusOk
	responseModel.Message = OrgSchedCreated
	return c.Status(200).JSON(responseModel)
}
