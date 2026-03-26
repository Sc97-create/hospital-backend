package medicine

import (
	"hospital-backend/internal/medicine/dto"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
)

func CreateHandler(c *fiber.Ctx, service *MedicineService) (err error) {
	payload, err := params.New(c)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	reqPayload := dto.RequestPayload{}

	reqPayload.Name, err = payload.Getstring("name")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	reqPayload.Form, err = payload.Getstring("form")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	reqPayload.Strength, err = payload.Getstring("strength")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	reqPayload.BatchNumber, err = payload.Getstring("batch_number")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	reqPayload.ExpiryDate, err = payload.GetTime("expiry_date")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	reqPayload.Quantity, err = payload.GetInt64("quantity")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	err = service.CreateMedicalSrv(reqPayload)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}

	resp := make(map[string]interface{})
	resp["code"] = 200
	resp["message"] = "medicine added successfully"
	err = c.JSON(resp)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	return
}
func GetByIDHandler(c *fiber.Ctx, service *MedicineService) (err error) {
	medicineID := c.Query("id")

	medicine, err := service.GetOne(medicineID)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	resp := make(map[string]interface{})
	resp["code"] = 200
	resp["data"] = medicine
	err = c.JSON(resp)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	return
}
func GetAllHandler(c *fiber.Ctx, service *MedicineService) (err error) {
	param, err := params.New(c)
	if err != nil {
		wrapError.Wrap(err, c, 409)
		return
	}
	limit, err := param.Getint("limit")
	if err != nil {
		wrapError.Wrap(err, c, 409)
		return
	}
	pageno, err := param.Getint("page_no")
	if err != nil {
		wrapError.Wrap(err, c, 409)
		return
	}
	med, err := service.GetMany(limit, pageno)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	response := make(map[string]interface{})
	response["data"] = med
	response["code"] = 200
	err = c.JSON(&response)
	if err != nil {
		wrapError.Wrap(err, c, 409)
		return
	}
	return
}
