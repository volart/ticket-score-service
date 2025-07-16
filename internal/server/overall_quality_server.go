package server

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"ticket-score-service/internal/service"
	pb "ticket-score-service/proto/generated/overall_quality"
)

// OverallQualityServiceInterface defines the interface for the overall quality service
type OverallQualityServiceInterface interface {
	GetOverallQualityScore(ctx context.Context, startDate, endDate time.Time) (*service.OverallQualityScore, error)
}

// OverallQualityServer implements the gRPC OverallQualityService
type OverallQualityServer struct {
	pb.UnimplementedOverallQualityServiceServer
	serviceLayer OverallQualityServiceInterface
}

// NewOverallQualityServer creates a new gRPC server for overall quality operations
func NewOverallQualityServer(serviceLayer OverallQualityServiceInterface) *OverallQualityServer {
	return &OverallQualityServer{
		serviceLayer: serviceLayer,
	}
}

// GetOverallQualityScore handles gRPC requests for calculating overall quality scores
func (s *OverallQualityServer) GetOverallQualityScore(ctx context.Context, req *pb.GetOverallQualityScoreRequest) (*pb.GetOverallQualityScoreResponse, error) {
	// Validate request
	if req.StartDate == "" || req.EndDate == "" {
		return nil, status.Errorf(codes.InvalidArgument, "start_date and end_date are required")
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid start_date format, expected YYYY-MM-DD: %v", err)
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid end_date format, expected YYYY-MM-DD: %v", err)
	}

	// Validate date range
	if startDate.After(endDate) {
		return nil, status.Errorf(codes.InvalidArgument, "start_date must be before or equal to end_date")
	}

	// Call service layer
	result, err := s.serviceLayer.GetOverallQualityScore(ctx, startDate, endDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to calculate overall quality score: %v", err)
	}

	// Convert to proto response
	response := &pb.GetOverallQualityScoreResponse{
		Period: result.Period,
		Score:  result.Score,
	}

	return response, nil
}