package prescription

import (
	"context"

	"hospital-backend/internal/appointments"
	notificationdto "hospital-backend/internal/notifications/dto"
	"hospital-backend/internal/prescription/dto"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type NotificationEnqueuer interface {
	Create(ctx context.Context, data notificationdto.CreateRequest) error
}

type AppointmentLookup interface {
	GetAppntmentByID(log *zap.Logger, id string) (appointments.Appointment, error)
}

type AppointmentStatusUpdater interface {
	UpdateStatusInTx(log *zap.Logger, tx *gorm.DB, status string, appointmentID string) error
}

type PrescriptionItemAdder interface {
	AddItems(log *zap.Logger, db *gorm.DB, medicine []dto.MedicineArray, prescriptionID string, userID string) error
}
