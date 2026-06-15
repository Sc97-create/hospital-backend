package config

type AppEnv string

var (
	ConfigLocal AppEnv = "local"
	ConfigProd  AppEnv = "prod"
	ConfigStg   AppEnv = "stg"
)

type Config struct {
	AppEnv             string
	ServerPort         string
	DatabaseURL        string
	PrivateKeyPath     string
	PublicKeyPath      string
	NotificationConfig NotificationConfig
	TemplatePath       NotificationTemplateFilepath
}

type NotificationConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromName     string
	MaxRetries   int
	RetryBackoff int // in minutes
}
type NotificationTemplateFilepath struct {
	Patientcreated     string
	PatientUpdated     string
	Appointmentcreated string
	//AppointmentUpdated  string
	PrescriptionCreated string
	MedicationAdherence string
	FollowUpReminder    string
	PaymentRecieved     string
	AppointmentReminder string
}
