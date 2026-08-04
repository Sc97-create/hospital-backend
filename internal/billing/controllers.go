package billing

import (
	"errors"
	"hospital-backend/internal/billing/dto"
	"hospital-backend/pkg/middleware"
	wrapErrors "hospital-backend/shared/error"
	"hospital-backend/shared/params"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type Ibilling struct {
	BillingServ *InvoiceServ
}

type BillingHandler interface {
	Checkout(c *fiber.Ctx) error
	GetInvoiceByPrescriptionID(c *fiber.Ctx) error
	RetryPaymentLink(c *fiber.Ctx) error
}

func NewBillingController(BillingServ *InvoiceServ) *Ibilling {
	return &Ibilling{BillingServ: BillingServ}
}

func (IB *Ibilling) Checkout(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)

	payload, err := params.New(c)
	if err != nil {
		logger.Warn("invoice checkout request invalid", zap.Error(err))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	var checkoutReq dto.CheckoutReq
	checkoutReq.PrescriptionID, err = payload.Getstring("prescription_id")
	if err != nil {
		logger.Warn("invoice checkout request invalid", zap.Error(err), zap.String("field", "prescription_id"))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	checkoutReq.PatientID, err = payload.Getstring("patient_id")
	if err != nil {
		logger.Warn("invoice checkout request invalid", zap.Error(err), zap.String("field", "patient_id"))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	checkoutReq.CashierID, err = payload.Getstring("cashier_id")
	if err != nil {
		logger.Warn("invoice checkout request invalid", zap.Error(err), zap.String("field", "cashier_id"))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	checkoutReq.SupplierID, err = payload.Getstring("supplier_id")
	if err != nil {
		logger.Warn("invoice checkout request invalid", zap.Error(err), zap.String("field", "supplier_id"))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	checkoutReq.PaymentMode, err = payload.Getstring("payment_mode")
	if err != nil {
		logger.Warn("invoice checkout request invalid", zap.Error(err), zap.String("field", "payment_mode"))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	checkoutReq.IdempotencyKey = strings.TrimSpace(c.Get("Idempotency-Key"))
	if checkoutReq.IdempotencyKey == "" {
		logger.Warn("invoice checkout request invalid", zap.String("reason", "missing_idempotency_key"))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	checkoutReq.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		logger.Warn("invoice checkout request invalid", zap.Error(err), zap.String("field", "organisation_id"))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	financials, err := payload.GetObject("financials")
	if err != nil {
		logger.Warn("invoice checkout request invalid", zap.Error(err), zap.String("field", "financials"))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	checkoutReq.Financials, err = IB.tofinancialMap(financials)
	if err != nil {
		logger.Warn("invoice checkout request invalid", zap.Error(err), zap.String("field", "financials"))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	items, err := payload.GetChildren("dispense_items")
	if err != nil {
		logger.Warn("invoice checkout request invalid", zap.Error(err), zap.String("field", "dispense_items"))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	checkoutReq.DispensedItems, err = IB.toDispenseItems(items)
	if err != nil {
		logger.Warn("invoice checkout request invalid", zap.Error(err), zap.String("field", "dispense_items"))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("invoice checkout attempt",
		zap.String("prescription_id", checkoutReq.PrescriptionID),
		zap.String("patient_id", checkoutReq.PatientID),
		zap.String("organisation_id", checkoutReq.OrganisationID),
		zap.String("cashier_id", checkoutReq.CashierID),
		zap.String("payment_mode", checkoutReq.PaymentMode),
		zap.Int("item_count", len(checkoutReq.DispensedItems)),
		zap.Bool("has_idempotency_key", true),
	)

	invoiceResponse, err := IB.BillingServ.CreateInvoice(logger, checkoutReq)
	if err != nil {
		return IB.wrapCheckoutError(c, err)
	}
	return c.JSON(fiber.Map{"message": "stored", "payment": invoiceResponse})
}

func (IB *Ibilling) wrapCheckoutError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapErrors.ErrInvoiceAlreadyExists):
		return wrapErrors.Wrap(err, c, fiber.StatusConflict)
	case errors.Is(err, wrapErrors.ErrPatientNotFound):
		return wrapErrors.Wrap(err, c, fiber.StatusNotFound)
	case errors.Is(err, wrapErrors.ErrMedicineNotInPrescription),
		errors.Is(err, wrapErrors.ErrQtyExceedsRemaining),
		errors.Is(err, wrapErrors.ErrInsufficientStock),
		errors.Is(err, wrapErrors.ErrUnsupportedPaymentMode),
		errors.Is(err, wrapErrors.ErrInvalidRequest):
		return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
	default:
		return wrapErrors.Wrap(wrapErrors.ErrInvoiceCreateFailed, c, fiber.StatusInternalServerError)
	}
}

func (IB *Ibilling) GetInvoiceByPrescriptionID(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	prescriptionID := strings.TrimSpace(c.Params("prescriptionID"))
	if prescriptionID == "" {
		logger.Warn("invoice get request invalid", zap.String("reason", "missing_prescription_id"))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("invoice get attempt", zap.String("prescription_id", prescriptionID))

	invoice, err := IB.BillingServ.GetInvoiceByPrescriptionID(logger, prescriptionID)
	if err != nil {
		if errors.Is(err, wrapErrors.ErrInvoiceNotFound) {
			return wrapErrors.Wrap(err, c, fiber.StatusNotFound)
		}
		if errors.Is(err, wrapErrors.ErrInvalidRequest) {
			return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
		}
		return wrapErrors.Wrap(err, c, fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": invoice,
		"code": 200,
	})
}

func (IB *Ibilling) RetryPaymentLink(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	invoiceID := strings.TrimSpace(c.Params("invoiceID"))
	if invoiceID == "" {
		logger.Warn("invoice retry payment link request invalid", zap.String("reason", "missing_invoice_id"))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		logger.Warn("invoice retry payment link request invalid", zap.String("reason", "missing_idempotency_key"))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("invoice retry payment link attempt",
		zap.String("invoice_id", invoiceID),
		zap.Bool("has_idempotency_key", true),
	)

	invoiceResponse, err := IB.BillingServ.RetryPaymentLink(logger, invoiceID, idempotencyKey)
	if err != nil {
		switch {
		case errors.Is(err, wrapErrors.ErrInvoiceNotFound),
			errors.Is(err, wrapErrors.ErrPatientNotFound):
			return wrapErrors.Wrap(err, c, fiber.StatusNotFound)
		case errors.Is(err, wrapErrors.ErrInvalidRequest),
			errors.Is(err, wrapErrors.ErrInvoiceNotUnpaid):
			return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
		default:
			return wrapErrors.Wrap(wrapErrors.ErrPaymentLinkRetryFailed, c, fiber.StatusInternalServerError)
		}
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "payment link created",
		"payment": invoiceResponse,
	})
}

func (IB *Ibilling) tofinancialMap(financials *params.Payload) (dto.Financial, error) {
	var finance dto.Financial
	var err error
	finance.DiscountAmount, _ = financials.Getfloat("discount_amount")
	finance.SubtotalAmount, err = financials.Getfloat("sub_total_amount")
	if err != nil {
		return dto.Financial{}, err
	}
	finance.TaxAmount, err = financials.Getfloat("tax_amount")
	if err != nil {
		return dto.Financial{}, err
	}
	finance.TotalAmount, err = financials.Getfloat("total_amount")
	if err != nil {
		return dto.Financial{}, err
	}
	return finance, nil
}

func (IB *Ibilling) toDispenseItems(dispenseItems []*params.Payload) ([]dto.DispensedItem, error) {
	var items []dto.DispensedItem
	var err error
	for _, each := range dispenseItems {
		var item dto.DispensedItem
		item.MedicineID, err = each.Getstring("medicine_id")
		if err != nil {
			return nil, err
		}
		item.MedicineInventoryID, err = each.Getstring("medicine_inventory_id")
		if err != nil {
			return nil, err
		}
		item.PrescriptionItemID, err = each.Getstring("prescription_item_id")
		if err != nil {
			return nil, err
		}
		item.BatchNo, err = each.Getstring("batch_no")
		if err != nil {
			return nil, err
		}
		item.CurrentStockUnits, _ = each.Getfloat("current_stock_units")
		item.QuantitySoldUnits, _ = each.Getfloat("quantity_sold_units")
		item.UnitPriceCharged, _ = each.Getfloat("unit_price_charged")
		item.ComputedItemTotal, _ = each.Getfloat("computed_item_total")
		item.TotalAmount, err = each.Getfloat("total_amount")
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
