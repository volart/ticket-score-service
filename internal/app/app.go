package app

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"ticket-score-service/internal/config"
	"ticket-score-service/internal/database"
	"ticket-score-service/internal/repository"
	"ticket-score-service/internal/server"
	"ticket-score-service/internal/service"
	overallQualityPb "ticket-score-service/proto/generated/overall_quality"
	ratingPb "ticket-score-service/proto/generated/rating_analytics"
	ticketPb "ticket-score-service/proto/generated/ticket_scores"
)

// App represents the application with all its dependencies
type App struct {
	config   *config.Config
	db       *database.DB
	server   *grpc.Server
	listener net.Listener
}

// New creates a new application instance with all dependencies initialized
func New() (*App, error) {
	// Load configuration
	cfg := config.New()

	// Initialize database
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		return nil, err
	}

	// Initialize repositories
	categoryRepo := repository.NewRatingCategoryRepository(db.GetConnection())
	ratingsRepo := repository.NewRatingsRepository(db.GetConnection())

	// Initialize services
	ticketScoreService := service.NewTicketScoreService()
	analyticsService := service.NewRatingAnalyticsService(categoryRepo, ratingsRepo, ticketScoreService)
	ticketScoresService := service.NewTicketScoresService(categoryRepo, ratingsRepo, ticketScoreService)
	overallQualityService := service.NewOverallQualityService(ratingsRepo, categoryRepo)
	// periodComparisonService := service.NewPeriodComparisonService(overallQualityService)

	// Create gRPC server
	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)

	// Register services
	analyticsServer := server.NewRatingAnalyticsServer(analyticsService)
	ratingPb.RegisterRatingAnalyticsServiceServer(grpcServer, analyticsServer)

	ticketScoresServer := server.NewTicketScoresServer(ticketScoresService)
	ticketPb.RegisterTicketScoresServiceServer(grpcServer, ticketScoresServer)

	overallQualityServer := server.NewOverallQualityServer(overallQualityService)
	overallQualityPb.RegisterOverallQualityServiceServer(grpcServer, overallQualityServer)

	// Create listener
	listener, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &App{
		config:   cfg,
		db:       db,
		server:   grpcServer,
		listener: listener,
	}, nil
}

// Run starts the application
func (a *App) Run() error {
	log.Printf("Connected to database: %s", a.config.DatabasePath)
	log.Printf("Server listening on port %s", a.config.Port)

	return a.server.Serve(a.listener)
}

// Shutdown gracefully shuts down the application
func (a *App) Shutdown() {
	if a.server != nil {
		a.server.GracefulStop()
	}
	if a.listener != nil {
		a.listener.Close()
	}
	if a.db != nil {
		a.db.Close()
	}
}
