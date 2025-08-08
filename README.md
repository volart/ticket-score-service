# Ticket Score Service

A gRPC service for rating analytics, ticket scoring, and overall quality assessment with SQLite database.

## Features

- **Rating Analytics Service**: Category-based score aggregation with daily/weekly analytics
- **Ticket Scores Service**: Ticket scoring with server-side streaming
- **Overall Quality Service**: Concurrent weighted quality score calculation with pagination
- **Period Comparison Service**: Period-over-period score comparison with relative percentage change

## Database

The service should be using provided sample data from SQLite database (`database.db`). The file should be placed in the root folder of the project.

## Quick Start

### Using Makefile
```bash
# Install tools
make install-proto-tools

# Generate proto files
make proto

# Start the server
make start

# Run tests
make test

# See all available commands
make help

# Build and run locally
make build
make run

# Build and run with Docker
make docker-build

# Run in background with docker
make docker-compose-up
```

## Development

### gRPC Code Generation

#### Using Makefile
```bash
# Install tools
make install-proto-tools

# Generate proto files
make proto

# Clean generated files
make clean-proto
```

**Note:** Generated files in `proto/generated/` are ignored by git and should be regenerated locally.

### Project Structure

```
ticket-score-service/
├── cmd/server/          # Main application entry point
├── internal/
│   ├── app/            # Application bootstrap and dependency initialization
│   ├── config/         # Configuration management
│   ├── database/       # Database connection and setup
│   ├── models/         # Data models
│   ├── repository/     # Data access layer
│   ├── server/         # gRPC server implementations
│   ├── service/        # Business logic layer
│   └── utils/          # Utility functions
├── proto/              # Protocol buffer definitions
│   ├── generated/      # Generated Go code (not in git)
│   ├── rating_analytics.proto
│   ├── ticket_scores.proto
│   ├── overall_quality.proto
│   └── period_comparison.proto
└── database.db         # SQLite database file (not included, purchase separately :) )
```

## Testing gRPC API

### Using grpcurl
Install grpcurl:
```bash
brew install grpcurl

# List available services
grpcurl -plaintext localhost:50051 list
```

### Rating Analytics Service

```bash
# List methods for RatingAnalyticsService
grpcurl -plaintext localhost:50051 list rating_analytics.RatingAnalyticsService

# Get category analytics (daily scores - short range)
grpcurl -plaintext -d '{
  "start_date": "2019-10-01",
  "end_date": "2019-10-03"
}' localhost:50051 rating_analytics.RatingAnalyticsService/GetCategoryAnalytics

# Get category analytics (weekly scores - long range)
grpcurl -plaintext -d '{
  "start_date": "2019-10-01",
  "end_date": "2019-11-03"
}' localhost:50051 rating_analytics.RatingAnalyticsService/GetCategoryAnalytics
```

**Response format:**
```json
{
  "analytics": [
    {
      "category": "Spelling",
      "ratings": 150,
      "dates": [
        {
          "date": "2019-10-01",
          "score": "85%"
        },
        {
          "date": "2019-10-02",
          "score": "78%"
        },
        {
          "date": "2019-10-03",
          "score": "N/A"
        }
      ],
      "score": "82%"
    },
    {
      "category": "Grammar",
      "ratings": 120,
      "dates": [
        {
          "date": "2019-10-01",
          "score": "90%"
        },
        {
          "date": "2019-10-02",
          "score": "88%"
        },
        {
          "date": "2019-10-03",
          "score": "N/A"
        }
      ],
      "score": "89%"
    }
  ]
}
```

**Features:**
- Date ranges ≤ 30 days return daily scores, ranges > 30 days return weekly scores
- Daily format: `"2019-10-01"`, Weekly format: `"2019-10-01 to 2019-10-07"`
- Scores formatted as percentages (e.g., "85%") or "N/A" when no data available
- Overall score calculated across entire date range for each category

### Ticket Scores Service

```bash
# List methods for TicketScoresService
grpcurl -plaintext localhost:50051 list ticket_scores.TicketScoresService

# Get ticket scores (server-side streaming)
grpcurl -plaintext -d '{
  "start_date": "2019-10-01",
  "end_date": "2019-10-03"
}' localhost:50051 ticket_scores.TicketScoresService/GetTicketScores
```

**Response format (server-side streaming):**
Each streamed message has this structure:
```json
{
  "ticketId": 123,
  "categories": [
    {
      "categoryName": "Spelling",
      "score": "85%"
    },
    {
      "categoryName": "Grammar",
      "score": "92%"
    },
    {
      "categoryName": "Tone",
      "score": "N/A"
    }
  ]
}
```

**Features:**
- Server-side streaming for efficient processing of large datasets
- Concurrent processing with goroutine pool
- Scores formatted as percentages (e.g., "85%") or "N/A" when no data available
- Each ticket includes all available categories for consistent response structure

### Overall Quality Service

```bash
# List methods for OverallQualityService
grpcurl -plaintext localhost:50051 list overall_quality.OverallQualityService

# Get overall quality score
grpcurl -plaintext -d '{
  "start_date": "2019-10-01",
  "end_date": "2019-10-07"
}' localhost:50051 overall_quality.OverallQualityService/GetOverallQualityScore
```

**Response format:**
```json
{
  "period": "2019-10-01 to 2019-10-07",
  "score": "85%"
}
```

**Features:**
- Concurrent processing with goroutine pool (default: 10)
- Chunked pagination for large datasets (default chunk size: 1000)
- Weighted scoring based on category weights
- Handles empty result sets gracefully (returns "N/A" for score)
- Simplified response with only essential fields

### Period Comparison Service

```bash
# List methods for PeriodComparisonService
grpcurl -plaintext localhost:50051 list period_comparison.PeriodComparisonService

# Get period comparison (week over week)
grpcurl -plaintext -d '{
  "starting_date": "2019-10-01",
  "period_type": "WEEK"
}' localhost:50051 period_comparison.PeriodComparisonService/GetPeriodComparison

# Get period comparison (month over month)
grpcurl -plaintext -d '{
  "starting_date": "2019-10-01",
  "period_type": "MONTH"
}' localhost:50051 period_comparison.PeriodComparisonService/GetPeriodComparison

# Get period comparison (quarter over quarter)
grpcurl -plaintext -d '{
  "starting_date": "2019-01-01",
  "period_type": "QUARTER"
}' localhost:50051 period_comparison.PeriodComparisonService/GetPeriodComparison

# Get period comparison (year over year)
grpcurl -plaintext -d '{
  "starting_date": "2019-01-01",
  "period_type": "YEAR"
}' localhost:50051 period_comparison.PeriodComparisonService/GetPeriodComparison
```

**Response format:**
```json
{
  "start_period": "2019-10-08 to 2019-10-14",
  "start_score": "89%",
  "end_period": "2019-10-01 to 2019-10-07",
  "end_score": "82%",
  "difference": "+8.5%"
}
```

**Features:**
- Only requires starting date and period type
- Calculates consecutive periods
- Shows true percentage change, not just difference
- Supports WEEK, MONTH, QUARTER, and YEAR comparisons
- Period Order: `start_period` = most recent period, `end_period` = older period

**Period Calculation Examples:**
- **WEEK**: `2019-10-01` → Period 1: `2019-10-01 to 2019-10-07`, Period 2: `2019-10-08 to 2019-10-14`
- **MONTH**: `2019-10-01` → Period 1: `2019-10-01 to 2019-10-31`, Period 2: `2019-11-01 to 2019-11-30`
- **QUARTER**: `2019-01-01` → Period 1: `2019-01-01 to 2019-03-31`, Period 2: `2019-04-01 to 2019-06-30`
- **YEAR**: `2019-01-01` → Period 1: `2019-01-01 to 2019-12-31`, Period 2: `2020-01-01 to 2020-12-31`

## Testing

```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose
