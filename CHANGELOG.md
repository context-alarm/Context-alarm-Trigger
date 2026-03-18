# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- GitHub-ready project structure
- Comprehensive documentation
- CI/CD workflowss
- Docker support
- Makefile for common tasks
- Contributing guidelines
- Code of conduct

### Changed
- Improved error handling and logging
- Enhanced configuration management
- Better database connection handling

## [1.0.0] - 2025-01-06

### Added
- Initial release of Context Alarm Checker
- MySQL database integration for alarm storage
- Google Gemini AI integration for intelligent condition checking
- Twilio integration for phone call notifications
- Real-time web search capabilities
- Time window filtering for alarms
- Comprehensive logging system
- Configuration management with environment variables
- Database helper utilities
- Scheduler for continuous monitoring
- Test mode for database validation

### Features
- **Alarm Management**: Create and manage context-based alarms
- **AI-Powered Checking**: Use Gemini AI to evaluate real-world conditions
- **Phone Notifications**: Automatic phone calls when conditions are met
- **Time Windows**: Schedule alarms for specific time periods
- **User Management**: Support for multiple users with phone verification
- **Logging**: Comprehensive audit trail of all operations
- **Error Handling**: Robust error handling and recovery

### Technical Details
- Built with Go 1.21+
- Uses MySQL 8.0+ for data persistence
- Integrates with Google Gemini AI API
- Uses Twilio for phone call automation
- Structured logging with logrus
- Environment-based configuration
- Docker support for containerization

## [0.1.0] - 2025-01-01

### Added
- Basic alarm checking functionality
- Database schema for users and alarms
- Simple condition evaluation
- Basic logging

---

## Version History

- **1.0.0**: First stable release with full feature set
- **0.1.0**: Initial development version

## Migration Guide

### From 0.1.0 to 1.0.0

1. **Database Schema Changes**:
   - New columns added to `context_alarms` table
   - New `alarm_check_logs` table created
   - Run `update_alarm.sql` to apply changes

2. **Configuration Changes**:
   - New environment variables for Gemini and Twilio
   - Updated configuration structure
   - See README.md for new configuration options

3. **API Changes**:
   - Enhanced alarm checking with AI capabilities
   - Improved error handling and logging
   - New phone notification system

## Support

For support and questions:
- Create an issue on GitHub
- Check the documentation in README.md
- Review the contributing guidelines in CONTRIBUTING.md 