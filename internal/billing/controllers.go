package billing

import (
	"errors"
	"fmt"
	"hospital-backend/internal/billing/dto"
	wrapErrors "hospital-backend/shared/error"
	"hospital-backend/shared/params"
	"strings"

	"github.com/gofiber/fiber/v2"
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

	payload, err := params.New(c)
	if err != nil {
		return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
	}
	var checkoutReq dto.CheckoutReq
	checkoutReq.PrescriptionID, err = payload.Getstring("prescription_id")
	if err != nil {
		return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
	}
	checkoutReq.PatientID, err = payload.Getstring("patient_id")
	if err != nil {
		return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
	}
	checkoutReq.CashierID, err = payload.Getstring("cashier_id")
	if err != nil {
		return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
	}
	checkoutReq.SupplierID, err = payload.Getstring("supplier_id")
	if err != nil {
		return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
	}
	checkoutReq.PaymentMode, err = payload.Getstring("payment_mode")
	if err != nil {
		return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
	}
	checkoutReq.IdempotencyKey = strings.TrimSpace(c.Get("Idempotency-Key"))
	if checkoutReq.IdempotencyKey == "" {
		return wrapErrors.Wrap(fmt.Errorf("Idempotency-Key header is required"), c, fiber.StatusBadRequest)
	}
	checkoutReq.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
	}
	financials, err := payload.GetObject("financials")
	if err != nil {
		return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
	}
	checkoutReq.Financials, err = IB.tofinancialMap(financials)
	if err != nil {
		return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
	}
	items, err := payload.GetChildren("dispense_items")
	if err != nil {
		return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
	}

	checkoutReq.DispensedItems, err = IB.toDispenseItems(items)
	if err != nil {
		return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
	}
	invoiceResponse, err := IB.BillingServ.CreateInvoice(checkoutReq)
	if err != nil {
		if errors.Is(err, wrapErrors.ErrInvoiceAlreadyExists) {
			return wrapErrors.Wrap(err, c, fiber.StatusConflict)
		}
		return wrapErrors.Wrap(err, c, fiber.StatusConflict)
	}
	return c.JSON(fiber.Map{"message": "stored", "payment": invoiceResponse})
}

func (IB *Ibilling) GetInvoiceByPrescriptionID(c *fiber.Ctx) error {
	prescriptionID := strings.TrimSpace(c.Params("prescriptionID"))
	if prescriptionID == "" {
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}

	invoice, err := IB.BillingServ.GetInvoiceByPrescriptionID(prescriptionID)
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
	invoiceID := strings.TrimSpace(c.Params("invoiceID"))
	if invoiceID == "" {
		return wrapErrors.Wrap(wrapErrors.ErrInvalidRequest, c, fiber.StatusBadRequest)
	}
	idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		return wrapErrors.Wrap(fmt.Errorf("Idempotency-Key header is required"), c, fiber.StatusBadRequest)
	}

	invoiceResponse, err := IB.BillingServ.RetryPaymentLink(invoiceID, idempotencyKey)
	if err != nil {
		if errors.Is(err, wrapErrors.ErrInvoiceNotFound) {
			return wrapErrors.Wrap(err, c, fiber.StatusNotFound)
		}
		if errors.Is(err, wrapErrors.ErrInvalidRequest) {
			return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
		}
		return wrapErrors.Wrap(err, c, fiber.StatusBadRequest)
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
