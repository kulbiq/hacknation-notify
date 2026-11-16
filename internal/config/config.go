package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DiscordWebhookURL   string
	TargetURL           string
	CheckInterval       time.Duration
	DBPath              string
	NotifyOnInitialRun  bool
	SendStartupMessage  bool
	DailySummaryEnabled bool
	DailySummaryHour    int
}

func LoadConfig() *Config {
	// Try loading .env file if present
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("Note: .env file not found or could not be loaded, using environment variables: %v", err)
	}

	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")

	targetURL := os.Getenv("TARGET_URL")
	if targetURL == "" {
		targetURL = "https://hacknation.pl/zadania"
	}

	intervalStr := os.Getenv("CHECK_INTERVAL")
	interval := 15 * time.Minute
	if intervalStr != "" {
		if d, err := time.ParseDuration(intervalStr); err == nil && d > 0 {
			interval = d
		} else {
			log.Printf("Invalid CHECK_INTERVAL '%s', falling back to default 15m", intervalStr)
		}
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/hacknation.db"
	}

	notifyInitial := false
	if val := os.Getenv("NOTIFY_ON_INITIAL_RUN"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			notifyInitial = b
		}
	}

	sendStartup := true
	if val := os.Getenv("SEND_STARTUP_MESSAGE"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			sendStartup = b
		}
	}

	dailySummaryEnabled := true
	if val := os.Getenv("DAILY_SUMMARY_ENABLED"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			dailySummaryEnabled = b
		}
	}

	dailySummaryHour := 10 // Domyślnie 10:00
	if val := os.Getenv("DAILY_SUMMARY_HOUR"); val != "" {
		if h, err := strconv.Atoi(val); err == nil && h >= 0 && h <= 23 {
			dailySummaryHour = h
		}
	}

	return &Config{
		DiscordWebhookURL:   webhookURL,
		TargetURL:           targetURL,
		CheckInterval:       interval,
		DBPath:              dbPath,
		NotifyOnInitialRun:  notifyInitial,
		SendStartupMessage:  sendStartup,
		DailySummaryEnabled: dailySummaryEnabled,
		DailySummaryHour:    dailySummaryHour,
	}
}
