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
