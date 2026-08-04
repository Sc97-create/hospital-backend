package medicine

import (
	"errors"
	"fmt"
	"hospital-backend/internal/medicine/dto"
	"hospital-backend/pkg/middleware"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type SupplierController struct {
	SupplierSrv *SupplierService
}

type ISupplierController interface {
	CreateSupplier(c *fiber.Ctx) error
	GetSupplierByID(c *fiber.Ctx) error
	GetSupplierByOrgID(c *fiber.Ctx) error
	GetTotalCount(c *fiber.Ctx) error
}

func NewSupplierController(SupplierService *SupplierService) *SupplierController {
	return &SupplierController{SupplierSrv: SupplierService}
}

func (SController *SupplierController) CreateSupplier(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	var supplierRequestPayload dto.Supplier
	payload, err := params.New(c)
	if err != nil {
		logger.Warn("supplier create request invalid", zap.Error(err))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	supplierRequestPayload.UserID, err = payload.Getstring("user_id")
	if err != nil || strings.TrimSpace(supplierRequestPayload.UserID) == "" {
		logger.Warn("supplier create request invalid", zap.String("field", "user_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	supplierRequestPayload.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil || strings.TrimSpace(supplierRequestPayload.OrganisationID) == "" {
		logger.Warn("supplier create request invalid", zap.String("field", "organisation_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	supplierRequestPayload.Name, err = payload.Getstring("name")
	if err != nil || strings.TrimSpace(supplierRequestPayload.Name) == "" {
		logger.Warn("supplier create request invalid", zap.String("field", "name"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	supplierRequestPayload.PaymentTerms, err = payload.Getstring("payment_terms")
	if err != nil || strings.TrimSpace(supplierRequestPayload.PaymentTerms) == "" {
		logger.Warn("supplier create request invalid", zap.String("field", "payment_terms"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	supplierRequestPayload.EmailID, err = payload.Getstring("email_id")
	if err != nil {
		logger.Warn("supplier create request invalid", zap.String("field", "email_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	supplierRequestPayload.DrugLicenseNumber, err = payload.Getstring("drug_license_number")
	if err != nil {
		logger.Warn("supplier create request invalid", zap.String("field", "drug_license_number"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	supplierRequestPayload.ContactNumber, err = payload.Getstring("contact_number")
	if err != nil {
		logger.Warn("supplier create request invalid", zap.String("field", "contact_number"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	supplierRequestPayload.CreditLimit, err = payload.Getfloat("credit_limit")
	if err != nil {
		logger.Warn("supplier create request invalid", zap.String("field", "credit_limit"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	supplierRequestPayload.GstNumber, _ = payload.Getstring("gst_number")

	logger.Info("supplier create attempt",
		zap.String("organisation_id", supplierRequestPayload.OrganisationID),
		zap.String("user_id", supplierRequestPayload.UserID),
		zap.String("payment_terms", supplierRequestPayload.PaymentTerms),
	)

	supplierID, err := SController.SupplierSrv.CretateSupplier(logger, supplierRequestPayload)
	if err != nil {
		return wrapError.Wrap(wrapError.ErrSupplierCreateFailed, c, fiber.StatusInternalServerError)
	}
	response := make(map[string]interface{})
	response["code"] = 200
	response["message"] = "supplier created successfully"
	response["supplier_id"] = supplierID
	return c.JSON(response)
}

func (SController *SupplierController) GetSupplierByID(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	supplierID := strings.TrimSpace(c.Query("supplier_id"))
	if supplierID == "" {
		logger.Warn("supplier get request invalid", zap.String("field", "supplier_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	supplier, err := SController.SupplierSrv.GetSupplierByID(logger, supplierID)
	if err != nil {
		return SController.wrapGetError(c, err)
	}
	resp := make(map[string]interface{})
	resp["code"] = 200
	resp["data"] = supplier
	return c.JSON(resp)
}

func (SController *SupplierController) GetSupplierByOrgID(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	organisationID := strings.TrimSpace(c.Query("organisation_id"))
	if organisationID == "" {
		logger.Warn("supplier list request invalid", zap.String("field", "organisation_id"))
		return wrapError.Wrap(wrapError.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	limit := c.QueryInt("limit", 10)
	pageNo := c.QueryInt("page_no", 1)
	suppliers, total, err := SController.SupplierSrv.GetSupplierByOrgID(logger, organisationID, limit, pageNo)
	if err != nil {
		return wrapError.Wrap(wrapError.ErrSupplierFetchFailed, c, fiber.StatusInternalServerError)
	}
	resp := make(map[string]interface{})
	resp["code"] = 200
	resp["data"] = suppliers
	resp["total"] = total
	return c.JSON(resp)
}

func (SController *SupplierController) GetTotalCount(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	organisationID := strings.TrimSpace(c.Query("organisation_id"))
	if organisationID == "" {
		logger.Warn("supplier count request invalid", zap.String("field", "organisation_id"))
		return wrapError.Wrap(fmt.Errorf("organisation_id is required"), c, fiber.StatusBadRequest)
	}
	total, err := SController.SupplierSrv.GetTotalCount(logger, organisationID)
	if err != nil {
		return wrapError.Wrap(wrapError.ErrSupplierFetchFailed, c, fiber.StatusInternalServerError)
	}
	resp := make(map[string]interface{})
	resp["code"] = 200
	resp["total"] = total
	return c.JSON(resp)
}

func (SController *SupplierController) wrapGetError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapError.ErrSupplierNotFound):
		return wrapError.Wrap(err, c, fiber.StatusNotFound)
	default:
		return wrapError.Wrap(wrapError.ErrSupplierFetchFailed, c, fiber.StatusInternalServerError)
	}
}
