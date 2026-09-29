package config

type AppEnv string

var (
	ConfigLocal AppEnv = "local"
	ConfigProd  AppEnv = "prod"
	ConfigStg   AppEnv = "stg"
)

type Config struct {
	AppEnv               string
	Env                  string // production | development (from ENV, falls back to APP_ENV)
	LogLevel             string // debug | info | warn | error (from LOG_LEVEL)
	ServerPort           string
	DatabaseURL          string
	PrivateKeyPath       string
	PublicKeyPath        string
	NotificationConfig   NotificationConfig
	RazorPayClient       RazorPayClient
	InternalBasicAuth    InternalBasicAuth
	InternalAPIBaseURL   string
	LoginUrl             string
	PasswordResetBaseURL string
}
type RazorPayClient struct {
	CallbackUrl   string
	RPayConfig    RazorpayConfig
	WebhookSecret string
}
type RazorpayConfig struct {
	BaseUrl   string
	ApiKey    string
	ApiSecret string
}

// InternalBasicAuth is the global Basic credential used by internal service APIs.
// Values come from CLIENT_ID and CLIENT_SECRET. Send as: Authorization: Basic base64(id:secret)
type InternalBasicAuth struct {
	ID     string
	Secret string
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
