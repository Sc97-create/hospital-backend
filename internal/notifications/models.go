package notifications

import "time"

type ChannelType string

type NotificationStatus string

type Notification struct {
	ID               string             `json:"id" gorm:"not null;type:uuid"`
	OrganisationID   string             `json:"organisation_id" gorm:"type:uuid;not null;primarykey"`
	PatientID        string             `json:"patient_id" gorm:"type:uuid;not null"`
	NotificationType string             `json:"notification_type" gorm:"column:notification_type"`
	Channel          ChannelType        `json:"channel" gorm:"column:channel"`
	Content          string             `json:"content" gorm:"column:content"`
	Status           NotificationStatus `json:"status" gorm:"default:PENDING"`
	RetryCount       int                `json:"retry_count" gorm:"default:0"`
	NextRetryAt      time.Time          `json:"next_retry_at" gorm:"type:timestamp"`
	LastError        *string            `json:"last_error"`
	CreatedAt        time.Time          `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time          `json:"updated_at" gorm:"autoCreateTime"`
	SentAt           *time.Time         `json:"sent_at" gorm:"type:timestamp"`
	Subject          string             `json:"subject" gorm:"type:text"`
}
type NotificationAttempts struct {
	ID             string    `json:"id" gorm:"type:uuid;primaryKey"`
	NotificationID string    `json:"notification_id" gorm:"column:notification_id;not null;type:uuid"`
	AttemptNumber  int       `json:"attempt_no" gorm:"column:attempt_no;default:0"`
	Status         string    `json:"status" gorm:"column:status"`
	ErrorMessage   *string   `json:"error_msg" gorm:"column:error_msg"`
	CreatedAt      time.Time `json:"created_at" gorm:"type:timestamp;autoCreateTime"`
}
