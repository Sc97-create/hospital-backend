package servicetest

import (
	"context"
	"testing"
	"time"

	admindto "hospital-backend/internal/admins/dto"
	authdto "hospital-backend/internal/authentication/dto"
	apptdto "hospital-backend/internal/appointments/dto"
	"hospital-backend/internal/billing/dto"
	"hospital-backend/internal/employee"
	empdto "hospital-backend/internal/employee/dto"
	meddto "hospital-backend/internal/medicine/dto"
	notificationdto "hospital-backend/internal/notifications/dto"
	orgdto "hospital-backend/internal/organisation/DTO"
	patientdto "hospital-backend/internal/patient/dto"
	paymentdto "hospital-backend/internal/payments/dto"
	prescdto "hospital-backend/internal/prescription/dto"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func NopLogger() *zap.Logger {
	return zap.NewNop()
}

type NoopNotifier struct{}

func (NoopNotifier) Create(_ context.Context, _ notificationdto.CreateRequest) error {
	return nil
}

func ValidPatientInfo() patientdto.PatientInfo {
	return patientdto.PatientInfo{
		UserID:         "user-1",
		Name:           "Jane Doe",
		Age:            "30",
		Weight:         "65",
		Gender:         "female",
		EmailID:        "jane@example.com",
		MobileNumber:   "9876543210",
		OrganisationID: "org-1",
		BloodGroup:     "O+",
		Address:        "123 Main St",
	}
}

func ValidAppointmentPayload() apptdto.NewApptmnt {
	future := time.Now().AddDate(0, 0, 1).Format(time.DateOnly)
	return apptdto.NewApptmnt{
		PatientID:       "pat-1",
		UserID:          "user-1",
		OrganisationID:  "org-1",
		DoctorID:        "doc-1",
		StartTime:       future + "T09:00:00Z",
		EndTime:         future + "T09:30:00Z",
		AppointmentDate: future,
		ReasonForVisit:  "consultation",
	}
}

func ValidOrgSchedule() admindto.GetResponse {
	return admindto.GetResponse{
		ID:             "sched-1",
		Starttime:      time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC),
		Endtime:        time.Date(2026, 1, 1, 17, 0, 0, 0, time.UTC),
		Slotduration:   30,
		BreakStarttime: time.Date(2026, 1, 1, 13, 0, 0, 0, time.UTC),
		BreakEndtime:   time.Date(2026, 1, 1, 14, 0, 0, 0, time.UTC),
	}
}

func ValidOrgScheduleReq() admindto.OrgScheduleReq {
	return admindto.OrgScheduleReq{
		OrganisationID: "org-1",
		WeekDays:       []string{"SUN", "SAT"},
		StartTime:      "09:00",
		EndTime:        "17:00",
		SlotDuration:   30,
		BreakStartTime: "13:00",
		BreakEndTime:   "14:00",
		IsClosed:       false,
	}
}

func ValidCheckoutReq() dto.CheckoutReq {
	return dto.CheckoutReq{
		IdempotencyKey: "idem-1",
		PaymentType:    "prescription",
		PrescriptionID: "rx-1",
		PatientID:      "pat-1",
		CashierID:      "cash-1",
		OrganisationID: "org-1",
		PaymentMode:    "cash",
		Financials: dto.Financial{
			SubtotalAmount: 100,
			TaxAmount:      10,
			TotalAmount:    110,
		},
		DispensedItems: []dto.DispensedItem{
			{
				MedicineID:          "med-1",
				MedicineInventoryID: "inv-1",
				PrescriptionItemID:  "pi-1",
				BatchNo:             "B1",
				QuantitySoldUnits:   2,
				CurrentStockUnits:   10,
				ComputedItemTotal:   100,
				TotalAmount:         110,
			},
		},
	}
}

func ValidDispensedItem() dto.DispensedItem {
	return dto.DispensedItem{
		MedicineID:          "med-1",
		MedicineInventoryID: "inv-1",
		PrescriptionItemID:  "pi-1",
		BatchNo:             "B1",
		QuantitySoldUnits:   2,
		CurrentStockUnits:   10,
		ComputedItemTotal:   100,
		TotalAmount:         110,
	}
}

func ValidLoginUser() authdto.LoginUser {
	return authdto.LoginUser{
		Username: "user@example.com",
		Password: "password123",
	}
}

func ValidUpdatePasswordRequest() authdto.UpdatePasswordRequest {
	return authdto.UpdatePasswordRequest{
		Password:        "newpassword",
		ConfirmPassword: "newpassword",
	}
}

func UserWithPassword(t *testing.T, plain string) *employee.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), 8)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	return &employee.User{
		ID:             "user-1",
		Username:       "user@example.com",
		EmailID:        "user@example.com",
		PasswordHash:   string(hash),
		OrganisationID: "org-1",
		RoleID:         "role-1",
	}
}

func ValidEmpFindManyRequest() empdto.FindManyRequest {
	return empdto.FindManyRequest{
		Limit:          10,
		PageNo:         1,
		OrganisationID: "org-1",
	}
}

func ValidCreatePaymentCommand() paymentdto.CreatePaymentCommand {
	return paymentdto.CreatePaymentCommand{
		InvoiceID:      "inv-1",
		PatientID:      "pat-1",
		InitiatedBy:    "user-1",
		Source:         "online",
		Channel:        "razorpay",
		Amount:         100,
		Currency:       "INR",
		PrescriptionID: "rx-1",
		IdempotencyKey: "idem-1",
	}
}

func ValidCreatePrescriptionRequest() prescdto.CreatePrescriptionRequest {
	return prescdto.CreatePrescriptionRequest{
		AppointmentID:  "appt-1",
		PatientID:      "pat-1",
		PrescribedBy:   "doc-1",
		OrganisationID: "org-1",
		MedicineArray:  []prescdto.MedicineArray{ValidMedicineArray()},
	}
}

func ValidMedicineArray() prescdto.MedicineArray {
	return prescdto.MedicineArray{
		MedicineID:      "med-1",
		MedicineName:    "Paracetamol",
		DurationDay:     7,
		DurationType:    "days",
		Morning:         1,
		FoodInstruction: "after food",
	}
}

func ValidPrescriptionItemUpdate() prescdto.UpdatePrescriptionItemRequest {
	return prescdto.UpdatePrescriptionItemRequest{
		PrescriptionItemID: "pi-1",
		MedicineID:         "med-1",
		DurationDay:        7,
		DurationType:       "days",
		Morning:            1,
		FoodInstruction:    "after food",
	}
}

func ValidMedicineRequestPayload() meddto.RequestPayload {
	return meddto.RequestPayload{
		UserID:         "user-1",
		SupplierID:     "sup-1",
		OrganisationID: "org-1",
		InvoiceNo:      "INV-001",
		MedicineArray: []meddto.MedicineInfo{
			{
				MedicineID:       "med-1",
				MedInventoryID:   "inv-1",
				Name:             "Paracetamol",
				Form:             "tablet",
				Strength:         "500mg",
				BatchNumber:      "B1",
				Quantity:         100,
				PurchaseQtyBoxes: 10,
				UnitPerBoxes:     10,
				Add:              true,
			},
		},
	}
}

func ValidSupplierCreateRequest() meddto.Supplier {
	return meddto.Supplier{
		UserID:         "user-1",
		OrganisationID: "org-1",
		Name:           "Acme Pharma",
		PaymentTerms:   "Net 30",
		EmailID:        "acme@example.com",
		ContactNumber:  "9876543210",
	}
}

func ValidOrganisationPayload() orgdto.OrganisationPayload {
	return orgdto.OrganisationPayload{
		OrganisationName: "City Hospital",
		LegalEntityName:  "City Hospital LLC",
		HospitalType:     "general",
	}
}
