package notifications

import (
	"encoding/json"
	"fmt"
	"time"
)

type ChannelType string

type NotificationStatus string

type Notification struct {
	ID               string             `json:"id" gorm:"not null;type:uuid;primarykey"`
	OrganisationID   string             `json:"organisation_id" gorm:"type:uuid;not null;"`
	PatientID        string             `json:"patient_id" gorm:"type:uuid;not null"`
	NotificationType string             `json:"notification_type" gorm:"column:notification_type"`
	Status           NotificationStatus `json:"status" gorm:"default:PENDING"`
	RetryCount       int                `json:"retry_count" gorm:"default:0"`
	NextRetryAt      time.Time          `json:"next_retry_at" gorm:"type:timestamp"`
	LastError        *string            `json:"last_error"`
	CreatedAt        time.Time          `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time          `json:"updated_at" gorm:"autoCreateTime"`
	SentAt           *time.Time         `json:"sent_at" gorm:"type:timestamp"`
	ProviderPayload  PPayload           `json:"p_payload" gorm:"type:jsonb"`
}

type PPayload struct {
	Content        string      `json:"content" gorm:"column:content"`
	Channel        ChannelType `json:"channel" gorm:"column:channel"`
	Subject        string      `json:"subject" gorm:"column:subject"`
	RecipientEmail string      `json:"recipient_email" gorm:"column:recipient_email"`
}
type NotificationAttempts struct {
	ID             string    `json:"id" gorm:"type:uuid;primaryKey"`
	NotificationID string    `json:"notification_id" gorm:"column:notification_id;not null;type:uuid"`
	AttemptNumber  int       `json:"attempt_no" gorm:"column:attempt_no;default:0"`
	Status         string    `json:"status" gorm:"column:status"`
	ErrorMessage   *string   `json:"error_msg" gorm:"column:error_msg"`
	CreatedAt      time.Time `json:"created_at" gorm:"type:timestamp;autoCreateTime"`
}

func (s *PPayload) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, &s)
	case string:
		return json.Unmarshal([]byte(v), &s)
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}
