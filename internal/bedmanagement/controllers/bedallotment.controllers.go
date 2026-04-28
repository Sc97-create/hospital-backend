package controllers

import (
	"hospital-backend/internal/bedmanagement/dto"
	"hospital-backend/internal/bedmanagement/services"
	errwrap "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
)

type BedAllotmentController struct {
	BedAllotmentService *services.BedAllotmentService
}
type IBedAllotmentController interface {
	CreateBedAllotmentController(c *fiber.Ctx) (err error)
}

func NewBedAllotmentController(bedAllotmentService *services.BedAllotmentService) *BedAllotmentController {
	return &BedAllotmentController{BedAllotmentService: bedAllotmentService}
}

func (i *BedAllotmentController) CreateBedAllotmentController(c *fiber.Ctx) (err error) {
	payload, err := params.New(c)
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}
	bedAllotment := dto.BedAllotmentCreatePayload{}
	bedAllotment.PatientID, err = payload.Getstring("patient_id")
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}
	bedAllotment.BedID, err = payload.Getstring("bed_id")
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}
	bedAllotment.RoomID, err = payload.Getstring("room_id")
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}
	bedAllotment.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}
	bedAllotment.RoomTypeID, err = payload.Getstring("room_type")
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}
	bedAllotment.BedCharges, err = payload.Getfloat("charges")
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}
	bedAllotment.DischargeAt, err = payload.GetTime("discharge_at")
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}
	//bedAllotment.IsEmergency, _ = payload.GetBool("is_emergency")

	err = i.BedAllotmentService.CreateBedAllotment(bedAllotment)
	if err != nil {
		return errwrap.Wrap(err, c, 409)
	}
	return c.JSON(fiber.Map{"data": bedAllotment, "code": 200, "message": "Bed Allotment Created Successfully"})
}
