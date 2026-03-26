package controllers

import (
	"hospital-backend/internal/bedmanagement/dto"
	"hospital-backend/internal/bedmanagement/services"
	errwrap "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
)

type IRoomModel struct {
	RoomService *services.RoomService
}
type IRoomController interface {
	CreateRoomController(c *fiber.Ctx) (err error)
}

func NewRoomControllerInterface(roomService *services.RoomService) *IRoomModel {
	return &IRoomModel{
		RoomService: roomService,
	}
}

func (i *IRoomModel) CreateRoomController(c *fiber.Ctx) (err error) {
	payloadReq := dto.RoomRequest{}
	payload, err := params.New(c)
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadReq.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadReq.Floor, err = payload.Getint("no_of_floor")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadReq.RoomPerFloor, err = payload.Getint("room_per_floor")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadReq.StartingPerFloor, err = payload.Getint("starting_per_floor")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadReq.Prefix, err = payload.Getstring("prefix")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	err = i.RoomService.CreateBatchRooms(payloadReq)
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	return
}
