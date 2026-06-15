package dto

type CreateRequest struct {
	NotificationType string
	Data             any
	Subject          string
}
