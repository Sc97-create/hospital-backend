package controllers

import (
	"hospital-backend/internal/bedmanagement/dto"
	"hospital-backend/internal/bedmanagement/services"
	"hospital-backend/shared/params"

	errwrap "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
)

// createroomtype
// updateroomtype
type IRoomTypeModel struct {
	RoomTypeService *services.RoomTypeService
}
type IRoomTypeController interface {
	CreateRoomTypeController() (err error)
}

func NewRoomtypeControllerInterface(roomTypeService *services.RoomTypeService) *IRoomTypeModel {
	return &IRoomTypeModel{
		RoomTypeService: roomTypeService,
	}
}

func (i *IRoomTypeModel) CreateRoomTypeController(c *fiber.Ctx) (err error) {
	payloadReq := dto.RoomTypeInfo{}
	payload, err := params.New(c)
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadReq.Name, err = payload.Getstring("name")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadReq.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadReq.IsDefault, err = payload.GetBool("is_default")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	payloadReq.BasePrice, err = payload.Getstring("base_price")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	err = i.RoomTypeService.CreateRoomTypeSrv(payloadReq)
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	return
}
