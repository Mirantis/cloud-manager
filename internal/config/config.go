package config

import (
	"os"
)

// Config holds application configuration
type Config struct {
	Port          string
	AWSRegion     string
	RoleName      string
	AdminUsername string
	AdminPassword string
	DBPath        string
}

// LoadConfig creates and returns application configuration from environment variables
func LoadConfig() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	roleName := os.Getenv("IAM_ORG_ROLE_NAME")
	if roleName == "" {
		roleName = "IAMManagerCrossAccountRole"
	}

	adminUsername := os.Getenv("ADMIN_USERNAME")
	if adminUsername == "" {
		adminUsername = "admin"
	}

	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin" // Default for development only
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./iam-manager.db"
	}

	return Config{
		Port:          port,
		AWSRegion:     region,
		RoleName:      roleName,
		AdminUsername: adminUsername,
		AdminPassword: adminPassword,
		DBPath:        dbPath,
	}
}
