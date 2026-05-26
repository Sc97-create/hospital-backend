package medicine

import (
	"hospital-backend/internal/medicine/dto"
	wrapError "hospital-backend/shared/error"
	"hospital-backend/shared/params"

	"github.com/gofiber/fiber/v2"
)

type SupplierController struct {
	SupplierSrv *SupplierService
}

type ISupplierController interface {
	CreateSupplier(c *fiber.Ctx) error
	GetSupplierByID(c *fiber.Ctx) error
}

func NewSupplierController(SupplierService *SupplierService) *SupplierController {
	return &SupplierController{SupplierSrv: SupplierService}
}
func (SController *SupplierController) CreateSupplier(c *fiber.Ctx) error {
	var supplierRequestPayload dto.Supplier
	payload, err := params.New(c)
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	supplierRequestPayload.UserID, err = payload.Getstring("user_id")
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	supplierRequestPayload.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	supplierRequestPayload.Name, err = payload.Getstring("name")
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	supplierRequestPayload.PaymentTerms, err = payload.Getstring("payment_terms")
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	supplierRequestPayload.EmailID, err = payload.Getstring("email_id")
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	supplierRequestPayload.DrugLicenseNumber, err = payload.Getstring("drug_license_number")
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	supplierRequestPayload.ContactNumber, err = payload.Getstring("contact_number")
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	supplierRequestPayload.CreditLimit, err = payload.Getfloat("credit_limit")
	if err != nil {
		return wrapError.Wrap(err, c, 400)
	}
	supplierRequestPayload.GstNumber, _ = payload.Getstring("gst_number")
	err = SController.SupplierSrv.CretateSupplier(supplierRequestPayload)
	if err != nil {
		return wrapError.Wrap(err, c, 500)
	}
	response := make(map[string]interface{})
	response["code"] = 200
	response["message"] = "supplier created successfully"
	err = c.JSON(response)
	if err != nil {
		return wrapError.Wrap(err, c, 500)
	}
	return nil
}

func (SController *SupplierController) GetSupplierByID(c *fiber.Ctx) error {
	supplierID := c.Query("supplier_id")
	supplier, err := SController.SupplierSrv.GetSupplierByID(supplierID)
	if err != nil {
		return wrapError.Wrap(err, c, 409)
	}
	resp := make(map[string]interface{})
	resp["code"] = 200
	resp["data"] = supplier
	err = c.JSON(resp)
	if err != nil {
		return wrapError.Wrap(err, c, 500)
	}
	return nil
}
