package main

import (
	"log"

	"ticket-score-service/internal/app"
)

func main() {
	application, err := app.New()
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}
	defer application.Shutdown()

	if err := application.Run(); err != nil {
		log.Fatalf("Failed to run application: %v", err)
	}
}
