package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// DatabaseHelper provides utility functions for database operations
type DatabaseHelper struct {
	db     *sql.DB
	logger *logrus.Logger
}

func NewDatabaseHelper(db *sql.DB, logger *logrus.Logger) *DatabaseHelper {
	return &DatabaseHelper{
		db:     db,
		logger: logger,
	}
}

// TestConnection tests the database connection and logs basic info
func (dh *DatabaseHelper) TestConnection() error {
	dh.logger.Info("Testing database connection...")

	// Test basic connectivity
	if err := dh.db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Get database version
	var version string
	if err := dh.db.QueryRow("SELECT VERSION()").Scan(&version); err != nil {
		dh.logger.WithError(err).Warn("Could not get database version")
	} else {
		dh.logger.WithField("version", version).Info("Database connected successfully")
	}

	// Check if required tables exist
	requiredTables := []string{
		"users",
		"context_alarms",
		"alarm_check_logs",
		"notification_settings",
	}

	for _, tableName := range requiredTables {
		exists, err := dh.TableExists(tableName)
		if err != nil {
			dh.logger.WithError(err).WithField("table", tableName).Error("Error checking table existence")
			continue
		}

		if !exists {
			dh.logger.WithField("table", tableName).Warn("Required table does not exist")
		} else {
			count, err := dh.GetTableRowCount(tableName)
			if err != nil {
				dh.logger.WithError(err).WithField("table", tableName).Warn("Could not get row count")
			} else {
				dh.logger.WithFields(logrus.Fields{
					"table": tableName,
					"rows":  count,
				}).Info("Table verified")
			}
		}
	}

	return nil
}

// TableExists checks if a table exists in the database
func (dh *DatabaseHelper) TableExists(tableName string) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM information_schema.tables 
		WHERE table_schema = DATABASE() 
		AND table_name = ?
	`

	var count int
	err := dh.db.QueryRow(query, tableName).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetTableRowCount returns the number of rows in a table
func (dh *DatabaseHelper) GetTableRowCount(tableName string) (int, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)

	var count int
	err := dh.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetTableStructure returns the structure of a table
func (dh *DatabaseHelper) GetTableStructure(tableName string) error {
	query := fmt.Sprintf("DESCRIBE %s", tableName)

	rows, err := dh.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to describe table %s: %w", tableName, err)
	}
	defer rows.Close()

	dh.logger.WithField("table", tableName).Info("Table structure:")

	for rows.Next() {
		var field, fieldType, null, key, defaultVal, extra sql.NullString

		err := rows.Scan(&field, &fieldType, &null, &key, &defaultVal, &extra)
		if err != nil {
			return fmt.Errorf("failed to scan table structure: %w", err)
		}

		dh.logger.WithFields(logrus.Fields{
			"table":   tableName,
			"field":   field.String,
			"type":    fieldType.String,
			"null":    null.String,
			"key":     key.String,
			"default": defaultVal.String,
			"extra":   extra.String,
		}).Info("Column info")
	}

	return rows.Err()
}

// GetActiveAlarmsSample returns a sample of active alarms for testing
func (dh *DatabaseHelper) GetActiveAlarmsSample() error {
	query := `
		SELECT id, user_id, title, description, active, last_checked
		FROM context_alarms 
		WHERE active = true
		LIMIT 5
	`

	rows, err := dh.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query active alarms: %w", err)
	}
	defer rows.Close()

	dh.logger.Info("Sample active alarms:")

	count := 0
	for rows.Next() {
		var id, userID, title, description string
		var active bool
		var lastChecked sql.NullTime

		err := rows.Scan(&id, &userID, &title, &description, &active, &lastChecked)
		if err != nil {
			return fmt.Errorf("failed to scan alarm row: %w", err)
		}

		lastCheckedStr := "never"
		if lastChecked.Valid {
			lastCheckedStr = lastChecked.Time.Format(time.RFC3339)
		}

		dh.logger.WithFields(logrus.Fields{
			"id":           id,
			"user_id":      userID,
			"title":        title,
			"description":  description,
			"active":       active,
			"last_checked": lastCheckedStr,
		}).Info("Active alarm")

		count++
	}

	if count == 0 {
		dh.logger.Info("No active alarms found")
	}

	return rows.Err()
}

// GetUsersSample returns a sample of users for testing
func (dh *DatabaseHelper) GetUsersSample() error {
	query := `
		SELECT email, first_name, last_name, phone_number, phone_verified
		FROM users 
		LIMIT 5
	`

	rows, err := dh.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	dh.logger.Info("Sample users:")

	count := 0
	for rows.Next() {
		var email, firstName, lastName, phoneNumber string
		var phoneVerified bool

		err := rows.Scan(&email, &firstName, &lastName, &phoneNumber, &phoneVerified)
		if err != nil {
			return fmt.Errorf("failed to scan user row: %w", err)
		}

		dh.logger.WithFields(logrus.Fields{
			"email":          email,
			"name":           fmt.Sprintf("%s %s", firstName, lastName),
			"phone_number":   phoneNumber,
			"phone_verified": phoneVerified,
		}).Info("User")

		count++
	}

	if count == 0 {
		dh.logger.Info("No users found")
	}

	return rows.Err()
}
