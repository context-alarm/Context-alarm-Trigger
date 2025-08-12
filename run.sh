#!/bin/bash

# Context Alarm Checker Runner Script

# Set working directory
cd "$(dirname "$0")"

# Check if .env file exists
if [ ! -f .env ]; then
    echo "Error: .env file not found!"
    exit 1
fi

# Function to run once
run_once() {
    echo "Running alarm checker once..."
    go run main.go config.go database_helper.go gemini_rest.go
}

# Function to run as daemon
run_daemon() {
    echo "Starting alarm checker daemon (checking every 5 minutes)..."
    go run main.go config.go database_helper.go gemini_rest.go scheduler.go -daemon
}

# Function to test database connection
test_db() {
    echo "Testing database connection..."
    go run -ldflags "-X main.mode=test" main.go config.go database_helper.go gemini_rest.go
}

# Function to install dependencies
install_deps() {
    echo "Installing Go dependencies..."
    go mod tidy
    go mod download
}

# Function to build the application
build() {
    echo "Building application..."
    go build -o alarm-checker main.go config.go database_helper.go gemini_rest.go scheduler.go
}

# Function to show help
show_help() {
    echo "Context Alarm Checker"
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  once       - Run alarm check once and exit"
    echo "  daemon     - Run as daemon (checks every 5 minutes)"
    echo "  test       - Test database connection"
    echo "  install    - Install dependencies"
    echo "  build      - Build the application"
    echo "  help       - Show this help message"
    echo ""
    echo "Default: run once"
}

# Parse command line arguments
case "${1:-once}" in
    "once")
        install_deps
        run_once
        ;;
    "daemon")
        install_deps
        run_daemon
        ;;
    "test")
        install_deps
        test_db
        ;;
    "install")
        install_deps
        ;;
    "build")
        install_deps
        build
        ;;
    "help"|"-h"|"--help")
        show_help
        ;;
    *)
        echo "Unknown command: $1"
        show_help
        exit 1
        ;;
esac
