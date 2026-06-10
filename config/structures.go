package config

type AppEnv string

var (
	ConfigLocal AppEnv = "local"
	ConfigProd  AppEnv = "prod"
	ConfigStg   AppEnv = "stg"
)

type Config struct {
	AppEnv         string
	ServerPort     string
	DatabaseURL    string
	PrivateKeyPath string
	PublicKeyPath  string
}
