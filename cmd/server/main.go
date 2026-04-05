package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/rusik69/aws-iam-manager/internal/config"
	"github.com/rusik69/aws-iam-manager/internal/server"
)

func main() {
	log.Printf("[INFO] Starting AWS IAM Manager server...")
	log.Printf("[INFO] Application version: v1.0.0")

	// Load .env.prod file if it exists (for local development)
	// In production (Docker/K8s), environment variables are injected directly
	if _, err := os.Stat(".env.prod"); err == nil {
		if err := godotenv.Load(".env.prod"); err != nil {
			log.Printf("[WARNING] Error loading .env.prod file: %v", err)
		} else {
			log.Printf("[INFO] Loaded environment variables from .env.prod")
		}
	} else if _, err := os.Stat(".env"); err == nil {
		// Fallback to .env if .env.prod doesn't exist
		if err := godotenv.Load(".env"); err != nil {
			log.Printf("[WARNING] Error loading .env file: %v", err)
		} else {
			log.Printf("[INFO] Loaded environment variables from .env")
		}
	}

	cfg := config.LoadConfig()
	log.Printf("[INFO] Configuration loaded successfully")

	srv, err := server.NewServer(cfg)
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize server: %v", err)
	}
	log.Printf("[INFO] Server instance created")

	if err := srv.Start(); err != nil {
		log.Fatalf("[FATAL] Failed to start server: %v", err)
	}
}