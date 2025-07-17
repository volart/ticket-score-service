FROM golang:1.24.5-alpine

WORKDIR /app

# Install required tools
RUN apk add --no-cache ca-certificates gcc musl-dev protoc sqlite-dev

# Install protobuf tools
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy proto files and generate them
COPY proto/ ./proto/
RUN mkdir -p proto/generated/rating_analytics proto/generated/ticket_scores proto/generated/overall_quality proto/generated/period_comparison
RUN protoc --go_out=. --go-grpc_out=. proto/*.proto

# Copy source code and build
COPY . .
RUN CGO_ENABLED=1 go build -o server cmd/server/main.go

EXPOSE 50051

CMD ["./server"]
