package billing

import (
	"hospital-backend/internal/billing/dto"
	"hospital-backend/shared/params"

	wrapErrors "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
)

type Ibilling struct {
	BillingServ *InvoiceServ
}

type BillingHandler interface {
	Checkout(c *fiber.Ctx) error
}

func NewBillingController(BillingServ *InvoiceServ) *Ibilling {
	return &Ibilling{BillingServ: BillingServ}
}

func (IB *Ibilling) Checkout(c *fiber.Ctx) error {

	payload, err := params.New(c)
	if err != nil {
		return wrapErrors.Wrap(err, c, 409)
	}
	var checkoutReq dto.CheckoutReq
	checkoutReq.PrescriptionID, err = payload.Getstring("prescription_id")
	if err != nil {
		return wrapErrors.Wrap(err, c, 409)
	}
	checkoutReq.PatientID, err = payload.Getstring("patient_id")
	if err != nil {
		return wrapErrors.Wrap(err, c, 409)
	}
	checkoutReq.CashierID, err = payload.Getstring("cashier_id")
	if err != nil {
		return wrapErrors.Wrap(err, c, 409)
	}
	checkoutReq.SupplierID, err = payload.Getstring("supplier_id")
	if err != nil {
		return wrapErrors.Wrap(err, c, 409)
	}
	checkoutReq.OrganisationID, err = payload.Getstring("organisation_id")
	if err != nil {
		return wrapErrors.Wrap(err, c, 409)
	}
	financials, err := payload.GetObject("financials")
	if err != nil {
		return wrapErrors.Wrap(err, c, 409)
	}
	checkoutReq.Financials, err = IB.tofinancialMap(financials)
	if err != nil {
		return wrapErrors.Wrap(err, c, 409)
	}
	items, err := payload.GetChildren("dispense_items")
	if err != nil {
		return wrapErrors.Wrap(err, c, 409)
	}
	checkoutReq.DispensedItems, err = IB.toDispenseItems(items)
	if err != nil {
		return wrapErrors.Wrap(err, c, 409)
	}
	IB.BillingServ.CreatePaymentLink(checkoutReq)
	return c.JSON(fiber.Map{"message": "stored"})
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
		item.BatchNo, err = each.Getstring("batch_no")
		if err != nil {
			return nil, err
		}
		item.QuantitySoldUnits, _ = each.GetInt64("quantity_sold_units")
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
