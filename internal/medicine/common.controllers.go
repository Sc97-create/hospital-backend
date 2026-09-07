package medicine

import (
	"errors"
	"hospital-backend/internal/medicine/dto"
	"hospital-backend/pkg/middleware"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IMedicine interface {
	GetByIDHandler(c *fiber.Ctx) (err error)
	GetAllHandler(c *fiber.Ctx) (err error)
	SearchMedicine(c *fiber.Ctx) (err error)
	AddMedicine(c *fiber.Ctx) (err error)
}

type MedicineServicer interface {
	SearchMedicine(log *zap.Logger, name string, organisationID string) ([]dto.SearchMedicineItem, error)
	CreateMedicine(log *zap.Logger, MedicinePayload dto.RequestPayload) error
}

type MedicineController struct {
	Service MedicineServicer
}

func NewMedicineController(service MedicineServicer) *MedicineController {
	return &MedicineController{Service: service}
}

func (MedCo *MedicineController) GetByIDHandler(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)
	logger.Warn("medicine get stub called")
	resp := make(map[string]interface{})
	resp["code"] = 200
	resp["data"] = "medicine"
	return c.JSON(resp)
}

func (MedCo *MedicineController) GetAllHandler(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)
	logger.Warn("medicine list stub called")

	param, err := params.New(c)
	if err != nil {
		logger.Warn("medicine list request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	if _, err = param.Getint("limit"); err != nil {
		logger.Warn("medicine list request invalid", zap.String("field", "limit"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	if _, err = param.Getint("page_no"); err != nil {
		logger.Warn("medicine list request invalid", zap.String("field", "page_no"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	response := make(map[string]interface{})
	response["data"] = "medicine"
	response["code"] = 200
	return c.JSON(&response)
}

func (MedCo *MedicineController) SearchMedicine(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		logger.Warn("medicine search request invalid", zap.String("field", "name"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	organisationID := strings.TrimSpace(c.Query("organisation_id"))
	if organisationID == "" {
		logger.Warn("medicine search request invalid", zap.String("field", "organisation_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("medicine search attempt",
		zap.String("organisation_id", organisationID),
		zap.Int("name_len", len(name)),
	)

	med, err := MedCo.Service.SearchMedicine(logger, name, organisationID)
	if err != nil {
		return wrapError.Wrap(wrapError.ErrMedicineSearchFailed, c, fiber.StatusInternalServerError)
	}
	resp := make(map[string]interface{})
	resp["code"] = 200
	resp["data"] = med
	return c.JSON(resp)
}

func (MedCo *MedicineController) AddMedicine(c *fiber.Ctx) (err error) {
	logger := middleware.GetLogger(c)
	payload, err := params.New(c)
	if err != nil {
		logger.Warn("medicine purchase request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	var reqPayload dto.RequestPayload
	reqPayload.UserID, err = payload.Getstring("user_id")
	if err != nil || strings.TrimSpace(reqPayload.UserID) == "" {
		logger.Warn("medicine purchase request invalid", zap.String("field", "user_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	reqPayload.SupplierID, err = payload.Getstring("supplier_id")
	if err != nil || strings.TrimSpace(reqPayload.SupplierID) == "" {
		logger.Warn("medicine purchase request invalid", zap.String("field", "supplier_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	reqPayload.PaymentDueDate, _ = payload.Getstring("payment_due_date")
	reqPayload.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil || strings.TrimSpace(reqPayload.OrganisationID) == "" {
		logger.Warn("medicine purchase request invalid", zap.String("field", "organisation_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	reqPayload.InvoiceNo, err = payload.Getstring("invoice_no")
	if err != nil || strings.TrimSpace(reqPayload.InvoiceNo) == "" {
		logger.Warn("medicine purchase request invalid", zap.String("field", "invoice_no"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	medicineArray, err := payload.GetChildren("medicine_info")
	if err != nil || len(medicineArray) == 0 {
		logger.Warn("medicine purchase request invalid", zap.String("field", "medicine_info"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	reqPayload.MedicineArray, err = MedCo.toMedicineInfo(logger, medicineArray)
	if err != nil {
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("medicine purchase attempt",
		zap.String("organisation_id", reqPayload.OrganisationID),
		zap.String("supplier_id", reqPayload.SupplierID),
		zap.String("user_id", reqPayload.UserID),
		zap.String("invoice_no", reqPayload.InvoiceNo),
		zap.Int("item_count", len(reqPayload.MedicineArray)),
	)

	err = MedCo.Service.CreateMedicine(logger, reqPayload)
	if err != nil {
		return MedCo.wrapPurchaseError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "created successfully"})
}

func (MedCo *MedicineController) wrapPurchaseError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrSupplierNotFound):
		return wrapError.Wrap(err, c, fiber.StatusNotFound)
	case errors.Is(err, wrapError.ErrInvalidRequest):
		return wrapError.Wrap(err, c, fiber.StatusBadRequest)
	default:
		return wrapError.Wrap(wrapError.ErrMedicinePurchaseFailed, c, fiber.StatusInternalServerError)
	}
}

func (MedCo *MedicineController) toMedicineInfo(log *zap.Logger, medicineArray []*params.Payload) ([]dto.MedicineInfo, error) {
	log = ensureLog(log)
	var medicineInfos []dto.MedicineInfo
	for i, medicine := range medicineArray {
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

		if medInfo.Add && strings.TrimSpace(medInfo.Name) == "" {
			log.Warn("medicine purchase request invalid",
				zap.String("field", "name"),
				zap.Int("item_index", i),
			)
			return nil, wrapError.ErrInvalidRequest
		}
		if medInfo.PurchaseQtyBoxes <= 0 {
			log.Warn("medicine purchase request invalid",
				zap.String("field", "purchase_qty_boxes"),
				zap.Int("item_index", i),
			)
			return nil, wrapError.ErrInvalidRequest
		}
		if medInfo.UnitPerBoxes <= 0 {
			log.Warn("medicine purchase request invalid",
				zap.String("field", "units_per_box"),
				zap.Int("item_index", i),
			)
			return nil, wrapError.ErrInvalidRequest
		}

		medicineInfos = append(medicineInfos, medInfo)
	}
	return medicineInfos, nil
}
