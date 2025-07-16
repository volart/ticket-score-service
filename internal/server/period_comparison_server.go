package server

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"ticket-score-service/internal/service"
	pb "ticket-score-service/proto/generated/period_comparison"
)

// PeriodComparisonServer implements the gRPC server for period comparison
type PeriodComparisonServer struct {
	pb.UnimplementedPeriodComparisonServiceServer
	periodComparisonService *service.PeriodComparisonService
}

// NewPeriodComparisonServer creates a new gRPC server instance
func NewPeriodComparisonServer(periodComparisonService *service.PeriodComparisonService) *PeriodComparisonServer {
	return &PeriodComparisonServer{
		periodComparisonService: periodComparisonService,
	}
}

// GetPeriodComparison handles the gRPC request for period comparison
func (s *PeriodComparisonServer) GetPeriodComparison(
	ctx context.Context,
	req *pb.GetPeriodComparisonRequest,
) (*pb.GetPeriodComparisonResponse, error) {
	// Validate request
	if req.StartingDate == "" {
		return nil, status.Error(codes.InvalidArgument, "starting_date is required")
	}

	// Parse starting date
	startingDate, err := time.Parse("2006-01-02", req.StartingDate)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid starting_date format: %v", err)
	}

	// Calculate both periods based on starting date and period type
	firstStart, firstEnd, secondStart, secondEnd, err := s.calculatePeriodDates(startingDate, req.PeriodType)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to calculate period dates: %v", err)
	}

	// Call service with first period and second period
	result, err := s.periodComparisonService.GetPeriodComparison(
		ctx,
		firstStart,
		firstEnd,
		secondStart,
		secondEnd,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get period comparison: %v", err)
	}

	// Build response
	response := &pb.GetPeriodComparisonResponse{
		StartPeriod: result.StartPeriod,
		StartScore:  result.StartScore,
		EndPeriod:   result.EndPeriod,
		EndScore:    result.EndScore,
		Difference:  result.Difference,
	}

	return response, nil
}

// calculatePeriodDates calculates both periods based on starting date and period type
func (s *PeriodComparisonServer) calculatePeriodDates(
	startingDate time.Time,
	periodType pb.PeriodType,
) (time.Time, time.Time, time.Time, time.Time, error) {
	var firstStart, firstEnd, secondStart, secondEnd time.Time

	switch periodType {
	case pb.PeriodType_WEEK:
		// First period: starting date to +6 days (7 days total)
		firstStart = startingDate
		firstEnd = startingDate.AddDate(0, 0, 6)
		// Second period: +7 days to +13 days (next 7 days)
		secondStart = startingDate.AddDate(0, 0, 7)
		secondEnd = startingDate.AddDate(0, 0, 13)

	case pb.PeriodType_MONTH:
		// First period: starting date to end of that month
		firstStart = startingDate
		firstEnd = time.Date(startingDate.Year(), startingDate.Month()+1, 1, 0, 0, 0, 0, startingDate.Location()).AddDate(0, 0, -1)
		// Second period: start of next month to end of next month
		secondStart = time.Date(startingDate.Year(), startingDate.Month()+1, 1, 0, 0, 0, 0, startingDate.Location())
		secondEnd = time.Date(startingDate.Year(), startingDate.Month()+2, 1, 0, 0, 0, 0, startingDate.Location()).AddDate(0, 0, -1)

	case pb.PeriodType_QUARTER:
		// First period: starting date to +3 months
		firstStart = startingDate
		firstEnd = startingDate.AddDate(0, 3, 0).AddDate(0, 0, -1)
		// Second period: +3 months to +6 months
		secondStart = startingDate.AddDate(0, 3, 0)
		secondEnd = startingDate.AddDate(0, 6, 0).AddDate(0, 0, -1)

	case pb.PeriodType_YEAR:
		// First period: starting date to +1 year
		firstStart = startingDate
		firstEnd = startingDate.AddDate(1, 0, 0).AddDate(0, 0, -1)
		// Second period: +1 year to +2 years
		secondStart = startingDate.AddDate(1, 0, 0)
		secondEnd = startingDate.AddDate(2, 0, 0).AddDate(0, 0, -1)

	default:
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("unsupported period type: %v", periodType)
	}

	return firstStart, firstEnd, secondStart, secondEnd, nil
}
