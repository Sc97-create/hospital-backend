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
	smtpHost := viper.GetString("SMTP_HOST")
	smtpPassword := viper.GetString("SMTP_PASSWORD")
	smtpPort := viper.GetInt("SMTP_PORT")
	smtpUsername := viper.GetString("SMTP_USERNAME")
	razorpayCallbackUrl := viper.GetString("RAZORPAY_CALLBACK_URL")
	razorpayWebhookSecret := viper.GetString("RAZORPAY_WEBHOOK_SECRET")
	razorpayApiKey := viper.GetString("RAZORPAY_API_KEY")
	razorpayApiSecret := viper.GetString("RAZORPAY_API_SECRET")
	razorpayBaseUrl := viper.GetString("RAZORPAY_BASE_URL")
	if razorpayBaseUrl == "" {
		razorpayBaseUrl = "https://api.razorpay.com/v1"
	}

	appEnv := viper.GetString("APP_ENV")
	env := viper.GetString("ENV")
	if env == "" {
		env = appEnv
	}
	if env == "" {
		env = "development"
	}
	logLevel := viper.GetString("LOG_LEVEL")

	return &Config{
		AppEnv:         appEnv,
		Env:            env,
		LogLevel:       logLevel,
		ServerPort:     port,
		DatabaseURL:    viper.GetString("DATABASE_URL"),
		PrivateKeyPath: viper.GetString("PRIVATE_KEY_PATH"),
		PublicKeyPath:  viper.GetString("PUBLIC_KEY_PATH"),
		NotificationConfig: NotificationConfig{
			SMTPHost:     smtpHost,
			SMTPPort:     smtpPort,
			SMTPUsername: smtpUsername,
			SMTPPassword: smtpPassword,
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
