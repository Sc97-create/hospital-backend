package service

import (
	"context"
	"fmt"
	"hospital-backend/internal/notifications"
	"hospital-backend/internal/notifications/dto"
	"hospital-backend/internal/notifications/render"
	"hospital-backend/internal/notifications/repository"
	"time"

	"github.com/google/uuid"
)

// Notificationservice handles notification operations
type Notificationservice struct {
	repo     repository.Repository
	renderer *render.HTMLRenderer
}

func NewNotificationService(repo repository.Repository, render *render.HTMLRenderer) *Notificationservice {
	return &Notificationservice{repo: repo, renderer: render}
}

func (s *Notificationservice) Create(ctx context.Context, data dto.CreateRequest) error {
	notificationdata := s.parseeventdata(data.Data)
	content, err := s.renderer.Render(data.NotificationType, notificationdata)
	if err != nil {
		return err
	}
	notification := &notifications.Notification{
		ID:               uuid.New().String(),
		OrganisationID:   notificationdata.OrganisationID,
		PatientID:        notificationdata.PatientID,
		NotificationType: data.NotificationType,
		ProviderPayload: notifications.PPayload{
			Channel:        notifications.ChannelType(notifications.EmailChannel),
			Subject:        data.Subject,
			RecipientEmail: notificationdata.PatientEmail,
			Content:        content,
		},
		Status:      notifications.NotificationStatus(notifications.PendingStatus),
		RetryCount:  0,
		NextRetryAt: time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return s.repo.Create(ctx, notification)
}
func (s *Notificationservice) parseeventdata(data any) dto.NotificationModel {
	var notificationData dto.NotificationModel
	switch v := data.(type) {
	case map[string]interface{}:
		appointmentDate, _ := v["appointment_date"].(time.Time)
		startTime, _ := v["start_time"].(time.Time)
		endTime, _ := v["end_time"].(time.Time)
		notificationData.AppointmentCode, _ = v["appointment_code"].(string)
		notificationData.AppointmentDate = appointmentDate.Format("02 Jan 2006")
		notificationData.AppointmentTime = fmt.Sprintf("%s - %s", startTime.Format("03:04 PM"), endTime.Format("03:04 PM"))
		notificationData.DoctorName, _ = v["doctor_name"].(string)
		notificationData.HospitalName, _ = v["hospital_name"].(string)
		notificationData.PatientName, _ = v["patient_name"].(string)
		notificationData.PatientEmail, _ = v["patient_email_id"].(string)
		notificationData.PatientID, _ = v["patient_id"].(string)
		notificationData.OrganisationID, _ = v["organisation_id"].(string)
		notificationData.PatientCode, _ = v["patient_code"].(string)
		notificationData.PatientPhone, _ = v["patient_phone"].(string)
		notificationData.ConsultedOn, _ = v["consulted_on"].(string)
		notificationData.PrescriptionCode, _ = v["prescription_code"].(string)
		notificationData.PrescriptionSummary, _ = v["prescription_summary"].(string)
		amountPaid, _ := v["amount_paid"].(float64)
		notificationData.Amount = fmt.Sprintf("%.2f", amountPaid)
		notificationData.Currency, _ = v["currency"].(string)
		notificationData.PaymentStatus, _ = v["payment_status"].(string)
		notificationData.PaidAt, _ = v["paid_at"].(string)
		notificationData.Medicines = parsePrescriptionMedicines(v["medicines"])
	}
	return notificationData
}

func parsePrescriptionMedicines(raw any) []dto.PrescriptionMedicine {
	switch medicineList := raw.(type) {
	case []map[string]interface{}:
		medicines := make([]dto.PrescriptionMedicine, 0, len(medicineList))
		for _, med := range medicineList {
			medicines = append(medicines, dto.PrescriptionMedicine{
				Name:            mapString(med, "medicine_name"),
				Form:            mapString(med, "medicine_form"),
				Strength:        mapString(med, "medicine_strength"),
				Dosage:          mapString(med, "dosage"),
				Duration:        mapString(med, "duration"),
				Quantity:        mapString(med, "quantity"),
				FoodInstruction: mapString(med, "food_instruction"),
			})
		}
		return medicines
	case []interface{}:
		medicines := make([]dto.PrescriptionMedicine, 0, len(medicineList))
		for _, item := range medicineList {
			med, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			medicines = append(medicines, dto.PrescriptionMedicine{
				Name:            mapString(med, "medicine_name"),
				Form:            mapString(med, "medicine_form"),
				Strength:        mapString(med, "medicine_strength"),
				Dosage:          mapString(med, "dosage"),
				Duration:        mapString(med, "duration"),
				Quantity:        mapString(med, "quantity"),
				FoodInstruction: mapString(med, "food_instruction"),
			})
		}
		return medicines
	default:
		return nil
	}
}

func mapString(data map[string]interface{}, key string) string {
	value, _ := data[key].(string)
	return value
}
