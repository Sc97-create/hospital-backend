package billing

import (
	"fmt"
	"hospital-backend/internal/billing/dto"
	"hospital-backend/internal/patient"
	patientDto "hospital-backend/internal/patient/dto"
	"hospital-backend/internal/payments"
	paymentDto "hospital-backend/internal/payments/dto"
	"hospital-backend/pkg/constants"
	"math"
	"math/rand"
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
func (IService *InvoiceServ) CreatePaymentLink(reqPayload dto.CheckoutReq) (string, error) {
	invoice := IService.toInvoiceModel(reqPayload)
	tx := IService.db.Begin()
	err := IService.InvRepo.CreateInvoice(tx, invoice)
	if err != nil {
		tx.Rollback()
		return "", err
	}
	err = IService.InoviceItemS.addInvoiceItems(tx, reqPayload.PrescriptionID, invoice.ID, reqPayload.DispensedItems)
	if err != nil {
		tx.Rollback()
		return "", err
	}
	tx.Commit()
	patientInfo, err := IService.PatientServ.FindOne(reqPayload.PatientID)
	if err != nil {
		return "", err
	}
	paymentDto := IService.toPaymentlinkModel(reqPayload, patientInfo, invoice.ID, invoice.InvoiceCode)
	paymentResponse, err := IService.PaymentServ.StorePaymentandNotifyUser(paymentDto)
	if err != nil {
		return "", err
	}
	if paymentResponse.PaymentURL != "" {
		return paymentResponse.PaymentURL, nil
	}
	return paymentResponse.PaymentURL, nil
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
	paymentdto.Channel = payload.PaymentMode
	paymentdto.Source = Source
	paymentdto.InitiatedBy = payload.CashierID
	paymentdto.PatientID = patientInfo.PatientID
	paymentdto.ReferenceID = invoiceCode
	paymentdto.InvoiceID = invoiceID
	paymentdto.Currency = constants.IndCurrnecy
	paymentdto.PrescriptionID = payload.PrescriptionID

	return paymentdto

}
func (IService *InvoiceServ) createCode() string {
	return fmt.Sprintf("%s-%d", InvPrefix, rand.Intn(9000)+1000)
}
func (IService *InvoiceServ) updateInvoiceStatus(tx *gorm.DB, invoiceID string, status string) error {
	return IService.InvRepo.UpdateInvoiceStatus(tx, invoiceID, status)
}
