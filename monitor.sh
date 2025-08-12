#!/bin/bash

# Context Alarm Checker - Monitoring Script
# This script helps monitor the alarm checker's activity

LOG_FILE="alarm_checker.log"
WATCH_INTERVAL=5

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "🚨 Context Alarm Checker Monitor"
echo "=================================="
echo ""

# Function to show current database status
show_db_status() {
    echo -e "${BLUE}📊 Current Database Status:${NC}"
    echo "Checking database connection..."
    
    if go run -ldflags "-X main.mode=test" main.go config.go database_helper.go >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Database: Connected${NC}"
        
        # Extract info from log
        active_alarms=$(tail -20 "$LOG_FILE" | grep "Fetched active alarms" | tail -1 | grep -o '"count":[0-9]*' | cut -d':' -f2)
        if [ ! -z "$active_alarms" ]; then
            echo -e "${GREEN}📋 Active Alarms: $active_alarms${NC}"
        fi
        
        users_count=$(tail -50 "$LOG_FILE" | grep "Table verified.*users" | tail -1 | grep -o '"rows":[0-9]*' | cut -d':' -f2)
        if [ ! -z "$users_count" ]; then
            echo -e "${GREEN}👥 Users: $users_count${NC}"
        fi
    else
        echo -e "${RED}❌ Database: Connection Failed${NC}"
    fi
    echo ""
}

# Function to show recent activity
show_recent_activity() {
    echo -e "${BLUE}📝 Recent Activity (Last 10 entries):${NC}"
    if [ -f "$LOG_FILE" ]; then
        tail -10 "$LOG_FILE" | jq -r 'select(.level != "debug") | "\(.time) [\(.level | ascii_upcase)] \(.msg)"' 2>/dev/null || \
        tail -10 "$LOG_FILE" | grep -o '"time":"[^"]*".*"level":"[^"]*".*"msg":"[^"]*"' | sed 's/"time":"//; s/","level":"/] [/; s/","msg":"/ /; s/"$//'
    else
        echo "No log file found yet."
    fi
    echo ""
}

# Function to show alarm check results
show_alarm_results() {
    echo -e "${BLUE}🎯 Recent Alarm Check Results:${NC}"
    if [ -f "$LOG_FILE" ]; then
        # Show condition check results
        grep "condition_met" "$LOG_FILE" | tail -5 | while read line; do
            alarm_id=$(echo "$line" | grep -o '"alarm_id":"[^"]*"' | cut -d'"' -f4)
            condition_met=$(echo "$line" | grep -o '"condition_met":[^,}]*' | cut -d':' -f2)
            time=$(echo "$line" | grep -o '"time":"[^"]*"' | cut -d'"' -f4)
            
            if [ "$condition_met" = "true" ]; then
                echo -e "${GREEN}✅ $time - Alarm $alarm_id: TRIGGERED${NC}"
            else
                echo -e "${YELLOW}⏳ $time - Alarm $alarm_id: Not triggered${NC}"
            fi
        done
    else
        echo "No alarm results found yet."
    fi
    echo ""
}

# Function to run a single check
run_single_check() {
    echo -e "${BLUE}🔄 Running Single Alarm Check...${NC}"
    echo "Starting alarm check process..."
    
    if go run main.go config.go database_helper.go; then
        echo -e "${GREEN}✅ Alarm check completed successfully${NC}"
        show_alarm_results
    else
        echo -e "${RED}❌ Alarm check failed${NC}"
    fi
}

# Function to watch logs in real-time
watch_logs() {
    echo -e "${BLUE}👀 Watching logs in real-time (Press Ctrl+C to stop)...${NC}"
    echo ""
    
    if [ -f "$LOG_FILE" ]; then
        tail -f "$LOG_FILE" | while read line; do
            level=$(echo "$line" | grep -o '"level":"[^"]*"' | cut -d'"' -f4)
            msg=$(echo "$line" | grep -o '"msg":"[^"]*"' | cut -d'"' -f4)
            time=$(echo "$line" | grep -o '"time":"[^"]*"' | cut -d'"' -f4)
            
            case "$level" in
                "error")
                    echo -e "${RED}[ERROR] $time - $msg${NC}"
                    ;;
                "warn"|"warning")
                    echo -e "${YELLOW}[WARN]  $time - $msg${NC}"
                    ;;
                "info")
                    echo -e "${GREEN}[INFO]  $time - $msg${NC}"
                    ;;
                *)
                    echo "[${level^^}]  $time - $msg"
                    ;;
            esac
        done
    else
        echo "Log file not found. Run an alarm check first."
    fi
}

# Function to show help
show_help() {
    echo "Context Alarm Checker Monitor"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  status     - Show current database and system status"
    echo "  check      - Run a single alarm check"
    echo "  watch      - Watch logs in real-time"
    echo "  activity   - Show recent activity from logs"
    echo "  results    - Show recent alarm check results"
    echo "  help       - Show this help message"
    echo ""
    echo "Default: show status and recent activity"
}

# Main script logic
case "${1:-status}" in
    "status")
        show_db_status
        show_recent_activity
        ;;
    "check")
        run_single_check
        ;;
    "watch")
        watch_logs
        ;;
    "activity")
        show_recent_activity
        ;;
    "results")
        show_alarm_results
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
