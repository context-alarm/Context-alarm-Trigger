package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// ScheduledChecker runs the alarm checker at regular intervals
type ScheduledChecker struct {
	checker  *AlarmChecker
	interval time.Duration
	logger   *logrus.Logger
	quit     chan bool
}

func NewScheduledChecker(checker *AlarmChecker, interval time.Duration, logger *logrus.Logger) *ScheduledChecker {
	return &ScheduledChecker{
		checker:  checker,
		interval: interval,
		logger:   logger,
		quit:     make(chan bool),
	}
}

func (sc *ScheduledChecker) Start() {
	sc.logger.WithField("interval", sc.interval).Info("Starting scheduled alarm checker")

	ticker := time.NewTicker(sc.interval)
	defer ticker.Stop()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run initial check
	sc.checker.checkAlarms()

	for {
		select {
		case <-ticker.C:
			sc.checker.checkAlarms()
		case <-sigChan:
			sc.logger.Info("Received shutdown signal, stopping alarm checker")
			return
		case <-sc.quit:
			sc.logger.Info("Stopping scheduled alarm checker")
			return
		}
	}
}

func (sc *ScheduledChecker) Stop() {
	close(sc.quit)
}

// Add this to the main function to run as a scheduler
func runAsScheduler() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	logger.SetLevel(logrus.InfoLevel)

	// Create log file
	logFile, err := os.OpenFile("alarm_checker_scheduler.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logger.Fatal("Failed to open log file:", err)
	}
	defer logFile.Close()
	logger.SetOutput(logFile)

	logger.Info("Starting Context Alarm Scheduler...")

	// Initialize database connection
	db, err := initDatabase()
	if err != nil {
		logger.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Initialize Gemini REST client with search capabilities
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		logger.Fatal("GEMINI_API_KEY not found in environment")
	}
	geminiRestClient := NewGeminiRestClient(apiKey, "gemini-2.5-flash", logger)

	// Initialize Twilio client
	twilioClient := initTwilioClient()

	// Create alarm checker
	checker := &AlarmChecker{
		db:               db,
		geminiRestClient: geminiRestClient,
		twilioClient:     twilioClient,
		logger:           logger,
	}

	// Create scheduled checker (runs every 15 minutes)
	scheduler := NewScheduledChecker(checker, 15*time.Minute, logger)

	// Start the scheduler
	scheduler.Start()
}
