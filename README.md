# Ticket Score Service

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
```

**Note:** Generated files in `proto/generated/` are ignored by git and should be regenerated locally.
