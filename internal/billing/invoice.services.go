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
	wrapError "hospital-backend/shared/error"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
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

func (IService *InvoiceServ) CreateInvoice(log *zap.Logger, reqPayload dto.CheckoutReq) (dto.InvoiceResponse, error) {
	log = ensureLog(log)

	if strings.TrimSpace(reqPayload.IdempotencyKey) == "" {
		log.Warn("invoice checkout failed",
			zap.String("prescription_id", reqPayload.PrescriptionID),
			zap.String("reason", "missing_idempotency_key"),
		)
		return dto.InvoiceResponse{}, wrapError.ErrInvalidRequest
	}

	// Replay: same frontend key must not create another invoice/payment
	existing, err := IService.PaymentServ.GetPaymentByIdempotencyKey(log, reqPayload.IdempotencyKey)
	if err == nil {
		paymentURL, _ := IService.PaymentServ.GetPaymentURLByPaymentID(log, existing.ID)
		log.Info("invoice checkout success",
			zap.String("invoice_id", existing.InvoiceID),
			zap.String("payment_mode", reqPayload.PaymentMode),
			zap.String("reason", "idempotency_replay"),
			zap.Bool("has_payment_url", paymentURL != ""),
		)
		return dto.InvoiceResponse{InvoiceID: existing.InvoiceID, PaymentURL: paymentURL}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Error("invoice checkout failed",
			zap.String("prescription_id", reqPayload.PrescriptionID),
			zap.String("reason", "idempotency_lookup"),
			zap.Error(err),
		)
		return dto.InvoiceResponse{}, wrapError.ErrInvoiceCreateFailed
	}

	invoice := IService.toInvoiceModel(reqPayload)
	tx := IService.db.Begin()
	err = IService.InvRepo.CreateInvoice(log, tx, invoice)
	if err != nil {
		tx.Rollback()
		if isUniqueViolation(err) {
			log.Warn("invoice checkout failed",
				zap.String("prescription_id", reqPayload.PrescriptionID),
				zap.String("reason", "invoice_already_exists"),
			)
			return dto.InvoiceResponse{}, wrapError.ErrInvoiceAlreadyExists
		}
		log.Error("invoice checkout failed",
			zap.String("prescription_id", reqPayload.PrescriptionID),
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return dto.InvoiceResponse{}, wrapError.ErrInvoiceCreateFailed
	}

	err = IService.InoviceItemS.addInvoiceItems(log, tx, reqPayload.PrescriptionID, invoice.ID, reqPayload.DispensedItems)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, wrapError.ErrMedicineNotInPrescription) ||
			errors.Is(err, wrapError.ErrQtyExceedsRemaining) ||
			errors.Is(err, wrapError.ErrInsufficientStock) {
			return dto.InvoiceResponse{}, err
		}
		log.Error("invoice checkout failed",
			zap.String("prescription_id", reqPayload.PrescriptionID),
			zap.String("invoice_id", invoice.ID),
			zap.String("reason", "db_add_items"),
			zap.Error(err),
		)
		return dto.InvoiceResponse{}, wrapError.ErrInvoiceCreateFailed
	}

	if err = tx.Commit().Error; err != nil {
		log.Error("invoice checkout failed",
			zap.String("prescription_id", reqPayload.PrescriptionID),
			zap.String("invoice_id", invoice.ID),
			zap.String("reason", "db_commit"),
			zap.Error(err),
		)
		return dto.InvoiceResponse{}, wrapError.ErrInvoiceCreateFailed
	}

	patientInfo, err := IService.PatientServ.FindOne(log, reqPayload.PatientID)
	if err != nil {
		if errors.Is(err, wrapError.ErrPatientNotFound) {
			log.Warn("invoice checkout failed",
				zap.String("invoice_id", invoice.ID),
				zap.String("patient_id", reqPayload.PatientID),
				zap.String("reason", "patient_not_found"),
				zap.Error(err),
			)
			return dto.InvoiceResponse{}, wrapError.ErrPatientNotFound
		}
		log.Error("invoice checkout failed",
			zap.String("invoice_id", invoice.ID),
			zap.String("patient_id", reqPayload.PatientID),
			zap.String("reason", "patient_lookup"),
			zap.Error(err),
		)
		return dto.InvoiceResponse{}, wrapError.ErrInvoiceCreateFailed
	}

	var paymentResponse paymentDto.CreatePaymentResponse
	cmd := IService.toPaymentlinkModel(reqPayload, patientInfo, invoice.ID, invoice.InvoiceCode)
	switch reqPayload.PaymentMode {
	case constants.PaymentLink:
		paymentResponse, err = IService.PaymentServ.CreateLinkPayment(log, cmd)
		if err != nil {
			log.Error("invoice checkout failed",
				zap.String("invoice_id", invoice.ID),
				zap.String("prescription_id", reqPayload.PrescriptionID),
				zap.String("payment_mode", reqPayload.PaymentMode),
				zap.String("reason", "payment_create"),
				zap.Error(err),
			)
			return dto.InvoiceResponse{}, wrapError.ErrInvoiceCreateFailed
		}
	case constants.PaymentCash, constants.PaymentQR:
		_, err = IService.PaymentServ.CreatePendingPayment(log, cmd)
		if err != nil {
			log.Error("invoice checkout failed",
				zap.String("invoice_id", invoice.ID),
				zap.String("prescription_id", reqPayload.PrescriptionID),
				zap.String("payment_mode", reqPayload.PaymentMode),
				zap.String("reason", "payment_create"),
				zap.Error(err),
			)
			return dto.InvoiceResponse{}, wrapError.ErrInvoiceCreateFailed
		}
	default:
		log.Warn("invoice checkout failed",
			zap.String("invoice_id", invoice.ID),
			zap.String("payment_mode", reqPayload.PaymentMode),
			zap.String("reason", "unsupported_payment_mode"),
		)
		return dto.InvoiceResponse{}, wrapError.ErrUnsupportedPaymentMode
	}

	hasURL := paymentResponse.PaymentURL != ""
	log.Info("invoice checkout success",
		zap.String("invoice_id", invoice.ID),
		zap.String("invoice_code", invoice.InvoiceCode),
		zap.String("prescription_id", invoice.PrescriptionID),
		zap.String("patient_id", invoice.PatientID),
		zap.String("organisation_id", invoice.OrganisationID),
		zap.String("payment_mode", reqPayload.PaymentMode),
		zap.Int("item_count", len(reqPayload.DispensedItems)),
		zap.Bool("has_payment_url", hasURL),
	)
	return dto.InvoiceResponse{InvoiceID: invoice.ID, PaymentURL: paymentResponse.PaymentURL}, nil
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
	paymentdto.Amount = math.Round(payload.Financials.TotalAmount)
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

func (IService *InvoiceServ) updateInvoiceStatus(log *zap.Logger, tx *gorm.DB, invoiceID string, status string) error {
	return IService.InvRepo.UpdateInvoiceStatus(log, tx, invoiceID, status)
}

func (IService *InvoiceServ) GetInvoiceByPrescriptionID(log *zap.Logger, prescriptionID string) (dto.InvoiceByPrescriptionResponse, error) {
	log = ensureLog(log)
	if strings.TrimSpace(prescriptionID) == "" {
		return dto.InvoiceByPrescriptionResponse{}, wrapError.ErrInvalidRequest
	}
	query := `
		SELECT
			invoices.*,
			payments.source AS payment_mode
		FROM invoices
		LEFT JOIN payments ON payments.invoice_id = invoices.id
		WHERE invoices.prescription_id = ?
	`
	invoice, err := IService.InvRepo.GetInvoiceByPrescriptionID(log, query, prescriptionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("invoice get failed",
				zap.String("prescription_id", prescriptionID),
				zap.String("reason", "not_found"),
			)
			return dto.InvoiceByPrescriptionResponse{}, wrapError.ErrInvoiceNotFound
		}
		log.Error("invoice get failed",
			zap.String("prescription_id", prescriptionID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return dto.InvoiceByPrescriptionResponse{}, wrapError.ErrInvoiceFetchFailed
	}

	resp := toInvoiceByPrescriptionResponse(invoice)
	log.Info("invoice get success",
		zap.String("invoice_id", resp.ID),
		zap.String("prescription_id", prescriptionID),
		zap.String("status", resp.Status),
		zap.String("payment_mode", resp.PaymentMode),
	)
	return resp, nil
}

func (IService *InvoiceServ) RetryPaymentLink(log *zap.Logger, invoiceID, idempotencyKey string) (dto.InvoiceResponse, error) {
	log = ensureLog(log)
	if strings.TrimSpace(invoiceID) == "" {
		return dto.InvoiceResponse{}, wrapError.ErrInvalidRequest
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		log.Warn("invoice retry payment link failed",
			zap.String("invoice_id", invoiceID),
			zap.String("reason", "missing_idempotency_key"),
		)
		return dto.InvoiceResponse{}, wrapError.ErrInvalidRequest
	}

	invoice, err := IService.InvRepo.GetInvoiceByID(log, invoiceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("invoice retry payment link failed",
				zap.String("invoice_id", invoiceID),
				zap.String("reason", "not_found"),
			)
			return dto.InvoiceResponse{}, wrapError.ErrInvoiceNotFound
		}
		log.Error("invoice retry payment link failed",
			zap.String("invoice_id", invoiceID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return dto.InvoiceResponse{}, wrapError.ErrInvoiceFetchFailed
	}
	if invoice.Status != StatusUnpaid && invoice.Status != constants.InvoiceUnpaid {
		log.Warn("invoice retry payment link failed",
			zap.String("invoice_id", invoiceID),
			zap.String("status", invoice.Status),
			zap.String("reason", "invoice_not_unpaid"),
		)
		return dto.InvoiceResponse{}, wrapError.ErrInvoiceNotUnpaid
	}

	patientInfo, err := IService.PatientServ.FindOne(log, invoice.PatientID)
	if err != nil {
		if errors.Is(err, wrapError.ErrPatientNotFound) {
			log.Warn("invoice retry payment link failed",
				zap.String("invoice_id", invoiceID),
				zap.String("patient_id", invoice.PatientID),
				zap.String("reason", "patient_not_found"),
				zap.Error(err),
			)
			return dto.InvoiceResponse{}, wrapError.ErrPatientNotFound
		}
		log.Error("invoice retry payment link failed",
			zap.String("invoice_id", invoiceID),
			zap.String("patient_id", invoice.PatientID),
			zap.String("reason", "patient_lookup"),
			zap.Error(err),
		)
		return dto.InvoiceResponse{}, wrapError.ErrPaymentLinkRetryFailed
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

	paymentResponse, err := IService.PaymentServ.RetryLinkPayment(log, cmd)
	if err != nil {
		log.Error("invoice retry payment link failed",
			zap.String("invoice_id", invoiceID),
			zap.String("reason", "payment_retry"),
			zap.Error(err),
		)
		return dto.InvoiceResponse{}, wrapError.ErrPaymentLinkRetryFailed
	}

	log.Info("invoice retry payment link success",
		zap.String("invoice_id", invoice.ID),
		zap.Bool("has_payment_url", paymentResponse.PaymentURL != ""),
	)
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
