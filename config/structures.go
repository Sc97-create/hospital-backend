package config

type AppEnv string

var (
	ConfigLocal AppEnv = "local"
	ConfigProd  AppEnv = "prod"
	ConfigStg   AppEnv = "stg"
)

type Config struct {
	AppEnv             string
	Env                string // production | development (from ENV, falls back to APP_ENV)
	LogLevel           string // debug | info | warn | error (from LOG_LEVEL)
	ServerPort         string
	DatabaseURL        string
	PrivateKeyPath     string
	PublicKeyPath      string
	NotificationConfig NotificationConfig
	RazorPayClient     RazorPayClient
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

type NotificationConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromName     string
	MaxRetries   int
	RetryBackoff int // in minutes
}
