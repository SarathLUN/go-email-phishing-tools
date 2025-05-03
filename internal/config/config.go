package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBPath            string
	SMTPHost          string
	SMTPPort          int
	SMTPUser          string
	SMTPPassword      string
	SMTPSenderAddress string
	TrackerHost       string
	TrackerPort       int
	TrackerBaseURL    string
	EmailSubject      string
	EmailTemplatePath string
}

func LoadConfig(path string) (*Config, error) {
	// If path is empty, try loading .env from current dir, but don't fail if missing
	if path == "" {
		_ = godotenv.Load() // Ignore error if .env doesn't exist
	} else {
		err := godotenv.Load(path)
		if err != nil {
			log.Printf("Warning: Error loading .env file from %s: %v", path, err)
			// Continue, maybe env vars are set directly
		}
	}

	smtpPortStr := getEnv("SMTP_PORT", "587")
	smtpPort, err := strconv.Atoi(smtpPortStr)
	if err != nil {
		log.Printf("Warning: Invalid SMTP_PORT value '%s', using default 587. Error: %v", smtpPortStr, err)
		smtpPort = 587
	}

	trackerPortStr := getEnv("TRACKER_PORT", "8080")
	trackerPort, err := strconv.Atoi(trackerPortStr)
	if err != nil {
		log.Printf("Warning: Invalid TRACKER_PORT value '%s', using default 8080. Error: %v", trackerPortStr, err)
		trackerPort = 8080
	}

	cfg := &Config{
		DBPath:            getEnv("DB_PATH", "./phishing_simulation.db"),
		SMTPHost:          getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:          smtpPort,
		SMTPUser:          getEnv("SMTP_USER", ""),
		SMTPPassword:      getEnv("SMTP_PASSWORD", ""),
		SMTPSenderAddress: getEnv("SMTP_SENDER_ADDRESS", ""),
		TrackerHost:       getEnv("TRACKER_HOST", "localhost"),
		TrackerPort:       trackerPort,
		TrackerBaseURL:    getEnv("TRACKER_BASE_URL", "http://localhost:"+trackerPortStr),
		EmailSubject:      getEnv("EMAIL_SUBJECT", "Important Security Update"),
		EmailTemplatePath: getEnv("EMAIL_TEMPLATE_PATH", "./configs/email_template.html"),
	}

	// Basic validation for critical SMTP settings for later stages
	if cfg.SMTPUser == "" || cfg.SMTPPassword == "" || cfg.SMTPSenderAddress == "" {
		log.Println("Warning: SMTP configuration (USER, PASSWORD, SENDER_ADDRESS) is incomplete in .env file.")
	}

	return cfg, nil
}

// Helper function to get env var or default
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("Using fallback for env var %s", key)
	return fallback
}
