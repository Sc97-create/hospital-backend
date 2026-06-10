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

	return &Config{
		AppEnv:         viper.GetString("APP_ENV"),
		ServerPort:     port,
		DatabaseURL:    viper.GetString("DATABASE_URL"),
		PrivateKeyPath: viper.GetString("PRIVATE_KEY_PATH"),
		PublicKeyPath:  viper.GetString("PUBLIC_KEY_PATH"),
	}, nil
}
