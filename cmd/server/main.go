package main

import (
	"log"
	"net"

	"ticket-score-service/internal/config"
	"ticket-score-service/internal/database"

	"google.golang.org/grpc"
)

func main() {
	cfg := config.New()

	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Printf("Connected to database: %s", cfg.DatabasePath)

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
