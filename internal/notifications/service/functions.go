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
	subject, content, err := s.renderer.Render(data.NotificationType, notificationdata)
	if err != nil {
		return err
	}
	notification := &notifications.Notification{
		ID:               uuid.New().String(),
		OrganisationID:   notificationdata.OrganisationID,
		PatientID:        notificationdata.PatientID,
		NotificationType: data.NotificationType,
		Channel:          notifications.ChannelType(notifications.EmailChannel),
		Content:          content,
		Status:           notifications.NotificationStatus(notifications.PendingStatus),
		RetryCount:       0,
		Subject:          subject,
		NextRetryAt:      time.Now(),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
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
	}
	return notificationData
}
