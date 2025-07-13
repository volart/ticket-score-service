package main

import (
	"log"
	"net"

	"ticket-score-service/internal/config"

	"google.golang.org/grpc"
)

func main() {
	cfg := config.New()

	lis, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()

	log.Printf("Server listening on port %s", cfg.Port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
