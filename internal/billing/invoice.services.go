package billing

import (
	"errors"
	"fmt"
	"hospital-backend/internal/appointments"
	"hospital-backend/internal/billing/dto"
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

type PatientLookup interface {
	FindOne(log *zap.Logger, id string) (patientDto.PatientResponse, error)
}

type AppointmentLookup interface {
	GetAppntmentByID(log *zap.Logger, appointmentID string) (appointments.Appointment, error)
}

type PaymentCheckout interface {
	GetPaymentByIdempotencyKey(log *zap.Logger, key string) (payments.Payments, error)
	GetPaymentURLByPaymentID(log *zap.Logger, paymentID string) (string, error)
	CreateLinkPayment(log *zap.Logger, paymentReq paymentDto.CreatePaymentCommand) (paymentDto.CreatePaymentResponse, error)
	CreatePendingPayment(log *zap.Logger, paymentReq paymentDto.CreatePaymentCommand) (payments.Payments, error)
	RetryLinkPayment(log *zap.Logger, paymentReq paymentDto.CreatePaymentCommand) (paymentDto.CreatePaymentResponse, error)
}

type InvoiceServ struct {
	db           *gorm.DB
	InvRepo      InvoiceRepo
	PaymentServ  PaymentCheckout
	InoviceItemS *InvoiceItemServ
	PatientServ  PatientLookup
	AppointmentS AppointmentLookup
}

func NewInvoiceServ(db *gorm.DB, IRepo InvoiceRepo, PaymentS PaymentCheckout, items *InvoiceItemServ, patientServ PatientLookup, appointmentServ AppointmentLookup) *InvoiceServ {
	return &InvoiceServ{db: db, InvRepo: IRepo, PaymentServ: PaymentS, InoviceItemS: items, PatientServ: patientServ, AppointmentS: appointmentServ}
}

// CreateInvoice is the single checkout entry point for both prescription and consultation
// invoices (payment_type discriminates). See docs/payment-type-invoice-design.md.
// Each checkout step (idempotency replay, validation, tx-scoped persist, patient lookup,
// payment creation) is extracted into its own helper below to stay under the complexity caps.
func (IService *InvoiceServ) CreateInvoice(log *zap.Logger, reqPayload dto.CheckoutReq) (dto.InvoiceResponse, error) {
	log = ensureLog(log)

	if strings.TrimSpace(reqPayload.IdempotencyKey) == "" {
		log.Warn("invoice checkout failed",
			zap.String("prescription_id", reqPayload.PrescriptionID),
			zap.String("reason", "missing_idempotency_key"),
		)
		return dto.InvoiceResponse{}, wrapError.ErrInvalidRequest
	}

	if resp, replayed, err := IService.replayIfIdempotent(log, reqPayload); replayed {
		return resp, err
	}

	paymentType := resolvePaymentType(reqPayload.PaymentType)
	if err := IService.validatePaymentTypeInputs(log, paymentType, reqPayload); err != nil {
		return dto.InvoiceResponse{}, err
	}

	invoice := IService.toInvoiceModel(reqPayload, paymentType)
	tx := IService.db.Begin()
	if err := IService.persistInvoiceAndItems(log, tx, invoice, paymentType, reqPayload); err != nil {
		return dto.InvoiceResponse{}, err
	}
	if err := tx.Commit().Error; err != nil {
		log.Error("invoice checkout failed",
			zap.String("prescription_id", reqPayload.PrescriptionID),
			zap.String("invoice_id", invoice.ID),
			zap.String("reason", "db_commit"),
			zap.Error(err),
		)
		return dto.InvoiceResponse{}, wrapError.ErrInvoiceCreateFailed
	}

	patientInfo, err := IService.lookupPatientOrFail(log, invoice.ID, reqPayload.PatientID)
	if err != nil {
		return dto.InvoiceResponse{}, err
	}

	cmd := IService.toPaymentlinkModel(reqPayload, patientInfo, invoice.ID, invoice.InvoiceCode, paymentType)
	paymentResponse, err := IService.createPaymentForMode(log, invoice.ID, reqPayload, cmd)
	if err != nil {
		return dto.InvoiceResponse{}, err
	}

	IService.logCheckoutSuccess(log, invoice, reqPayload, paymentResponse.PaymentURL != "")
	return dto.InvoiceResponse{InvoiceID: invoice.ID, PaymentURL: paymentResponse.PaymentURL}, nil
}

// replayIfIdempotent short-circuits CreateInvoice when the same Idempotency-Key was already
// used — same frontend key must not create a second invoice/payment. replayed=true means the
// caller should return (resp, err) immediately, whatever err is (nil on a real replay, a mapped
// failure if the idempotency lookup itself errored).
func (IService *InvoiceServ) replayIfIdempotent(log *zap.Logger, reqPayload dto.CheckoutReq) (dto.InvoiceResponse, bool, error) {
	existing, err := IService.PaymentServ.GetPaymentByIdempotencyKey(log, reqPayload.IdempotencyKey)
	if err == nil {
		paymentURL, _ := IService.PaymentServ.GetPaymentURLByPaymentID(log, existing.ID)
		log.Info("invoice checkout success",
			zap.String("invoice_id", existing.InvoiceID),
			zap.String("payment_mode", reqPayload.PaymentMode),
			zap.String("reason", "idempotency_replay"),
			zap.Bool("has_payment_url", paymentURL != ""),
		)
		return dto.InvoiceResponse{InvoiceID: existing.InvoiceID, PaymentURL: paymentURL}, true, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Error("invoice checkout failed",
			zap.String("prescription_id", reqPayload.PrescriptionID),
			zap.String("reason", "idempotency_lookup"),
			zap.Error(err),
		)
		return dto.InvoiceResponse{}, true, wrapError.ErrInvoiceCreateFailed
	}
	return dto.InvoiceResponse{}, false, nil
}

// persistInvoiceAndItems inserts the invoice row and (for prescription invoices) its
// invoice_items, all inside the caller's tx. Owns rollback on any failure; caller owns commit.
func (IService *InvoiceServ) persistInvoiceAndItems(log *zap.Logger, tx *gorm.DB, invoice Invoice, paymentType PaymentType, reqPayload dto.CheckoutReq) error {
	if err := IService.InvRepo.CreateInvoice(log, tx, invoice); err != nil {
		tx.Rollback()
		return IService.mapInvoiceCreateErr(log, err, paymentType, reqPayload)
	}

	// Consultation invoices have no dispensed medicines — nothing to validate/insert into invoice_items.
	if paymentType != PaymentTypePrescription {
		return nil
	}
	err := IService.InoviceItemS.addInvoiceItems(log, tx, reqPayload.PrescriptionID, invoice.ID, reqPayload.DispensedItems)
	if err == nil {
		return nil
	}
	tx.Rollback()
	if errors.Is(err, wrapError.ErrMedicineNotInPrescription) ||
		errors.Is(err, wrapError.ErrQtyExceedsRemaining) ||
		errors.Is(err, wrapError.ErrInsufficientStock) {
		return err
	}
	log.Error("invoice checkout failed",
		zap.String("prescription_id", reqPayload.PrescriptionID),
		zap.String("invoice_id", invoice.ID),
		zap.String("reason", "db_add_items"),
		zap.Error(err),
	)
	return wrapError.ErrInvoiceCreateFailed
}

func (IService *InvoiceServ) mapInvoiceCreateErr(log *zap.Logger, err error, paymentType PaymentType, reqPayload dto.CheckoutReq) error {
	if !isUniqueViolation(err) {
		log.Error("invoice checkout failed",
			zap.String("prescription_id", reqPayload.PrescriptionID),
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return wrapError.ErrInvoiceCreateFailed
	}
	log.Warn("invoice checkout failed",
		zap.String("prescription_id", reqPayload.PrescriptionID),
		zap.String("appointment_id", reqPayload.AppointmentID),
		zap.String("payment_type", string(paymentType)),
		zap.String("reason", "invoice_already_exists"),
	)
	if paymentType == PaymentTypeConsultation {
		return wrapError.ErrAppointmentAlreadyBilled
	}
	return wrapError.ErrInvoiceAlreadyExists
}

func (IService *InvoiceServ) lookupPatientOrFail(log *zap.Logger, invoiceID, patientID string) (patientDto.PatientResponse, error) {
	patientInfo, err := IService.PatientServ.FindOne(log, patientID)
	if err == nil {
		return patientInfo, nil
	}
	if errors.Is(err, wrapError.ErrPatientNotFound) {
		log.Warn("invoice checkout failed",
			zap.String("invoice_id", invoiceID),
			zap.String("patient_id", patientID),
			zap.String("reason", "patient_not_found"),
			zap.Error(err),
		)
		return patientDto.PatientResponse{}, wrapError.ErrPatientNotFound
	}
	log.Error("invoice checkout failed",
		zap.String("invoice_id", invoiceID),
		zap.String("patient_id", patientID),
		zap.String("reason", "patient_lookup"),
		zap.Error(err),
	)
	return patientDto.PatientResponse{}, wrapError.ErrInvoiceCreateFailed
}

// createPaymentForMode kicks off the payment side of checkout: a Razorpay link for
// payment_mode=link, or a pending record (no gateway call) for cash/qr collected in person.
func (IService *InvoiceServ) createPaymentForMode(log *zap.Logger, invoiceID string, reqPayload dto.CheckoutReq, cmd paymentDto.CreatePaymentCommand) (paymentDto.CreatePaymentResponse, error) {
	var resp paymentDto.CreatePaymentResponse
	var err error
	switch reqPayload.PaymentMode {
	case constants.PaymentLink:
		resp, err = IService.PaymentServ.CreateLinkPayment(log, cmd)
	case constants.PaymentCash, constants.PaymentQR:
		_, err = IService.PaymentServ.CreatePendingPayment(log, cmd)
	default:
		log.Warn("invoice checkout failed",
			zap.String("invoice_id", invoiceID),
			zap.String("payment_mode", reqPayload.PaymentMode),
			zap.String("reason", "unsupported_payment_mode"),
		)
		return paymentDto.CreatePaymentResponse{}, wrapError.ErrUnsupportedPaymentMode
	}
	if err != nil {
		log.Error("invoice checkout failed",
			zap.String("invoice_id", invoiceID),
			zap.String("prescription_id", reqPayload.PrescriptionID),
			zap.String("payment_mode", reqPayload.PaymentMode),
			zap.String("reason", "payment_create"),
			zap.Error(err),
		)
		return paymentDto.CreatePaymentResponse{}, wrapError.ErrInvoiceCreateFailed
	}
	return resp, nil
}

func (IService *InvoiceServ) logCheckoutSuccess(log *zap.Logger, invoice Invoice, reqPayload dto.CheckoutReq, hasPaymentURL bool) {
	log.Info("invoice checkout success",
		zap.String("invoice_id", invoice.ID),
		zap.String("invoice_code", invoice.InvoiceCode),
		zap.String("payment_type", string(invoice.PaymentType)),
		zap.String("prescription_id", derefString(invoice.PrescriptionID)),
		zap.String("appointment_id", derefString(invoice.AppointmentID)),
		zap.String("patient_id", invoice.PatientID),
		zap.String("organisation_id", invoice.OrganisationID),
		zap.String("payment_mode", reqPayload.PaymentMode),
		zap.Int("item_count", len(reqPayload.DispensedItems)),
		zap.Bool("has_payment_url", hasPaymentURL),
	)
}

// resolvePaymentType defaults empty/omitted payment_type to "prescription" for back-compat
// with clients that predate this field; anything else is passed through as-is so
// validatePaymentTypeInputs can reject genuinely unsupported values.
func resolvePaymentType(raw string) PaymentType {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return PaymentTypePrescription
	}
	return PaymentType(trimmed)
}

// validatePaymentTypeInputs enforces the payment_type ↔ required-field rule: prescription
// invoices need prescription_id, consultation invoices need a real, unbilled appointment
// that belongs to this patient/org. See docs/payment-type-invoice-design.md §2.2.
func (IService *InvoiceServ) validatePaymentTypeInputs(log *zap.Logger, paymentType PaymentType, reqPayload dto.CheckoutReq) error {
	switch paymentType {
	case PaymentTypeConsultation:
		return IService.validateConsultationAppointment(log, reqPayload)
	case PaymentTypePrescription:
		if strings.TrimSpace(reqPayload.PrescriptionID) == "" {
			log.Warn("invoice checkout failed", zap.String("reason", "missing_prescription_id"))
			return wrapError.ErrInvalidRequest
		}
		return nil
	default:
		log.Warn("invoice checkout failed",
			zap.String("payment_type", string(paymentType)),
			zap.String("reason", "invalid_payment_type"),
		)
		return wrapError.ErrInvalidPaymentType
	}
}

// validateConsultationAppointment checks the appointment exists, belongs to this
// patient/org, and hasn't already been billed — before any invoice row is written.
func (IService *InvoiceServ) validateConsultationAppointment(log *zap.Logger, reqPayload dto.CheckoutReq) error {
	if strings.TrimSpace(reqPayload.AppointmentID) == "" {
		log.Warn("invoice checkout failed", zap.String("reason", "missing_appointment_id"))
		return wrapError.ErrInvalidRequest
	}
	appointment, err := IService.AppointmentS.GetAppntmentByID(log, reqPayload.AppointmentID)
	if err != nil {
		return err
	}
	if appointment.OrganisationID != reqPayload.OrganisationID || appointment.PatientID != reqPayload.PatientID {
		log.Warn("invoice checkout failed",
			zap.String("appointment_id", reqPayload.AppointmentID),
			zap.String("reason", "appointment_mismatch"),
		)
		return wrapError.ErrAppointmentMismatch
	}

	_, err = IService.GetInvoiceByAppointmentID(log, reqPayload.AppointmentID)
	if err == nil {
		log.Warn("invoice checkout failed",
			zap.String("appointment_id", reqPayload.AppointmentID),
			zap.String("reason", "appointment_already_billed"),
		)
		return wrapError.ErrAppointmentAlreadyBilled
	}
	if !errors.Is(err, wrapError.ErrInvoiceNotFound) {
		return err
	}
	return nil
}

func (IService *InvoiceServ) toInvoiceModel(payload dto.CheckoutReq, paymentType PaymentType) Invoice {
	var invoice Invoice
	invoice.ID = uuid.New().String()
	invoice.InvoiceCode = IService.createCode()
	invoice.PaymentType = paymentType
	invoice.CashierID = payload.CashierID
	invoice.CreatedAt = time.Now()
	invoice.OrganisationID = payload.OrganisationID
	invoice.Status = StatusUnpaid
	invoice.SubtotalAmount = payload.Financials.SubtotalAmount
	invoice.TotalAmount = payload.Financials.TotalAmount
	invoice.DiscountAmount = payload.Financials.DiscountAmount
	invoice.TaxAmount = payload.Financials.TaxAmount
	invoice.PatientID = payload.PatientID
	if paymentType == PaymentTypeConsultation {
		invoice.AppointmentID = nonEmptyPtr(payload.AppointmentID)
		return invoice
	}
	invoice.PrescriptionID = nonEmptyPtr(payload.PrescriptionID)
	return invoice
}

// nonEmptyPtr turns a blank string into a real nil so nullable uuid columns store
// SQL NULL instead of ” — required so multiple rows can share a blank FK without
// tripping a uniqueIndex (see docs/payment-type-invoice-design.md §2.1).
func nonEmptyPtr(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return &v
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func (IService *InvoiceServ) toPaymentlinkModel(payload dto.CheckoutReq, patientInfo patientDto.PatientResponse, invoiceID string, invoiceCode string, paymentType PaymentType) paymentDto.CreatePaymentCommand {
	var paymentdto paymentDto.CreatePaymentCommand
	paymentdto.Amount = math.Round(payload.Financials.TotalAmount)
	paymentdto.ExpiresAt = time.Now().Add(20 * time.Minute)
	paymentdto.Customer.Email = patientInfo.PatientEmail
	paymentdto.Customer.Mobile = patientInfo.PatientPhone
	paymentdto.Customer.Name = patientInfo.PatientName
	paymentdto.SendEmail = true
	paymentdto.SendSMS = true
	paymentdto.Description = paymentDescription(paymentType)
	switch payload.PaymentMode {
	case constants.PaymentCash:
		paymentdto.Channel = constants.PaymentCash
	case constants.PaymentQR:
		paymentdto.Channel = constants.PaymentUPI
	default:
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

// paymentDescription is patient-facing copy shown alongside the SMS/email payment link.
func paymentDescription(paymentType PaymentType) string {
	if paymentType == PaymentTypeConsultation {
		return "please pay the consultation fee"
	}
	return "please pay the amount to get prescribed medicine"
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

	resp := toInvoiceDetailResponse(invoice)
	log.Info("invoice get success",
		zap.String("invoice_id", resp.ID),
		zap.String("prescription_id", prescriptionID),
		zap.String("status", resp.Status),
		zap.String("payment_mode", resp.PaymentMode),
	)
	return resp, nil
}

// GetInvoiceByAppointmentID is the consultation-invoice equivalent of GetInvoiceByPrescriptionID —
// consultation invoices have no prescription_id, so they can only be looked up this way.
func (IService *InvoiceServ) GetInvoiceByAppointmentID(log *zap.Logger, appointmentID string) (dto.InvoiceByPrescriptionResponse, error) {
	log = ensureLog(log)
	if strings.TrimSpace(appointmentID) == "" {
		return dto.InvoiceByPrescriptionResponse{}, wrapError.ErrInvalidRequest
	}
	query := `
		SELECT
			invoices.*,
			payments.source AS payment_mode
		FROM invoices
		LEFT JOIN payments ON payments.invoice_id = invoices.id
		WHERE invoices.appointment_id = ?
	`
	invoice, err := IService.InvRepo.GetInvoiceByAppointmentID(log, query, appointmentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("invoice get failed",
				zap.String("appointment_id", appointmentID),
				zap.String("reason", "not_found"),
			)
			return dto.InvoiceByPrescriptionResponse{}, wrapError.ErrInvoiceNotFound
		}
		log.Error("invoice get failed",
			zap.String("appointment_id", appointmentID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return dto.InvoiceByPrescriptionResponse{}, wrapError.ErrInvoiceFetchFailed
	}

	resp := toInvoiceDetailResponse(invoice)
	log.Info("invoice get success",
		zap.String("invoice_id", resp.ID),
		zap.String("appointment_id", appointmentID),
		zap.String("status", resp.Status),
		zap.String("payment_mode", resp.PaymentMode),
	)
	return resp, nil
}

// GetBillDetailsByPrescriptionID returns patient, appointment, and fee lines
// (consultation + prescription) for a prescription. Grand total is not computed here.
func (IService *InvoiceServ) GetBillDetailsByPrescriptionID(log *zap.Logger, prescriptionID string) (dto.BillDetailsByPrescriptionResponse, error) {
	log = ensureLog(log)
	if strings.TrimSpace(prescriptionID) == "" {
		return dto.BillDetailsByPrescriptionResponse{}, wrapError.ErrInvalidRequest
	}
	var billDetailsByPrescriptionQuery = `
SELECT
	p.id   AS prescription_id,
	p.code AS prescription_code,

	pt.id            AS patient_id,
	pt.uh_id         AS patient_uhid,
	pt.name          AS patient_name,
	pt.age           AS patient_age,
	pt.gender        AS patient_gender,
	pt.mobile_number AS patient_phone,
	pt.email_id      AS patient_email,

	a.id               AS appointment_id,
	a.appointment_code AS appointment_code,
	a.visit_type       AS visit_type,
	a.status           AS appointment_status,

	CASE WHEN ci.id IS NOT NULL THEN 1 END AS consultation_qty,
	ci.tax_amount       AS consultation_tax,
	ci.discount_amount  AS consultation_discount,
	ci.total_amount     AS consultation_total_amount,
	ci.status           AS consultation_invoice_status,
	cp.source           AS consultation_payment_mode,

	ri.qty              AS prescription_qty,
	pi.tax_amount       AS prescription_tax,
	pi.discount_amount  AS prescription_discount,
	pi.total_amount     AS prescription_total_amount,
	pi.status           AS prescription_invoice_status,
	pp.source           AS prescription_payment_mode,
	pi.id               AS prescription_invoice_id,
	pi.invoice_code     AS prescription_invoice_code,
	pi.created_at       AS prescription_invoice_created_at

FROM prescriptions p
LEFT JOIN patients pt
	ON pt.id = p.patient_id
LEFT JOIN appointments a
	ON a.id = p.appointment_id
LEFT JOIN invoices pi
	ON pi.prescription_id = p.id
	AND pi.payment_type = 'prescription'
LEFT JOIN invoices ci
	ON ci.appointment_id = p.appointment_id
	AND ci.payment_type = 'consultation'
LEFT JOIN LATERAL (
	SELECT source
	FROM payments
	WHERE invoice_id = pi.id
	ORDER BY created_at DESC
	LIMIT 1
) pp ON true
LEFT JOIN LATERAL (
	SELECT source
	FROM payments
	WHERE invoice_id = ci.id
	ORDER BY created_at DESC
	LIMIT 1
) cp ON true
LEFT JOIN (
	SELECT invoice_id, COALESCE(SUM(dispensed_qty), 0)::int AS qty
	FROM invoice_items
	GROUP BY invoice_id
) ri ON ri.invoice_id = pi.id
WHERE p.id = ?
`

	row, err := IService.InvRepo.GetBillDetailsByPrescriptionID(log, billDetailsByPrescriptionQuery, prescriptionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("bill details get failed",
				zap.String("prescription_id", prescriptionID),
				zap.String("reason", "not_found"),
			)
			return dto.BillDetailsByPrescriptionResponse{}, wrapError.ErrPrescriptionNotFound
		}
		log.Error("bill details get failed",
			zap.String("prescription_id", prescriptionID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return dto.BillDetailsByPrescriptionResponse{}, wrapError.ErrInvoiceFetchFailed
	}

	resp := toBillDetailsResponse(row)
	log.Info("bill details get success",
		zap.String("prescription_id", prescriptionID),
		zap.String("invoice_id", resp.InvoiceID),
		zap.String("invoice_status", resp.InvoiceStatus),
	)
	return resp, nil
}

// Latest payment row per invoice (avoids duplicate rows when multiple payments exist).

func toBillDetailsResponse(row BillDetailsRow) dto.BillDetailsByPrescriptionResponse {
	resp := dto.BillDetailsByPrescriptionResponse{
		PatientDetail:     toPatientDetail(row),
		AppointmentDetail: toAppointmentDetail(row),
		PaymentDetails:    toPaymentDetails(row),
		InvoiceStatus:     derefString(row.PrescriptionInvoiceStatus),
		InvoiceCode:       derefString(row.PrescriptionInvoiceCode),
		InvoiceID:         derefString(row.PrescriptionInvoiceID),
	}
	if row.PrescriptionInvoiceCreatedAt != nil {
		resp.CreatedAt = row.PrescriptionInvoiceCreatedAt.Format(time.RFC3339)
	}
	return resp
}

func toPatientDetail(row BillDetailsRow) dto.PatientDetail {
	return dto.PatientDetail{
		ID:     derefString(row.PatientID),
		UHID:   derefString(row.PatientUHID),
		Name:   derefString(row.PatientName),
		Age:    derefInt(row.PatientAge),
		Gender: derefString(row.PatientGender),
		Phone:  derefString(row.PatientPhone),
		Email:  derefString(row.PatientEmail),
	}
}

func toAppointmentDetail(row BillDetailsRow) dto.AppointmentDetail {
	return dto.AppointmentDetail{
		ID:              derefString(row.AppointmentID),
		AppointmentCode: derefString(row.AppointmentCode),
		VisitType:       derefString(row.VisitType),
		Status:          derefString(row.AppointmentStatus),
	}
}

func toPaymentDetails(row BillDetailsRow) dto.PaymentDetails {
	var details dto.PaymentDetails
	if row.ConsultationTotalAmount != nil || row.ConsultationInvoiceStatus != nil {
		details.Consultation = &dto.FeeLine{
			Code:          derefString(row.AppointmentCode),
			Category:      derefString(row.VisitType),
			Qty:           derefInt(row.ConsultationQty),
			Tax:           derefFloat(row.ConsultationTax),
			Discount:      derefFloat(row.ConsultationDiscount),
			TotalAmount:   derefFloat(row.ConsultationTotalAmount),
			InvoiceStatus: derefString(row.ConsultationInvoiceStatus),
			PaymentMode:   derefString(row.ConsultationPaymentMode),
		}
	}
	if row.PrescriptionInvoiceID != nil {
		details.Prescription = &dto.FeeLine{
			Code:          row.PrescriptionCode,
			Qty:           derefInt(row.PrescriptionQty),
			Tax:           derefFloat(row.PrescriptionTax),
			Discount:      derefFloat(row.PrescriptionDiscount),
			TotalAmount:   derefFloat(row.PrescriptionTotalAmount),
			InvoiceStatus: derefString(row.PrescriptionInvoiceStatus),
			PaymentMode:   derefString(row.PrescriptionPaymentMode),
		}
	}
	return details
}

func derefInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func derefFloat(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
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
		Description:    paymentDescription(invoice.PaymentType),
		ExpiresAt:      time.Now().Add(20 * time.Minute),
		SendEmail:      true,
		SendSMS:        true,
		PrescriptionID: derefString(invoice.PrescriptionID),
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

func (IService *InvoiceServ) GetTodayCompletedInvoiceSummary(log *zap.Logger, organisationID string) (dto.TodayInvoiceCollectionSummary, error) {
	log = ensureLog(log)
	if strings.TrimSpace(organisationID) == "" {
		return dto.TodayInvoiceCollectionSummary{}, wrapError.ErrInvalidRequest
	}
	query := `
		SELECT
			p.source AS payment_mode,
			COUNT(*)::int AS invoice_count,
			COALESCE(SUM(i.total_amount), 0) AS total_amount
		FROM invoices i
		INNER JOIN payments p ON p.invoice_id = i.id
		WHERE i.organisation_id = $1
			AND i.status = 'paid'
			AND i.updated_at >= CURRENT_DATE
			AND i.updated_at < CURRENT_DATE + INTERVAL '1 day'
		GROUP BY p.source
	`
	rows, err := IService.InvRepo.GetTodayCompletedInvoiceSummary(log, query, organisationID)
	if err != nil {
		log.Error("dashboard invoice summary failed",
			zap.String("organisation_id", organisationID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return dto.TodayInvoiceCollectionSummary{}, wrapError.ErrInvoiceFetchFailed
	}
	summary := buildTodayInvoiceCollectionSummary(rows)
	log.Info("dashboard invoice summary success",
		zap.String("organisation_id", organisationID),
		zap.Int("total_invoices", summary.TotalInvoices),
		zap.Float64("total_amount", summary.TotalAmount),
	)
	return summary, nil
}

func buildTodayInvoiceCollectionSummary(rows []TodayInvoiceCollectionRow) dto.TodayInvoiceCollectionSummary {
	summary := dto.TodayInvoiceCollectionSummary{}
	for _, row := range rows {
		bucket := dto.PaymentModeSummary{Count: row.Count, Amount: row.Amount}
		switch row.PaymentMode {
		case constants.PaymentCash:
			summary.Cash = bucket
		case constants.PaymentQR:
			summary.QR = bucket
		case constants.PaymentLink:
			summary.Link = bucket
		}
		summary.TotalInvoices += row.Count
		summary.TotalAmount += row.Amount
	}
	return summary
}

// toInvoiceDetailResponse is shared by GetInvoiceByPrescriptionID and GetInvoiceByAppointmentID —
// same invoice shape either way, just looked up by a different key.
func toInvoiceDetailResponse(row InvoiceWithPayment) dto.InvoiceByPrescriptionResponse {
	return dto.InvoiceByPrescriptionResponse{
		ID:             row.ID,
		InvoiceCode:    row.InvoiceCode,
		PaymentType:    string(row.PaymentType),
		PrescriptionID: derefString(row.PrescriptionID),
		AppointmentID:  derefString(row.AppointmentID),
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
