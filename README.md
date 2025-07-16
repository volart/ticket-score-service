# Ticket Score Service

A gRPC service for rating analytics, ticket scoring, and overall quality assessment with SQLite database backend.

## Features

- **Rating Analytics Service**: Category-based score aggregation with daily/weekly analytics
- **Ticket Scores Service**: Ticket scoring with server-side streaming
- **Overall Quality Service**: Concurrent weighted quality score calculation with pagination

## Database

The service should be using provided sample data from SQLite database (`database.db`). The file should be placed in the root folder of the project.

## To run:
  ### Build and run locally
  go run cmd/server/main.go

  ### Build and run with Docker
  docker build -t ticket-score-service .
  docker run -p 50051:50051 ticket-score-service

  ### Run in background with docker
  docker-compose up -d

## Development

### gRPC Code Generation

To regenerate the gRPC code from proto files:

```bash
# Install required tools
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate Go code from proto files
export PATH=$PATH:$(go env GOPATH)/bin
protoc --go_out=. --go-grpc_out=. proto/generated/rating_analytics/rating_analytics.proto
protoc --go_out=. --go-grpc_out=. proto/generated/ticket_scores/ticket_scores.proto
protoc --go_out=. --go-grpc_out=. proto/generated/overall_quality/overall_quality.proto
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
│   └── overall_quality.proto
└── database.db         # SQLite database file (not included, purchase separately :) )
```

## Testing gRPC API

### Using grpcurl
Install grpcurl:
```bash
brew install grpcurl
```

```bash
# List available services
grpcurl -plaintext localhost:50051 list

### Rating Analytics Service

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

## Testing

Run the test suite:

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for specific package
go test -v ./internal/service
go test -v ./internal/server
```
