package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

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
