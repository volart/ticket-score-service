# Ticket Score Service

A gRPC service for rating analytics and ticket scoring with SQLite database backend.

## Features

- **Rating Analytics Service**: Category-based score aggregation with daily/weekly analytics
- **Ticket Scores Service**: Ticket scoring with server-side streaming

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
protoc --go_out=. --go-grpc_out=. proto/rating_analytics.proto
protoc --go_out=. --go-grpc_out=. proto/ticket_scores.proto
```

**Note:** Generated files in `proto/generated/` are ignored by git and should be regenerated locally.

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

**Note:** Date ranges ≤ 30 days return daily scores, ranges > 30 days return weekly scores.

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
