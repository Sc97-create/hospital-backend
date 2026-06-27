package medicine

import (
	"errors"
	"hospital-backend/internal/medicine/dto"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type IMedicine interface {
	GetByIDHandler(c *fiber.Ctx) (err error)
	GetAllHandler(c *fiber.Ctx) (err error)
	SearchMedicine(c *fiber.Ctx) (err error)
	AddMedicine(c *fiber.Ctx) (err error)
}

type MedicineController struct {
	Service *MedicineService
}

func NewMedicineController(service *MedicineService) *MedicineController {
	return &MedicineController{Service: service}
}

func (MedCo *MedicineController) GetByIDHandler(c *fiber.Ctx) (err error) {
	//medicineID := c.Query("id")

	// medicine, err := service.GetOne(medicineID)
	// if err != nil {
	// 	return wrapError.Wrap(err, c, 409)
	// }
	resp := make(map[string]interface{})
	resp["code"] = 200
	resp["data"] = "medicine"
	err = c.JSON(resp)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	return
}
func (MedCo *MedicineController) GetAllHandler(c *fiber.Ctx) (err error) {
	param, err := params.New(c)
	if err != nil {
		wrapError.Wrap(err, c, 409)
		return
	}
	_, err = param.Getint("limit")
	if err != nil {
		wrapError.Wrap(err, c, 409)
		return
	}
	_, err = param.Getint("page_no")
	if err != nil {
		wrapError.Wrap(err, c, 409)
		return
	}
	// med, err := service.GetMany(limit, pageno)
	// if err != nil {
	// 	return wrapError.Wrap(err, c, 409)
	// }
	response := make(map[string]interface{})
	response["data"] = "medicine"
	response["code"] = 200
	err = c.JSON(&response)
	if err != nil {
		wrapError.Wrap(err, c, 409)
		return
	}
	return
}
func (MedCo *MedicineController) SearchMedicine(c *fiber.Ctx) (err error) {
	name := c.Query("name")
	if name == "" {
		return wrapError.Wrap(errors.New("name is required"), c, 409)
	}
	med, err := MedCo.Service.SearchMedicine(name)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	resp := make(map[string]interface{})
	resp["code"] = 200
	resp["data"] = med
	return c.JSON(resp)
}
func (MedCo *MedicineController) AddMedicine(c *fiber.Ctx) (err error) {
	payload, err := params.New(c)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	var reqPayload dto.RequestPayload
	reqPayload.UserID, err = payload.Getstring("user_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	reqPayload.SupplierID, err = payload.Getstring("supplier_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	reqPayload.PaymentDueDate, _ = payload.Getstring("payment_due_date")
	reqPayload.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	reqPayload.InvoiceNo, err = payload.Getstring("invoice_no")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}

	medicineArray, err := payload.GetChildren("medicine_info")
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	reqPayload.MedicineArray, err = MedCo.toMedicineInfo(medicineArray)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	err = MedCo.Service.CreateMedicine(reqPayload)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	return c.Status(200).JSON(fiber.Map{"message": "created successfully"})
	//store in medicines, medicines_inventory and medicine stock movement
}
func (Medco *MedicineController) toMedicineInfo(medicineArray []*params.Payload) ([]dto.MedicineInfo, error) {
	var medicineInfos []dto.MedicineInfo
	for _, medicine := range medicineArray {
		var medInfo dto.MedicineInfo
		medicineID, _ := medicine.Getstring("medicine_id")
		medInfo.MedicineID = medicineID
		if medicineID == "" {
			medInfo.MedicineID = uuid.NewString()
			medInfo.Add = true
		}
		medInfo.MedInventoryID = uuid.NewString()
		medInfo.Name, _ = medicine.Getstring("name")
		medInfo.Form, _ = medicine.Getstring("form")
		medInfo.Strength, _ = medicine.Getstring("strength")
		medInfo.BatchNumber, _ = medicine.Getstring("batch_no")
		medInfo.ExpiryDate, _ = medicine.Getstring("expiry_date")
		medInfo.Quantity, _ = medicine.Getfloat("quantity")
		medInfo.MRP, _ = medicine.Getfloat("mrp")
		medInfo.Discount, _ = medicine.Getfloat("discount")
		medInfo.PurchasePrice, _ = medicine.Getfloat("purchase_price")
		medInfo.SellingPrice, _ = medicine.Getfloat("selling_price")
		if medInfo.SellingPrice == 0 {
			medInfo.SellingPrice = medInfo.MRP
		}
		medInfo.HsnCode, _ = medicine.Getstring("hsn_code")
		medInfo.ReorderLevel, _ = medicine.Getint("reorder_level")
		medInfo.MaxStockTarget, _ = medicine.Getint("max_stock_target")
		medInfo.PurchaseQtyBoxes, _ = medicine.Getint("purchase_qty_boxes")
		medInfo.UnitPerBoxes, _ = medicine.Getint("units_per_box")
		medInfo.ShelfLocation, _ = medicine.Getstring("shelf_location")
		medicineInfos = append(medicineInfos, medInfo)
	}
	return medicineInfos, nil
}
