# Build stage
FROM golang:1.24.5-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Tidy dependencies and build
RUN go mod tidy && CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Runtime stage
FROM alpine:latest

WORKDIR /root/

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy database file
COPY --from=builder /app/database.db .

# Expose port
EXPOSE 50051

# Set environment variable
ENV PORT=50051

# Run the binary
CMD ["./main"]
