package server

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"ticket-score-service/internal/service"
	pb "ticket-score-service/proto/generated/ticket_scores"
)

// TicketScoresServer implements the gRPC TicketScoresService
type TicketScoresServer struct {
	pb.UnimplementedTicketScoresServiceServer
	ticketScoresService *service.TicketScoresService
}

// NewTicketScoresServer creates a new gRPC server instance
func NewTicketScoresServer(ticketScoresService *service.TicketScoresService) *TicketScoresServer {
	return &TicketScoresServer{
		ticketScoresService: ticketScoresService,
	}
}

// GetTicketScores handles the gRPC streaming request for ticket scores  
func (s *TicketScoresServer) GetTicketScores(req *pb.GetTicketScoresRequest, stream grpc.ServerStreamingServer[pb.TicketScore]) error {
	// Validate request
	if req.StartDate == "" || req.EndDate == "" {
		return status.Error(codes.InvalidArgument, "start_date and end_date are required")
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid start_date format, expected YYYY-MM-DD: %v", err)
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid end_date format, expected YYYY-MM-DD: %v", err)
	}

	// Validate date range
	if startDate.After(endDate) {
		return status.Error(codes.InvalidArgument, "start_date must be before or equal to end_date")
	}

	// Get ticket scores stream
	ctx := stream.Context()
	ticketScores, errorChan := s.ticketScoresService.GetTicketScores(ctx, startDate, endDate)

	// Stream results
	for {
		select {
		case ticketScore, ok := <-ticketScores:
			if !ok {
				// Channel closed, all tickets processed
				return nil
			}

			// Convert to proto message
			protoTicketScore := &pb.TicketScore{
				TicketId:   int32(ticketScore.TicketID),
				Categories: make([]*pb.TicketCategoryScore, len(ticketScore.Categories)),
			}

			for i, category := range ticketScore.Categories {
				protoTicketScore.Categories[i] = &pb.TicketCategoryScore{
					CategoryName: category.CategoryName,
					Score:        category.Score,
				}
			}

			// Send to client
			if err := stream.Send(protoTicketScore); err != nil {
				return status.Errorf(codes.Internal, "failed to send ticket score: %v", err)
			}

		case err := <-errorChan:
			if err != nil {
				return status.Errorf(codes.Internal, "failed to calculate ticket scores: %v", err)
			}

		case <-ctx.Done():
			return status.Error(codes.Canceled, "request canceled")
		}
	}
}