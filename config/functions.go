package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

func Load() (*Config, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	filep := filepath.Join(dir, "deploy/local/.env")
	err = godotenv.Load(filep)
	if err != nil {
		log.Println("err", err)
	}

	viper.AutomaticEnv()

	port := viper.GetString("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	appointmentcreated := viper.GetString("APPOINTMENT_CREATED")
	//appointmentupdated := viper.GetString("APPOINTMENT_UPDATED")
	patientcreated := viper.GetString("PATIENT_CREATED")
	patientupdated := viper.GetString("PATIENT_UPDATED")
	appointmentreminder := viper.GetString("APPOINTMENT_REMINDER")
	prescriptioncreated := viper.GetString("PRESCRIPTION_CREATED")
	paymentrecieved := viper.GetString("PAYMENT_RECIEVED")
	medicineadherence := viper.GetString("MEDICINE_ADHERENCE")
	follwupReminder := viper.GetString("FOLLOW_UP_REMINDER")
	smtpHost := viper.GetString("SMTP_HOST")
	smtpPassword := viper.GetString("SMTP_PASSWORD")
	smtpPort := viper.GetInt("SMTP_PORT")
	smtpUsername := viper.GetString("SMTP_USERNAME")

	return &Config{
		AppEnv:         viper.GetString("APP_ENV"),
		ServerPort:     port,
		DatabaseURL:    viper.GetString("DATABASE_URL"),
		PrivateKeyPath: viper.GetString("PRIVATE_KEY_PATH"),
		PublicKeyPath:  viper.GetString("PUBLIC_KEY_PATH"),
		TemplatePath: NotificationTemplateFilepath{
			Appointmentcreated: appointmentcreated,
			//AppointmentUpdated:  appointmentupdated,
			Patientcreated:      patientcreated,
			PaymentRecieved:     paymentrecieved,
			FollowUpReminder:    follwupReminder,
			MedicationAdherence: medicineadherence,
			PrescriptionCreated: prescriptioncreated,
			AppointmentReminder: appointmentreminder,
			PatientUpdated:      patientupdated,
		},
		NotificationConfig: NotificationConfig{
			SMTPHost:     smtpHost,
			SMTPPort:     smtpPort,
			SMTPUsername: smtpUsername,
			SMTPPassword: smtpPassword,
		},
	}, nil
}
