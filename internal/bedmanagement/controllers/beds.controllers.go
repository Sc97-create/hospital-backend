package controllers

import (
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
}

func NewBedControllerInterface(bedService *services.BedService) *IBedModel {
	return &IBedModel{
		BedService: bedService,
	}
}

func (i *IBedModel) CreateBedController(c *fiber.Ctx) (err error) {
	payloadReq := dto.BedInfo{}
	payload, err := params.New(c)
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadReq.BedsPerRoom, err = payload.Getint("beds_per_room")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadReq.RoomNumber, err = payload.GetStringArray("room_number")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadReq.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	err = i.BedService.CreateBedSrv(payloadReq)
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	return
}
