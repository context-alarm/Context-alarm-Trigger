package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
	"google.golang.org/api/option"
)

// Alarm represents a context alarm from the database
type Alarm struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	ContextData   string     `json:"context_data"`
	ConditionData string     `json:"condition_data"`
	TimeWindow    string     `json:"time_window"`
	Active        bool       `json:"active"`
	LastChecked   *time.Time `json:"last_checked"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// User represents user information from the database
type User struct {
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Email         string    `json:"email"`
	UniqueID      string    `json:"unique_id"`
	PhoneNumber   string    `json:"phone_number"`
	PhoneVerified bool      `json:"phone_verified"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CheckResult represents the result of checking an alarm condition
type CheckResult struct {
	AlarmID      string                 `json:"alarm_id"`
	ConditionMet bool                   `json:"condition_met"`
	ResultData   map[string]interface{} `json:"result_data"`
	ErrorMessage string                 `json:"error_message,omitempty"`
}

// AlarmChecker handles the main logic for checking alarms
type AlarmChecker struct {
	db               *sql.DB
	geminiRestClient *GeminiRestClient
	twilioClient     *twilio.RestClient
	logger           *logrus.Logger
	config           *Config
}

var mode string

func main() {
	// Load environment variables if a local .env is present; otherwise rely on process env
	_ = godotenv.Load()

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	// Set log level
	switch config.App.LogLevel {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	// Create log file
	logFile, err := os.OpenFile(config.App.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logger.Fatal("Failed to open log file:", err)
	}
	defer logFile.Close()
	logger.SetOutput(logFile)

	logger.Info("Starting Context Alarm Checker...")
	logger.Debug("Configuration loaded:", config.ToJSON())

	// Initialize database connection
	db, err := initDatabaseWithConfig(config)
	if err != nil {
		logger.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Create database helper for testing
	dbHelper := NewDatabaseHelper(db, logger)

	// Check if we're in test mode
	if mode == "test" {
		logger.Info("Running in test mode...")
		if err := dbHelper.TestConnection(); err != nil {
			logger.Fatal("Database connection test failed:", err)
		}

		// Show table structures
		tables := []string{"users", "context_alarms", "alarm_check_logs"}
		for _, table := range tables {
			if exists, _ := dbHelper.TableExists(table); exists {
				dbHelper.GetTableStructure(table)
			}
		}

		// Show sample data
		dbHelper.GetActiveAlarmsSample()
		dbHelper.GetUsersSample()

		logger.Info("Test mode completed successfully")
		return
	}

	// Initialize Gemini REST client with search capabilities
	geminiRestClient := NewGeminiRestClient(config.Gemini.APIKey, config.Gemini.Model, logger)

	// Initialize Twilio client
	twilioClient := initTwilioClientWithConfig(config)

	// Create alarm checker
	checker := &AlarmChecker{
		db:               db,
		geminiRestClient: geminiRestClient,
		twilioClient:     twilioClient,
		logger:           logger,
		config:           config,
	}

	// Test database connection
	if err := dbHelper.TestConnection(); err != nil {
		logger.Fatal("Database connection test failed:", err)
	}

	// Start the alarm checking process
	checker.checkAlarms()
}

func initDatabaseWithConfig(config *Config) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&tls=skip-verify",
		config.Database.User, config.Database.Password,
		config.Database.Host, config.Database.Port, config.Database.Name)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

func initDatabase() (*sql.DB, error) {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&tls=skip-verify",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

func initGeminiClientWithConfig(config *Config) (*genai.Client, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(config.Gemini.APIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return client, nil
}

func initGeminiClient() (*genai.Client, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not found in environment")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return client, nil
}

func initTwilioClientWithConfig(config *Config) *twilio.RestClient {
	return twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: config.Twilio.AccountSID,
		Password: config.Twilio.AuthToken,
	})
}

func initTwilioClient() *twilio.RestClient {
	accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")

	return twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSid,
		Password: authToken,
	})
}

func (ac *AlarmChecker) checkAlarms() {
	ac.logger.Info("Starting alarm check process...")

	// Fetch active alarms
	alarms, err := ac.fetchActiveAlarms()
	if err != nil {
		ac.logger.WithError(err).Error("Failed to fetch active alarms")
		return
	}

	ac.logger.WithField("count", len(alarms)).Info("Fetched active alarms")

	// Concurrency settings
	maxConcurrent := 4
	if ac.config != nil && ac.config.App.MaxConcurrent != "" {
		if v, err := strconv.Atoi(ac.config.App.MaxConcurrent); err == nil && v > 0 {
			maxConcurrent = v
		}
	}
	requestTimeout := 45 * time.Second
	if ac.config != nil && ac.config.App.RequestTimeout != "" {
		if d, err := time.ParseDuration(ac.config.App.RequestTimeout); err == nil && d > 0 {
			requestTimeout = d
		}
	}

	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, alarm := range alarms {
		sem <- struct{}{}
		wg.Add(1)
		go func(al Alarm) {
			defer wg.Done()
			defer func() { <-sem }()

			ac.logger.WithField("alarm_id", al.ID).Info("Processing alarm")

			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			defer cancel()

			// Check alarm condition using Gemini (with retry)
			result := ac.checkAlarmConditionWithRetry(ctx, al)

			// Log the check result
			if err := ac.logAlarmCheck(al.ID, result); err != nil {
				ac.logger.WithError(err).WithField("alarm_id", al.ID).Error("Failed to log alarm check")
			}

			// If condition is met, make phone call
			if result.ConditionMet {
				user, err := ac.getUserByEmail(al.UserID)
				if err != nil {
					ac.logger.WithError(err).WithField("user_id", al.UserID).Error("Failed to fetch user")
				} else if user.PhoneVerified && user.PhoneNumber != "" {
					ac.makePhoneCall(user, al)
				} else {
					ac.logger.WithField("user_id", al.UserID).Warn("User phone not verified or missing")
				}
			}

			// Update last checked time
			if err := ac.updateLastChecked(al.ID); err != nil {
				ac.logger.WithError(err).WithField("alarm_id", al.ID).Error("Failed to update last checked time")
			}
		}(alarm)
	}

	wg.Wait()

	ac.logger.Info("Completed alarm check process")
}

// checkAlarmConditionWithRetry retries transient Gemini errors with exponential backoff
func (ac *AlarmChecker) checkAlarmConditionWithRetry(ctx context.Context, alarm Alarm) CheckResult {
	backoff := 500 * time.Millisecond
	maxBackoff := 4 * time.Second
	attempts := 0

	for {
		attempts++
		select {
		case <-ctx.Done():
			return CheckResult{AlarmID: alarm.ID, ErrorMessage: "request timeout"}
		default:
		}

		res := ac.checkAlarmCondition(alarm)
		if res.ErrorMessage == "" {
			return res
		}
		// Retry on likely transient errors
		if !isTransientError(res.ErrorMessage) || attempts >= 4 {
			return res
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			res.ErrorMessage = "timeout while retrying: " + res.ErrorMessage
			return res
		case <-timer.C:
			if backoff < maxBackoff {
				backoff *= 2
			}
		}
	}
}

func isTransientError(msg string) bool {
	m := strings.ToLower(msg)
	return strings.Contains(m, "429") || strings.Contains(m, "resource_exhausted") ||
		strings.Contains(m, "deadline exceeded") || strings.Contains(m, "timeout") ||
		strings.Contains(m, "temporarily") || strings.Contains(m, "internal error") ||
		strings.Contains(m, "unavailable")
}

func (ac *AlarmChecker) fetchActiveAlarms() ([]Alarm, error) {
	// Query to fetch alarms that are active and within their time window (start_time <= NOW() <= end_time)
	// Use DB server time directly to avoid CONVERT_TZ dependency on timezone tables
	query := `
		SELECT id, user_id, title, description, context_data, condition_data, 
		       time_window, active, last_checked, created_at, updated_at
		FROM context_alarms 
		WHERE active = true 
		AND NOW() BETWEEN start_time AND end_time
	`

	rows, err := ac.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active alarms: %w", err)
	}
	defer rows.Close()

	var alarms []Alarm
	for rows.Next() {
		var alarm Alarm
		var lastChecked sql.NullTime

		err := rows.Scan(
			&alarm.ID, &alarm.UserID, &alarm.Title, &alarm.Description,
			&alarm.ContextData, &alarm.ConditionData, &alarm.TimeWindow,
			&alarm.Active, &lastChecked, &alarm.CreatedAt, &alarm.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alarm row: %w", err)
		}

		if lastChecked.Valid {
			alarm.LastChecked = &lastChecked.Time
		}

		alarms = append(alarms, alarm)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating alarm rows: %w", err)
	}

	return alarms, nil
}

func (ac *AlarmChecker) getUserByEmail(email string) (*User, error) {
	query := `
		SELECT first_name, last_name, email, unique_id, phone_number, 
		       phone_verified, created_at, updated_at
		FROM users 
		WHERE email = ?
	`

	var user User
	err := ac.db.QueryRow(query, email).Scan(
		&user.FirstName, &user.LastName, &user.Email, &user.UniqueID,
		&user.PhoneNumber, &user.PhoneVerified, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	return &user, nil
}

func (ac *AlarmChecker) isWithinTimeWindow(timeWindow string) bool {
	if timeWindow == "" {
		return true // No time restriction
	}

	now := time.Now()
	currentTime := now.Format("15:04")
	currentDay := strings.ToLower(now.Weekday().String())

	// Handle different time window formats
	timeWindow = strings.ToLower(timeWindow)

	// Check for weekday restrictions
	if strings.Contains(timeWindow, "weekdays") {
		if currentDay == "saturday" || currentDay == "sunday" {
			return false
		}
	} else if strings.Contains(timeWindow, "weekends") {
		if currentDay != "saturday" && currentDay != "sunday" {
			return false
		}
	}

	// Extract time range (e.g., "9am-5pm", "09:00-17:00")
	timeWindow = strings.ReplaceAll(timeWindow, "weekdays", "")
	timeWindow = strings.ReplaceAll(timeWindow, "weekends", "")
	timeWindow = strings.TrimSpace(timeWindow)

	if timeWindow == "" {
		return true
	}

	// Parse time range
	parts := strings.Split(timeWindow, "-")
	if len(parts) != 2 {
		ac.logger.WithField("time_window", timeWindow).Warn("Invalid time window format")
		return true
	}

	startTime := ac.parseTime(strings.TrimSpace(parts[0]))
	endTime := ac.parseTime(strings.TrimSpace(parts[1]))

	if startTime == "" || endTime == "" {
		return true
	}

	return currentTime >= startTime && currentTime <= endTime
}

func (ac *AlarmChecker) parseTime(timeStr string) string {
	timeStr = strings.ToLower(timeStr)

	// Handle AM/PM format
	if strings.Contains(timeStr, "am") || strings.Contains(timeStr, "pm") {
		timeStr = strings.ReplaceAll(timeStr, "am", "")
		timeStr = strings.ReplaceAll(timeStr, "pm", "")
		timeStr = strings.TrimSpace(timeStr)

		// Convert to 24-hour format
		hour, err := time.Parse("3", timeStr)
		if err != nil {
			hour, err = time.Parse("15", timeStr)
			if err != nil {
				return ""
			}
		}

		if strings.Contains(strings.ToLower(timeStr), "pm") && hour.Hour() != 12 {
			return fmt.Sprintf("%02d:00", hour.Hour()+12)
		} else if strings.Contains(strings.ToLower(timeStr), "am") && hour.Hour() == 12 {
			return "00:00"
		}

		return fmt.Sprintf("%02d:00", hour.Hour())
	}

	// Handle 24-hour format
	if strings.Contains(timeStr, ":") {
		return timeStr
	}

	return ""
}

func (ac *AlarmChecker) checkAlarmCondition(alarm Alarm) CheckResult {
	ac.logger.WithField("alarm_id", alarm.ID).Info("Checking alarm condition with Gemini REST API (with Google Search)")

	// Create the prompt for Gemini with instructions to access current data
	prompt := ac.buildGeminiPrompt(alarm)

	// Use the REST client with Google Search capabilities
	result, err := ac.geminiRestClient.GenerateContentWithSearch(prompt)
	if err != nil {
		result.AlarmID = alarm.ID
		result.ErrorMessage = fmt.Sprintf("Gemini REST API error: %v", err)
		ac.logger.WithError(err).WithField("alarm_id", alarm.ID).Error("Failed to call Gemini REST API")
		return result
	}

	// Set the alarm ID
	result.AlarmID = alarm.ID

	ac.logger.WithFields(logrus.Fields{
		"alarm_id":      alarm.ID,
		"condition_met": result.ConditionMet,
		"used_search":   result.ResultData["used_search"],
		"response_data": result.ResultData,
	}).Info("Alarm condition check completed with Google Search")

	return result
}

func (ac *AlarmChecker) buildGeminiPrompt(alarm Alarm) string {
	currentTime := time.Now().Format("January 2, 2006 at 3:04 PM MST")

	// TEMPORARY: Force true condition for test_weather_now alarm to test Twilio
	if alarm.ID == "test_weather_now" {
		return fmt.Sprintf(`
You are a context alarm checker with access to real-time information. It is currently %s.

TESTING MODE: This is a test alarm for Twilio phone call functionality.

Alarm Details:
- Title: %s
- Description: %s

INSTRUCTIONS:
This is a TEST ALARM designed to always trigger a phone call to test the Twilio integration.
Please return a JSON response indicating the condition is met.

REQUIRED RESPONSE FORMAT (JSON only):
{
  "condition_met": true,
  "current_data": {
    "searched_information": "Test condition - always true for Twilio testing",
    "data_source": "Test Mode",
    "timestamp": "%s",
    "key_values": {"test_mode": true}
  },
  "reason": "This is a test alarm designed to always trigger a Twilio phone call for testing purposes."
}
`, currentTime, alarm.Title, alarm.Description, currentTime)
	}

	return fmt.Sprintf(`
Check if this condition is met: %s

Search for current data and return JSON:
{
  "condition_met": true/false,
  "current_data": {
    "searched_information": "what you found",
    "data_source": "source",
    "timestamp": "when updated",
    "key_values": {}
  },
  "reason": "explanation"
}
`, alarm.ConditionData)
}

func (ac *AlarmChecker) logAlarmCheck(alarmID string, result CheckResult) error {
	status := "not_met"
	if result.ConditionMet {
		status = "met"
	}
	if result.ErrorMessage != "" {
		status = "error"
	}

	resultDataJSON, _ := json.Marshal(result.ResultData)

	query := `
		INSERT INTO alarm_check_logs (id, alarm_id, check_time, status, result_data, error_message)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	logID := fmt.Sprintf("log-%d", time.Now().UnixNano())

	_, err := ac.db.Exec(query, logID, alarmID, time.Now(), status, string(resultDataJSON), result.ErrorMessage)
	if err != nil {
		return fmt.Errorf("failed to insert alarm check log: %w", err)
	}

	return nil
}

func (ac *AlarmChecker) makePhoneCall(user *User, alarm Alarm) {
	ac.logger.WithFields(logrus.Fields{
		"user_id":      user.Email,
		"phone_number": user.PhoneNumber,
		"alarm_id":     alarm.ID,
	}).Info("Making phone call notification")

	message := fmt.Sprintf("Hello %s, this is your Context Alarm notification. Your alarm '%s' condition has been met. %s",
		user.FirstName, alarm.Title, alarm.Description)

	ac.logger.WithFields(logrus.Fields{
		"message":  message,
		"alarm_id": alarm.ID,
	}).Info("Generated phone call message")

	params := &twilioApi.CreateCallParams{}
	params.SetTo(user.PhoneNumber)
	params.SetFrom(os.Getenv("TWILIO_PHONE_NUMBER"))
	params.SetTwiml(fmt.Sprintf(`<Response><Say>%s</Say></Response>`, message))

	resp, err := ac.twilioClient.Api.CreateCall(params)
	if err != nil {
		ac.logger.WithError(err).WithField("user_id", user.Email).Error("Failed to make phone call")
		return
	}

	ac.logger.WithFields(logrus.Fields{
		"user_id":  user.Email,
		"call_sid": *resp.Sid,
		"alarm_id": alarm.ID,
	}).Info("Phone call initiated successfully")
}

func (ac *AlarmChecker) updateLastChecked(alarmID string) error {
	query := `UPDATE context_alarms SET last_checked = ? WHERE id = ?`

	_, err := ac.db.Exec(query, time.Now(), alarmID)
	if err != nil {
		return fmt.Errorf("failed to update last_checked: %w", err)
	}

	return nil
}
