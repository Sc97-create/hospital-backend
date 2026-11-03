package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

func Load() (*Config, error) {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" || appEnv == "local" {
		dir, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		filep := filepath.Join(dir, "deploy/local/.env")
		err = godotenv.Load(filep)
		if err != nil {
			log.Println("err", err)
		}
	}

	viper.AutomaticEnv()

	port := viper.GetString("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	smtpHost := strings.TrimSpace(viper.GetString("SMTP_HOST"))
	smtpPassword := strings.TrimSpace(viper.GetString("SMTP_PASSWORD"))
	smtpPort := viper.GetInt("SMTP_PORT")
	smtpUsername := strings.TrimSpace(viper.GetString("SMTP_USERNAME"))
	mailFromName := strings.TrimSpace(viper.GetString("MAIL_FROM_NAME"))
	mailFromEmail := strings.TrimSpace(viper.GetString("MAIL_FROM_EMAIL"))
	mailReplyTo := strings.TrimSpace(viper.GetString("MAIL_REPLY_TO"))
	if mailFromEmail == "" {
		mailFromEmail = smtpUsername
	}
	razorpayCallbackUrl := viper.GetString("RAZORPAY_CALLBACK_URL")
	razorpayWebhookSecret := viper.GetString("RAZORPAY_WEBHOOK_SECRET")
	razorpayApiKey := viper.GetString("RAZORPAY_API_KEY")
	razorpayApiSecret := viper.GetString("RAZORPAY_API_SECRET")
	razorpayBaseUrl := viper.GetString("RAZORPAY_BASE_URL")
	if razorpayBaseUrl == "" {
		razorpayBaseUrl = "https://api.razorpay.com/v1"
	}

	//appEnv := viper.GetString("APP_ENV")
	env := viper.GetString("ENV")
	if env == "" {
		env = appEnv
	}
	if env == "" {
		env = "development"
	}
	logLevel := viper.GetString("LOG_LEVEL")

	passwordResetBaseURL := viper.GetString("PASSWORD_RESET_BASE_URL")
	if passwordResetBaseURL == "" {
		passwordResetBaseURL = "http://localhost:5173"
	}

	internalBasicAuthID := viper.GetString("CLIENT_ID")
	internalBasicAuthSecret := viper.GetString("CLIENT_SECRET")
	internalAPIBaseURL := viper.GetString("INTERNAL_API_BASE_URL")
	if internalAPIBaseURL == "" {
		internalAPIBaseURL = "http://127.0.0.1:" + port
	}

	return &Config{
		AppEnv:               appEnv,
		Env:                  env,
		LogLevel:             logLevel,
		ServerPort:           port,
		DatabaseURL:          viper.GetString("DATABASE_URL"),
		PrivateKeyPath:       viper.GetString("PRIVATE_KEY_PATH"),
		PublicKeyPath:        viper.GetString("PUBLIC_KEY_PATH"),
		LoginUrl:             viper.GetString("LOGIN_URL"),
		PasswordResetBaseURL: passwordResetBaseURL,
		InternalAPIBaseURL:   internalAPIBaseURL,
		InternalBasicAuth: InternalBasicAuth{
			ID:     internalBasicAuthID,
			Secret: internalBasicAuthSecret,
		},
		NotificationConfig: NotificationConfig{
			SMTPHost:     smtpHost,
			SMTPPort:     smtpPort,
			SMTPUsername: smtpUsername,
			SMTPPassword: smtpPassword,
			FromName:     mailFromName,
			FromEmail:    mailFromEmail,
			ReplyTo:      mailReplyTo,
		},
		RazorPayClient: RazorPayClient{
			CallbackUrl: razorpayCallbackUrl,
			RPayConfig: RazorpayConfig{
				ApiKey:    razorpayApiKey,
				ApiSecret: razorpayApiSecret,
				BaseUrl:   razorpayBaseUrl,
			},
			WebhookSecret: razorpayWebhookSecret,
		},
	}, nil
}
