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
	GetInvoiceByAppointmentID(c *fiber.Ctx) error
	GetBillDetailsByPrescriptionID(c *fiber.Ctx) error
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
	// Optional: empty payment_type defaults to "prescription" in the service (back-compat).
	// supplier_id is only meaningful for a prescription checkout, and appointment_id only for consultation.
	checkoutReq.PaymentType, _ = payload.Getstring("payment_type")
	checkoutReq.SupplierID, _ = payload.Getstring("supplier_id")
	checkoutReq.AppointmentID, _ = payload.Getstring("appointment_id")

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
	if err = IB.parsePrescriptionFields(payload, &checkoutReq); err != nil {
		logger.Warn("invoice checkout request invalid", zap.Error(err), zap.String("field", "prescription_id/dispense_items"))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("invoice checkout attempt",
		zap.String("payment_type", checkoutReq.PaymentType),
		zap.String("prescription_id", checkoutReq.PrescriptionID),
		zap.String("appointment_id", checkoutReq.AppointmentID),
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
	case errors.Is(err, wrapErrors.ErrInvoiceAlreadyExists),
		errors.Is(err, wrapErrors.ErrAppointmentAlreadyBilled):
		return wrapErrors.Wrap(err, c, fiber.StatusConflict)
	case errors.Is(err, wrapErrors.ErrPatientNotFound),
		errors.Is(err, wrapErrors.ErrAppointmentNotFound):
		return wrapErrors.Wrap(err, c, fiber.StatusNotFound)
	case errors.Is(err, wrapErrors.ErrMedicineNotInPrescription),
		errors.Is(err, wrapErrors.ErrQtyExceedsRemaining),
		errors.Is(err, wrapErrors.ErrInsufficientStock),
		errors.Is(err, wrapErrors.ErrUnsupportedPaymentMode),
		errors.Is(err, wrapErrors.ErrInvalidPaymentType),
		errors.Is(err, wrapErrors.ErrAppointmentMismatch),
		errors.Is(err, wrapErrors.ErrInvalidRequest):
		return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
	default:
		return wrapErrors.Wrap(wrapErrors.ErrInvoiceCreateFailed, c, fiber.StatusInternalServerError)
	}
}

// parsePrescriptionFields parses prescription_id + dispense_items — required only for a
// prescription checkout. Consultation checkouts (payment_type=consultation) have neither.
func (IB *Ibilling) parsePrescriptionFields(payload *params.Payload, checkoutReq *dto.CheckoutReq) error {
	if checkoutReq.PaymentType == string(PaymentTypeConsultation) {
		return nil
	}
	var err error
	checkoutReq.PrescriptionID, err = payload.Getstring("prescription_id")
	if err != nil {
		return err
	}
	items, err := payload.GetChildren("dispense_items")
	if err != nil {
		return err
	}
	checkoutReq.DispensedItems, err = IB.toDispenseItems(items)
	return err
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

func (IB *Ibilling) GetBillDetailsByPrescriptionID(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	prescriptionID := strings.TrimSpace(c.Params("prescriptionID"))
	if prescriptionID == "" {
		logger.Warn("bill details get request invalid", zap.String("reason", "missing_prescription_id"))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("bill details get attempt", zap.String("prescription_id", prescriptionID))

	details, err := IB.BillingServ.GetBillDetailsByPrescriptionID(logger, prescriptionID)
	if err != nil {
		return IB.wrapBillDetailsError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": details,
		"code": 200,
	})
}

func (IB *Ibilling) wrapBillDetailsError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, wrapErrors.ErrPrescriptionNotFound):
		return wrapErrors.Wrap(err, c, fiber.StatusNotFound)
	case errors.Is(err, wrapErrors.ErrInvalidRequest):
		return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
	default:
		return wrapErrors.Wrap(err, c, fiber.StatusInternalServerError)
	}
}

// GetInvoiceByAppointmentID is the consultation-invoice lookup — consultation invoices have
// no prescription_id, so GetInvoiceByPrescriptionID can never find them.
func (IB *Ibilling) GetInvoiceByAppointmentID(c *fiber.Ctx) error {
	logger := middleware.GetLogger(c)
	appointmentID := strings.TrimSpace(c.Params("appointmentID"))
	if appointmentID == "" {
		logger.Warn("invoice get request invalid", zap.String("reason", "missing_appointment_id"))
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	logger.Info("invoice get attempt", zap.String("appointment_id", appointmentID))

	invoice, err := IB.BillingServ.GetInvoiceByAppointmentID(logger, appointmentID)
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
