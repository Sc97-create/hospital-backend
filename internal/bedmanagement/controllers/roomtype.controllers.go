package controllers

import (
	"errors"
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
	CreateRoomTypeController(c *fiber.Ctx) (err error)
	GetRoomTypeData(c *fiber.Ctx) (err error)
	FindAllRoomTypes(c *fiber.Ctx) (err error)
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
	//check if length of name should be greater then 3
	if payloadReq.Name != "" && len(payloadReq.Name) < 3 {
		err = errors.New("room type name should be greater then 3 characters")
		return errwrap.Wrap(err, c, 409)
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
	payloadReq.BasePrice, err = payload.Getfloat("base_price")
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	roomTypeResponse, err := i.RoomTypeService.CreateRoomTypeSrv(payloadReq)
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}
	return c.JSON(fiber.Map{"data": roomTypeResponse, "code": 200, "message": "Room Type Created Successfully"})
}
func (i *IRoomTypeModel) GetRoomTypeData(c *fiber.Ctx) (err error) {
	roomTypeId := c.Query("id")
	if roomTypeId == "" {
		err = errors.New("room type id is required")
		return errwrap.Wrap(err, c, 409)

	}
	roomTypeResponse, err := i.RoomTypeService.GetRoomTypeData(roomTypeId)
	if err != nil {
		errwrap.Wrap(err, c, 409)
		return
	}
	return c.JSON(fiber.Map{"data": roomTypeResponse, "code": 200, "message": "Room Type Data Fetched Successfully"})
}
func (i *IRoomTypeModel) FindAllRoomTypes(c *fiber.Ctx) (err error) {
	organisationID := c.Query("organisation_id")
	if organisationID == "" {
		err = errors.New("organisation id is required")
		return errwrap.Wrap(err, c, 409)

	}
	roomTypes, err := i.RoomTypeService.FindAllRoomTypes(organisationID)
	if err != nil {
		return errwrap.Wrap(err, c, 409)

	}
	return c.JSON(fiber.Map{"data": roomTypes, "code": 200, "message": "Room Types Data Fetched Successfully"})
}
