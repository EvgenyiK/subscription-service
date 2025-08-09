package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	ServerPort string
}

func LoadConfig() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	config := &Config{
		DBHost:     viper.GetString("DB_HOST"),
		DBPort:     viper.GetString("DB_PORT"),
		DBUser:     viper.GetString("DB_USER"),
		DBPassword: viper.GetString("DB_PASSWORD"),
		DBName:     viper.GetString("DB_NAME"),
		ServerPort: viper.GetString("SERVER_PORT"),
	}

	return config, nil
}
