package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	GoogleClientID     string
	GoogleClientSecret string
	GoogleCallbackUrl  string
	JwtSecret          string
	DbUrl              string
	Port               string
	Environment        string
}

func Load() (*Config, error) {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "dev"
	}
	if env == "dev" {
		//create a fake env variable from .env files and use them
		err := godotenv.Load()
		if err != nil {
			slog.Error("environment_var_load_failure", "error", err)
			return nil, fmt.Errorf("failed to load the .env file %w", err)
		}
	}

	config := &Config{
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleCallbackUrl:  os.Getenv("GOOGLE_CALLBACK_URL"),
		JwtSecret:          os.Getenv("JWT_SECRET"),
		DbUrl:              os.Getenv("DB_URL"),
		Port:               os.Getenv("PORT"),
		Environment:        env,
	}

	if config.GoogleClientID == "" {
		//this should not happen - os should always return non empty string values when set in enviroment
		return nil, fmt.Errorf("config_load_failed_empty_env_values : GOOGLE_CLIENT_ID")
	}

	if config.DbUrl == "" {
		//this should not happen - os should always return non empty string values when set in enviroment
		return nil, fmt.Errorf("config_load_failed_empty_env_values : DB_URL")
	}

	if config.Port == "" {
		config.Port = ":8080"
		slog.Error("missing_env_value : PORT, set to default 8080")
	}

	return config, nil

}

func (c *Config) IsProd() bool {
	return c.Environment == "prod"
}

func (c *Config) GetJWTKey() []byte {
	return []byte(c.JwtSecret)
}
