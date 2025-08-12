package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config holds all configuration for the alarm checker
type Config struct {
	Database DatabaseConfig `json:"database"`
	Redis    RedisConfig    `json:"redis"`
	Gemini   GeminiConfig   `json:"gemini"`
	Twilio   TwilioConfig   `json:"twilio"`
	App      AppConfig      `json:"app"`
}

type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type RedisConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Password string `json:"password"`
}

type GeminiConfig struct {
	APIKey string `json:"api_key"`
	Model  string `json:"model"`
}

type TwilioConfig struct {
	AccountSID  string `json:"account_sid"`
	AuthToken   string `json:"auth_token"`
	PhoneNumber string `json:"phone_number"`
}

type AppConfig struct {
	CheckInterval  string `json:"check_interval"` // e.g., "5m"
	LogLevel       string `json:"log_level"`      // debug, info, warn, error
	LogFile        string `json:"log_file"`
	MaxConcurrent  string `json:"max_concurrent"`  // e.g., "4"
	RequestTimeout string `json:"request_timeout"` // e.g., "45s"
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", ""),
			Port:     getEnv("DB_PORT", "3306"),
			User:     getEnv("DB_USER", ""),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", ""),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", ""),
			Port:     getEnv("REDIS_PORT", "6380"),
			Password: getEnv("REDIS_PASSWORD", ""),
		},
		Gemini: GeminiConfig{
			APIKey: getEnv("GEMINI_API_KEY", ""),
			Model:  getEnv("GEMINI_MODEL", "gemini-1.5-flash"),
		},
		Twilio: TwilioConfig{
			AccountSID:  getEnv("TWILIO_ACCOUNT_SID", ""),
			AuthToken:   getEnv("TWILIO_AUTH_TOKEN", ""),
			PhoneNumber: getEnv("TWILIO_PHONE_NUMBER", ""),
		},
		App: AppConfig{
			CheckInterval:  getEnv("CHECK_INTERVAL", "5m"),
			LogLevel:       getEnv("LOG_LEVEL", "info"),
			LogFile:        getEnv("LOG_FILE", "alarm_checker.log"),
			MaxConcurrent:  getEnv("MAX_CONCURRENT", "4"),
			RequestTimeout: getEnv("REQUEST_TIMEOUT", "45s"),
		},
	}

	// Validate required fields
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// Validate checks if all required configuration values are present
func (c *Config) Validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("DB_USER is required")
	}
	if c.Database.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	if c.Gemini.APIKey == "" {
		return fmt.Errorf("GEMINI_API_KEY is required")
	}
	if c.Twilio.AccountSID == "" {
		return fmt.Errorf("TWILIO_ACCOUNT_SID is required")
	}
	if c.Twilio.AuthToken == "" {
		return fmt.Errorf("TWILIO_AUTH_TOKEN is required")
	}
	if c.Twilio.PhoneNumber == "" {
		return fmt.Errorf("TWILIO_PHONE_NUMBER is required")
	}
	return nil
}

// ToJSON converts config to JSON string for logging
func (c *Config) ToJSON() string {
	// Create a copy without sensitive data
	safeCopy := *c
	safeCopy.Database.Password = "***"
	safeCopy.Gemini.APIKey = "***"
	safeCopy.Twilio.AuthToken = "***"
	safeCopy.Redis.Password = "***"

	data, _ := json.MarshalIndent(safeCopy, "", "  ")
	return string(data)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
