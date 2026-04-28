package controllers

import (
	"errors"
	"hospital-backend/internal/bedmanagement/dto"
	"hospital-backend/internal/bedmanagement/services"
	errwrap "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
)

//createbeds
//getbeds
//updatebeds

type IBedModel struct {
	BedService *services.BedService
}
type IBedController interface {
	CreateBedController(c *fiber.Ctx) (err error)
	GenerateBeds(c *fiber.Ctx) (err error)
	BedSummary(c *fiber.Ctx) (err error)
	FindAllAvailableBeds(c *fiber.Ctx) (err error)
}

func NewBedControllerInterface(bedService *services.BedService) *IBedModel {
	return &IBedModel{
		BedService: bedService,
	}
}

func (i *IBedModel) CreateBedController(c *fiber.Ctx) (err error) {
	payloadReq := dto.CreateBed{}
	payload, err := params.New(c)
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadReq.RoomTypeID, err = payload.Getstring("room_type_id")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	chilPayload, err := payload.GetChildren("beds")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadReq.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	if len(chilPayload) == 0 {
		err = errors.New("beds is required")
		errwrap.Wrap(err, c, 409)
		return
	}
	for _, each := range chilPayload {
		bedArray, _ := each.GetStringArray("beds")
		roomId, _ := each.Getstring("room_id")
		payloadReq.Beds = append(payloadReq.Beds,
			dto.ReqBed{
				RoomID:    roomId,
				BedsArray: bedArray,
			})

	}

	err = i.BedService.CreateBedSrv(payloadReq)
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	return c.JSON(fiber.Map{"message": "created bed successfully", "code": 200})
}
func (i *IBedModel) GenerateBeds(c *fiber.Ctx) (err error) {
	payloadReq := dto.BedGenerate{}
	payload, err := params.New(c)
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadReq.BedsPerRoom, err = payload.Getint("beds_per_room")
	if err != nil {
		return errwrap.Wrap(err, c, 409)

	}
	payloadReq.RoomNumber, err = payload.GetStringArray("room_number")
	if err != nil {
		return errwrap.Wrap(err, c, 409)

	}
	payloadReq.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		return errwrap.Wrap(err, c, 409)

	}
	payloadReq.RoomTypeID, err = payload.Getstring("room_type_id")
	if err != nil {
		return errwrap.Wrap(err, c, 409)

	}
	bedResp, roomSummary, err := i.BedService.GenerateBeds(payloadReq)
	if err != nil {
		return errwrap.Wrap(err, c, 409)

	}
	return c.JSON(fiber.Map{"data": bedResp, "room_summary": roomSummary, "code": 200})
}
func (i *IBedModel) BedSummary(c *fiber.Ctx) (err error) {

	return
}
func (i *IBedModel) FindAllAvailableBeds(c *fiber.Ctx) (err error) {
	organisationID := c.Query("organisation_id")
	roomID := c.Query("room_id")
	limit := c.Query("limit")
	offset := c.Query("pageno")
	err = i.ValidateFindManyBedRequest(organisationID, limit, offset)
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}
	beds, err := i.BedService.FindAllAvailableBeds(organisationID, limit, offset, roomID)
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}
	return c.JSON(fiber.Map{"data": beds, "code": 200})
}
func (i *IBedModel) ValidateFindManyBedRequest(organisationID string, limit string, offset string) error {
	if organisationID == "" {
		return errors.New("organisation id is required")
	}
	if limit == "" {
		return errors.New("limit is required")
	}
	if offset == "" {
		return errors.New("offset is required")
	}
	return nil
}
