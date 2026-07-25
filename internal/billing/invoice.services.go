package billing

import (
	"errors"
	"fmt"
	"hospital-backend/internal/billing/dto"
	"hospital-backend/internal/patient"
	patientDto "hospital-backend/internal/patient/dto"
	"hospital-backend/internal/payments"
	paymentDto "hospital-backend/internal/payments/dto"
	"hospital-backend/pkg/constants"
	"hospital-backend/pkg/logger"
	wrapError "hospital-backend/shared/error"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InvoiceServ struct {
	db           *gorm.DB
	InvRepo      InvoiceRepo
	PaymentServ  *payments.PaymentsService
	InoviceItemS *InvoiceItemServ
	PatientServ  *patient.PatientService
}

func NewInvoiceServ(db *gorm.DB, IRepo InvoiceRepo, PaymentS *payments.PaymentsService, items *InvoiceItemServ, patientServ *patient.PatientService) *InvoiceServ {
	return &InvoiceServ{db: db, InvRepo: IRepo, PaymentServ: PaymentS, InoviceItemS: items, PatientServ: patientServ}
}
func (IService *InvoiceServ) CreateInvoice(reqPayload dto.CheckoutReq) (dto.InvoiceResponse, error) {
	if strings.TrimSpace(reqPayload.IdempotencyKey) == "" {
		return dto.InvoiceResponse{}, fmt.Errorf("idempotency_key is required")
	}

	// Replay: same frontend key must not create another invoice/payment
	existing, err := IService.PaymentServ.GetPaymentByIdempotencyKey(reqPayload.IdempotencyKey)
	if err == nil {
		paymentURL, _ := IService.PaymentServ.GetPaymentURLByPaymentID(existing.ID)
		return dto.InvoiceResponse{InvoiceID: existing.InvoiceID, PaymentURL: paymentURL}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.InvoiceResponse{}, err
	}

	invoice := IService.toInvoiceModel(reqPayload)
	tx := IService.db.Begin()
	err = IService.InvRepo.CreateInvoice(tx, invoice)
	if err != nil {
		tx.Rollback()
		if isUniqueViolation(err) {
			return dto.InvoiceResponse{}, wrapError.ErrInvoiceAlreadyExists
		}
		return dto.InvoiceResponse{}, err
	}
	err = IService.InoviceItemS.addInvoiceItems(tx, reqPayload.PrescriptionID, invoice.ID, reqPayload.DispensedItems)
	if err != nil {
		tx.Rollback()
		return dto.InvoiceResponse{}, err
	}
	tx.Commit()
	patientInfo, err := IService.PatientServ.FindOne(logger.Log, reqPayload.PatientID)
	if err != nil {
		return dto.InvoiceResponse{}, err
	}

	var paymentResponse paymentDto.CreatePaymentResponse

	cmd := IService.toPaymentlinkModel(reqPayload, patientInfo, invoice.ID, invoice.InvoiceCode)
	switch reqPayload.PaymentMode {
	case constants.PaymentLink:
		paymentResponse, err = IService.PaymentServ.CreateLinkPayment(cmd)
		if err != nil {
			return dto.InvoiceResponse{}, err
		}
	case constants.PaymentCash, constants.PaymentQR:
		_, err = IService.PaymentServ.CreatePendingPayment(cmd)
		if err != nil {
			return dto.InvoiceResponse{}, err
		}
	default:
		return dto.InvoiceResponse{}, fmt.Errorf("unsupported payment_mode: %s", reqPayload.PaymentMode)
	}
	if paymentResponse.PaymentURL != "" {
		return dto.InvoiceResponse{InvoiceID: invoice.ID, PaymentURL: paymentResponse.PaymentURL}, nil
	}
	return dto.InvoiceResponse{InvoiceID: invoice.ID, PaymentURL: ""}, nil
}
func (IService *InvoiceServ) toInvoiceModel(payload dto.CheckoutReq) Invoice {
	var invoice Invoice
	invoice.ID = uuid.New().String()
	invoice.InvoiceCode = IService.createCode()
	invoice.CashierID = payload.CashierID
	invoice.CreatedAt = time.Now()
	invoice.OrganisationID = payload.OrganisationID
	invoice.Status = StatusUnpaid
	invoice.SubtotalAmount = payload.Financials.SubtotalAmount
	invoice.TotalAmount = payload.Financials.TotalAmount
	invoice.DiscountAmount = payload.Financials.DiscountAmount
	invoice.TaxAmount = payload.Financials.TaxAmount
	invoice.PrescriptionID = payload.PrescriptionID
	invoice.PatientID = payload.PatientID
	return invoice
}
func (IService *InvoiceServ) toPaymentlinkModel(payload dto.CheckoutReq, patientInfo patientDto.PatientResponse, invoiceID string, invoiceCode string) paymentDto.CreatePaymentCommand {

	var paymentdto paymentDto.CreatePaymentCommand
	paymentdto.Amount = math.Round(payload.Financials.TotalAmount) // rupees; gateway converts to paise
	paymentdto.ExpiresAt = time.Now().Add(20 * time.Minute)
	paymentdto.Customer.Email = patientInfo.PatientEmail
	paymentdto.Customer.Mobile = patientInfo.PatientPhone
	paymentdto.Customer.Name = patientInfo.PatientName
	paymentdto.SendEmail = true
	paymentdto.SendSMS = true
	paymentdto.Description = "please pay the amount to get prescribed medicine"
	if payload.PaymentMode == constants.PaymentCash {
		paymentdto.Channel = constants.PaymentCash
	} else if payload.PaymentMode == constants.PaymentQR {
		paymentdto.Channel = constants.PaymentUPI
	} else {
		paymentdto.Channel = constants.PaymentUPI
	}
	paymentdto.Source = payload.PaymentMode
	paymentdto.InitiatedBy = payload.CashierID
	paymentdto.PatientID = patientInfo.PatientID
	paymentdto.ReferenceID = invoiceCode
	paymentdto.InvoiceID = invoiceID
	paymentdto.Currency = constants.IndCurrnecy
	paymentdto.PrescriptionID = payload.PrescriptionID
	paymentdto.IdempotencyKey = payload.IdempotencyKey

	return paymentdto

}
func (IService *InvoiceServ) createCode() string {
	return fmt.Sprintf("%s-%d", InvPrefix, rand.Intn(9000)+1000)
}
func (IService *InvoiceServ) updateInvoiceStatus(tx *gorm.DB, invoiceID string, status string) error {
	return IService.InvRepo.UpdateInvoiceStatus(tx, invoiceID, status)
}

func (IService *InvoiceServ) GetInvoiceByPrescriptionID(prescriptionID string) (dto.InvoiceByPrescriptionResponse, error) {
	if strings.TrimSpace(prescriptionID) == "" {
		return dto.InvoiceByPrescriptionResponse{}, wrapError.ErrInvalidRequest
	}
	query := `
		SELECT
			invoices.*,
			payments.source AS payment_mode
		FROM invoices
		JOIN payments ON payments.invoice_id = invoices.id
		WHERE invoices.prescription_id = ?
	`
	invoice, err := IService.InvRepo.GetInvoiceByPrescriptionID(query, prescriptionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.InvoiceByPrescriptionResponse{}, wrapError.ErrInvoiceNotFound
		}
		return dto.InvoiceByPrescriptionResponse{}, err
	}
	return toInvoiceByPrescriptionResponse(invoice), nil
}

// RetryPaymentLink creates another provider payment link for an existing unpaid invoice (new attempt only).
func (IService *InvoiceServ) RetryPaymentLink(invoiceID, idempotencyKey string) (dto.InvoiceResponse, error) {
	if strings.TrimSpace(invoiceID) == "" {
		return dto.InvoiceResponse{}, wrapError.ErrInvalidRequest
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return dto.InvoiceResponse{}, fmt.Errorf("idempotency_key is required")
	}

	invoice, err := IService.InvRepo.GetInvoiceByID(invoiceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.InvoiceResponse{}, wrapError.ErrInvoiceNotFound
		}
		return dto.InvoiceResponse{}, err
	}
	if invoice.Status != StatusUnpaid && invoice.Status != constants.InvoiceUnpaid {
		return dto.InvoiceResponse{}, fmt.Errorf("invoice must be unpaid to retry payment link (status=%s)", invoice.Status)
	}

	patientInfo, err := IService.PatientServ.FindOne(logger.Log, invoice.PatientID)
	if err != nil {
		return dto.InvoiceResponse{}, err
	}

	cmd := paymentDto.CreatePaymentCommand{
		InvoiceID:      invoice.ID,
		PatientID:      invoice.PatientID,
		InitiatedBy:    invoice.CashierID,
		Source:         constants.PaymentLink,
		Channel:        constants.PaymentUPI,
		Amount:         math.Round(invoice.TotalAmount),
		ReferenceID:    invoice.InvoiceCode,
		Currency:       constants.IndCurrnecy,
		Description:    "please pay the amount to get prescribed medicine",
		ExpiresAt:      time.Now().Add(20 * time.Minute),
		SendEmail:      true,
		SendSMS:        true,
		PrescriptionID: invoice.PrescriptionID,
		IdempotencyKey: idempotencyKey,
		Customer: paymentDto.CustomerInfo{
			Name:   patientInfo.PatientName,
			Email:  patientInfo.PatientEmail,
			Mobile: patientInfo.PatientPhone,
		},
	}

	paymentResponse, err := IService.PaymentServ.RetryLinkPayment(cmd)
	if err != nil {
		return dto.InvoiceResponse{}, err
	}
	return dto.InvoiceResponse{InvoiceID: invoice.ID, PaymentURL: paymentResponse.PaymentURL}, nil
}

func toInvoiceByPrescriptionResponse(row InvoiceWithPayment) dto.InvoiceByPrescriptionResponse {
	return dto.InvoiceByPrescriptionResponse{
		ID:             row.ID,
		InvoiceCode:    row.InvoiceCode,
		PrescriptionID: row.PrescriptionID,
		PatientID:      row.PatientID,
		Status:         row.Status,
		CashierID:      row.CashierID,
		OrganisationID: row.OrganisationID,
		PaymentMode:    row.PaymentMode,
		SubtotalAmount: row.SubtotalAmount,
		TaxAmount:      row.TaxAmount,
		TotalAmount:    row.TotalAmount,
		DiscountAmount: row.DiscountAmount,
		CreatedAt:      row.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      row.UpdatedAt.Format(time.RFC3339),
	}
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique")
}
