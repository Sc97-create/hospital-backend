package config

// SampleNotificationConfig provides an example of how to configure the notification system
// This is for reference only and should be customized based on your email provider

var SampleNotificationConfig = NotificationConfig{
	// SMTP server configuration
	SMTPHost:     "smtp.gmail.com", // or your email provider's SMTP host
	SMTPPort:     587,              // or 465 for SSL, 587 for TLS
	SMTPUsername: "your-email@gmail.com",
	SMTPPassword: "your-app-specific-password", // Use app-specific password for Gmail

	// Email sender configuration
	FromName: "Hospital Notification System",

	// Retry configuration
	MaxRetries:   3,  // Maximum number of retry attempts
	RetryBackoff: 15, // Minutes to wait before retry
}

// Usage Example:
// In your main.go or initialization code:
//
// config := &Config{
//     AppEnv:         "prod",
//     ServerPort:     "8080",
//     DatabaseURL:    "postgres://...",
//     PrivateKeyPath: "/path/to/private/key",
//     PublicKeyPath:  "/path/to/public/key",
//     NotificationConfig: NotificationConfig{
//         SMTPHost:     "smtp.gmail.com",
//         SMTPPort:     587,
//         SMTPUsername: os.Getenv("SMTP_USERNAME"),
//         SMTPPassword: os.Getenv("SMTP_PASSWORD"),
//         FromEmail:    "noreply@hospital.com",
//         FromName:     "Hospital",
//         MaxRetries:   3,
//         RetryBackoff: 15,
//     },
// }
//
// // Create notification container
// notificationContainer, err := container.NewNotificationContainer(db, config.NotificationConfig)
// if err != nil {
//     log.Fatal(err)
// }
//
// // Start the worker to process pending notifications
// go notificationContainer.Start(context.Background())
//
// // Create a new notification
// err = notificationContainer.Service.Create(context.Background(), dto.CreateRequest{
//     OrganisationID:   1,
//     PatientID:        ptrTo(uint64(123)),
//     Recipient:        "patient@example.com",
//     Subject:          "Appointment Reminder",
//     Content:          "<h1>Your appointment is tomorrow</h1>",
//     NotificationType: "APPOINTMENT_REMINDER",
// })
// if err != nil {
//     log.Fatal(err)
// }
