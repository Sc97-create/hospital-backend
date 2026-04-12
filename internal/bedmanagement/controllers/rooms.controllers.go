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
	payloadReq.RoomTypeID, err = payload.Getstring("room_type_id")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadReq.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadReq.Floor, err = payload.Getfloat("no_of_floors")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadReq.RoomPerFloor, err = payload.Getfloat("room_per_floor")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadReq.StartingPerFloor, err = payload.Getfloat("starting_per_floor")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadReq.Prefix, err = payload.Getstring("prefix")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	arrayRooms, err := i.RoomService.CreateBatchRooms(payloadReq)
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	return c.JSON(fiber.Map{"data": arrayRooms, "code": 200})
}

// func (i *IRoomModel) GetRoomsByIDController(c *fiber.Ctx) (err error) {
// 	roomID := c.Params("roomID")
// 	room, err := i.RoomService.GetRoomByID(roomID)
// 	if err != nil {
// 		errwrap.Wrap(err, c, 409)
// 		return
// 	}
// 	return c.JSON(fiber.Map{"data": room, "code": 200})
// }
