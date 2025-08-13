# 🚨 Context Alarm Checker

## 📋 Overview

The Context Alarm Checker is a Go-based program that monitors and evaluates context-based alarms stored in a MySQL database. When alarm conditions are met, it automatically initiates phone calls to users via the Twilio API. The program uses Google's Gemini AI to intelligently assess whether real-world conditions match the user-defined alarm criteria.

## 🎯 How It Works

### 1. **Database Connection & Discovery**
- Connects to Azure MySQL database using provided credentials
- Discovers and validates table structures (`users`, `context_alarms`, `alarm_check_logs`)
- Verifies data integrity and logs table statistics

### 2. **Active Alarm Retrieval**
- Fetches all alarms where `active = true` from the `context_alarms` table
- Filters alarms based on current time being between `start_time` and `end_time` datetime fields
- Only processes alarms that are currently within their scheduled time window

### 3. **Intelligent Condition Checking with Real-Time Data**
- For each active alarm, sends the context and condition data to **Gemini AI**
- Gemini has access to current real-time information and can search the web
- Analyzes current data (weather, news, stock prices, sports scores, etc.)
- Uses enhanced prompts that explicitly request current, real-time information
- Returns a structured JSON response indicating if the condition is met

### 4. **Phone Call Notifications**
- When conditions are met, retrieves user information
- Validates phone number verification status
- Uses **Twilio API** to initiate automated phone calls
- Delivers personalized voice messages about the triggered alarm

### 5. **Comprehensive Logging**
- Logs all activities to `alarm_checker.log` in JSON format
- Records alarm check results in the `alarm_check_logs` database table
- Tracks success/failure rates and system performance

## 🗂️ Database Structure

Based on the actual database inspection, here's what was found:

### Users Table
```sql
- first_name: varchar(255)
- last_name: varchar(255) 
- email: varchar(255) [PRIMARY KEY]
- unique_id: varchar(255)
- phone_number: varchar(20) [UNIQUE]
- phone_verified: tinyint(1)
- verification_attempts: int
- created_at: timestamp
- updated_at: timestamp
- last_verification_attempt: timestamp
```

### Context Alarms Table
```sql
- id: varchar(50) [PRIMARY KEY]
- user_id: varchar(255) [FOREIGN KEY -> users.email]
- title: varchar(255)
- description: text
- context_data: text (JSON string)
- condition_data: text (JSON string)
- time_window: varchar(255)
- active: tinyint(1)
- last_checked: timestamp
- start_time: datetime
- end_time: datetime
- created_at: timestamp
- updated_at: timestamp
```

### Alarm Check Logs Table
```sql
- id: varchar(50) [PRIMARY KEY]
- alarm_id: varchar(50) [FOREIGN KEY -> context_alarms.id]
- check_time: timestamp
- status: enum('pending','checking','met','not_met','error')
- result_data: text
- error_message: text
```

## 🔧 Program Components

### 1. **main.go** - Core Logic
- **AlarmChecker struct**: Main orchestrator
- **Database operations**: Connection, queries, updates
- **Gemini integration**: AI-powered condition evaluation
- **Twilio integration**: Phone call automation
- **Time window parsing**: Intelligent scheduling

### 2. **config.go** - Configuration Management
- Loads environment variables
- Validates required settings
- Provides safe configuration logging (masks sensitive data)

### 3. **database_helper.go** - Database Utilities
- Connection testing and validation
- Table structure inspection
- Sample data retrieval for debugging

### 4. **scheduler.go** - Automated Execution
- Runs alarm checks at regular intervals (default: 5 minutes)
- Handles graceful shutdown signals
- Continuous monitoring mode

## 🚀 Running the Program

### Test Mode (Database Connection Check)
```bash
# Test database connectivity and show table structures
./run.sh test

# Or manually:
go run -ldflags "-X main.mode=test" main.go config.go database_helper.go
```

### Single Execution
```bash
# Run alarm check once and exit
./run.sh once

# Or manually:
go run main.go config.go database_helper.go
```

### Daemon Mode (Continuous Monitoring)
```bash
# Run as background service checking every 5 minutes
./run.sh daemon

# Or manually:
go run main.go config.go database_helper.go scheduler.go -daemon
```

## 📊 Current Database Status

Based on the latest test run:
- **Users**: 2 active users with verified phone numbers
- **Active Alarms**: 2 alarms currently being monitored
  - UFC 319 End Alarm (user: nipunapamuditha234@gmail.com)
  - Bitcoin hits $100k (user: nipunapamuditha234@gmail.com)
- **Check Logs**: 0 entries (fresh system)

## 🔍 Example Alarm Processing Flow

1. **Alarm Retrieved**: "Bitcoin hits $100k"
   - Context Data: `{"cryptocurrency": "bitcoin", "target_price": 100000}`
   - Condition Data: `{"price_threshold": "$100,000", "trigger": "greater_than"}`

2. **Gemini Analysis**: 
   - Checks current Bitcoin price from real-time data
   - Compares against $100,000 threshold
   - Returns structured JSON response

3. **Decision Making**:
   - If Bitcoin > $100k → Condition Met = true
   - If Bitcoin < $100k → Condition Met = false

4. **Action Taken**:
   - **Condition Met**: Call user at +14373762044
   - **Not Met**: Log result and continue monitoring

## 📝 Log Output Example

```json
{
  "level": "info",
  "msg": "Checking alarm condition with Gemini API",
  "alarm_id": "ca_1754436328959041832",
  "time": "2025-08-06T11:17:47-04:00"
}
{
  "level": "info", 
  "msg": "Alarm condition check completed",
  "alarm_id": "ca_1754436328959041832",
  "condition_met": false,
  "time": "2025-08-06T11:17:56-04:00"
}
```

## 🔧 Configuration

All configuration is loaded from `.env` file:

```env
# Database (Azure MySQL)
DB_HOST=mysql-gemini-free.mysql.database.azure.com
DB_PORT=3306
DB_USER=geminiadmin
DB_PASSWORD=GeminiPass123!
DB_NAME=contextalarm

# Gemini AI
GEMINI_API_KEY=AIzaSyDxeqzMcg1uHo31xAVauEN9_2hyMPsW2os
GEMINI_MODEL=gemini-2.5-flash

# Twilio (Phone Calls)
TWILIO_ACCOUNT_SID=AC6e89aadda2b91b3cfe75ba0a73753a37
TWILIO_AUTH_TOKEN=65b62e7b7739705ed3e7a631cbc084b5
TWILIO_PHONE_NUMBER=+12313105343
```

## 🎯 Key Features

- ✅ **Real-time monitoring** of context-based alarms
- ✅ **AI-powered condition evaluation** using Gemini
- ✅ **Automated phone notifications** via Twilio
- ✅ **Intelligent time window filtering**
- ✅ **Comprehensive logging** and audit trails
- ✅ **Database connection resilience**
- ✅ **Graceful error handling**
- ✅ **Configurable scheduling intervals**

## 🔮 Alarm Types Supported

The system can handle various alarm types:
- **Weather conditions**: Rain, temperature, storm alerts
- **Financial markets**: Stock prices, cryptocurrency values
- **Sports events**: Game endings, score thresholds
- **News monitoring**: Breaking news, keyword alerts
- **Custom conditions**: Any real-world data Gemini can access

## 🚨 Error Handling

The program handles various error scenarios:
- Database connection failures
- Gemini API timeouts or errors
- Twilio call failures
- Invalid time window formats
- Missing user phone numbers
- Unverified phone numbers

## 📈 Performance

- **Average alarm check**: 2-10 seconds (depending on Gemini response time)
- **Database queries**: Optimized with proper indexing
- **Memory usage**: Minimal (stateless operation)
- **Concurrent processing**: Handles multiple alarms sequentially

This system provides a robust, intelligent alarm monitoring solution that bridges real-world conditions with automated user notifications.
