package server

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"ticket-score-service/internal/service"
	pb "ticket-score-service/proto/generated/rating_analytics"
)

// RatingAnalyticsServer implements the gRPC RatingAnalyticsService
type RatingAnalyticsServer struct {
	pb.UnimplementedRatingAnalyticsServiceServer
	analyticsService *service.RatingAnalyticsService
}

// NewRatingAnalyticsServer creates a new gRPC server instance
func NewRatingAnalyticsServer(analyticsService *service.RatingAnalyticsService) *RatingAnalyticsServer {
	return &RatingAnalyticsServer{
		analyticsService: analyticsService,
	}
}

// GetCategoryAnalytics handles the gRPC request for category analytics
func (s *RatingAnalyticsServer) GetCategoryAnalytics(ctx context.Context, req *pb.GetCategoryAnalyticsRequest) (*pb.GetCategoryAnalyticsResponse, error) {
	// Validate request
	if req.StartDate == "" || req.EndDate == "" {
		return nil, status.Error(codes.InvalidArgument, "start_date and end_date are required")
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
		return nil, status.Error(codes.InvalidArgument, "start_date must be before or equal to end_date")
	}

	// Call service layer
	analytics, err := s.analyticsService.GetCategoryAnalytics(ctx, startDate, endDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get category analytics: %v", err)
	}

	// Convert to proto response
	response := &pb.GetCategoryAnalyticsResponse{
		Analytics: make([]*pb.CategoryAnalytics, len(analytics)),
	}

	for i, analyticsItem := range analytics {
		response.Analytics[i] = &pb.CategoryAnalytics{
			Category: analyticsItem.Category,
			Ratings:  int32(analyticsItem.Ratings),
			Score:    analyticsItem.Score,
			Dates:    convertDailyScores(analyticsItem.Dates),
		}
	}

	return response, nil
}

// convertDailyScores converts service layer DailyScore to proto DailyScore
func convertDailyScores(dailyScores []service.DailyScore) []*pb.DailyScore {
	protoScores := make([]*pb.DailyScore, len(dailyScores))
	for i, score := range dailyScores {
		protoScores[i] = &pb.DailyScore{
			Date:  score.Date,
			Score: score.Score,
		}
	}
	return protoScores
}