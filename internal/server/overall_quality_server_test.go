package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"ticket-score-service/internal/service"
	pb "ticket-score-service/proto/generated/overall_quality"
)

// Mock service for testing
type mockOverallQualityService struct {
	result *service.OverallQualityScore
	err    error
}

func (m *mockOverallQualityService) GetOverallQualityScore(ctx context.Context, startDate, endDate time.Time) (*service.OverallQualityScore, error) {
	return m.result, m.err
}

func TestOverallQualityServer_GetOverallQualityScore(t *testing.T) {
	tests := []struct {
		name           string
		request        *pb.GetOverallQualityScoreRequest
		serviceResult  *service.OverallQualityScore
		serviceError   error
		expectedError  codes.Code
		expectedResult *pb.GetOverallQualityScoreResponse
	}{
		{
			name: "successful request",
			request: &pb.GetOverallQualityScoreRequest{
				StartDate: "2024-01-01",
				EndDate:   "2024-01-07",
			},
			serviceResult: &service.OverallQualityScore{
				Period: "2024-01-01 to 2024-01-07",
				Score:  "85%",
			},
			expectedResult: &pb.GetOverallQualityScoreResponse{
				Period: "2024-01-01 to 2024-01-07",
				Score:  "85%",
			},
		},
		{
			name: "missing start_date",
			request: &pb.GetOverallQualityScoreRequest{
				EndDate: "2024-01-07",
			},
			expectedError: codes.InvalidArgument,
		},
		{
			name: "missing end_date",
			request: &pb.GetOverallQualityScoreRequest{
				StartDate: "2024-01-01",
			},
			expectedError: codes.InvalidArgument,
		},
		{
			name: "invalid start_date format",
			request: &pb.GetOverallQualityScoreRequest{
				StartDate: "invalid-date",
				EndDate:   "2024-01-07",
			},
			expectedError: codes.InvalidArgument,
		},
		{
			name: "invalid end_date format",
			request: &pb.GetOverallQualityScoreRequest{
				StartDate: "2024-01-01",
				EndDate:   "invalid-date",
			},
			expectedError: codes.InvalidArgument,
		},
		{
			name: "start_date after end_date",
			request: &pb.GetOverallQualityScoreRequest{
				StartDate: "2024-01-07",
				EndDate:   "2024-01-01",
			},
			expectedError: codes.InvalidArgument,
		},
		{
			name: "service error",
			request: &pb.GetOverallQualityScoreRequest{
				StartDate: "2024-01-01",
				EndDate:   "2024-01-07",
			},
			serviceError:  errors.New("database connection failed"),
			expectedError: codes.Internal,
		},
		{
			name: "no ratings found",
			request: &pb.GetOverallQualityScoreRequest{
				StartDate: "2024-01-01",
				EndDate:   "2024-01-07",
			},
			serviceResult: &service.OverallQualityScore{
				Period: "2024-01-01 to 2024-01-07",
				Score:  "N/A",
			},
			expectedResult: &pb.GetOverallQualityScoreResponse{
				Period: "2024-01-01 to 2024-01-07",
				Score:  "N/A",
			},
		},
		{
			name: "same date range",
			request: &pb.GetOverallQualityScoreRequest{
				StartDate: "2024-01-01",
				EndDate:   "2024-01-01",
			},
			serviceResult: &service.OverallQualityScore{
				Period: "2024-01-01 to 2024-01-01",
				Score:  "92%",
			},
			expectedResult: &pb.GetOverallQualityScoreResponse{
				Period: "2024-01-01 to 2024-01-01",
				Score:  "92%",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock service
			mockService := &mockOverallQualityService{
				result: tt.serviceResult,
				err:    tt.serviceError,
			}

			// Create server
			server := NewOverallQualityServer(mockService)

			// Execute request
			ctx := context.Background()
			response, err := server.GetOverallQualityScore(ctx, tt.request)

			// Check for expected errors
			if tt.expectedError != codes.OK {
				if err == nil {
					t.Errorf("Expected error with code %v, but got no error", tt.expectedError)
					return
				}

				statusErr, ok := status.FromError(err)
				if !ok {
					t.Errorf("Expected gRPC status error, got %T: %v", err, err)
					return
				}

				if statusErr.Code() != tt.expectedError {
					t.Errorf("Expected error code %v, got %v", tt.expectedError, statusErr.Code())
				}
				return
			}

			// Check for unexpected errors
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify response
			if response == nil {
				t.Error("Response should not be nil")
				return
			}

			// Compare all fields
			if response.Period != tt.expectedResult.Period {
				t.Errorf("Expected period %s, got %s", tt.expectedResult.Period, response.Period)
			}
			if response.Score != tt.expectedResult.Score {
				t.Errorf("Expected score %s, got %s", tt.expectedResult.Score, response.Score)
			}
		})
	}
}

func TestOverallQualityServer_DateParsing(t *testing.T) {
	// Test various date formats to ensure proper validation
	invalidDates := []string{
		"2024-1-1",             // Single digit month/day
		"2024/01/01",           // Wrong separator
		"01-01-2024",           // Wrong order
		"2024-13-01",           // Invalid month
		"2024-01-32",           // Invalid day
		"not-a-date",           // Not a date
		"",                     // Empty string
		"2024-01-01T00:00:00Z", // ISO format with time
	}

	mockService := &mockOverallQualityService{
		result: &service.OverallQualityScore{
			Period: "2024-01-01 to 2024-01-07",
			Score:  "85%",
		},
	}

	server := NewOverallQualityServer(mockService)

	for _, invalidDate := range invalidDates {
		t.Run("invalid_start_date_"+invalidDate, func(t *testing.T) {
			ctx := context.Background()
			request := &pb.GetOverallQualityScoreRequest{
				StartDate: invalidDate,
				EndDate:   "2024-01-07",
			}

			_, err := server.GetOverallQualityScore(ctx, request)
			if err == nil {
				t.Errorf("Expected error for invalid start_date %s, but got none", invalidDate)
				return
			}

			statusErr, ok := status.FromError(err)
			if !ok {
				t.Errorf("Expected gRPC status error, got %T: %v", err, err)
				return
			}

			if statusErr.Code() != codes.InvalidArgument {
				t.Errorf("Expected InvalidArgument error, got %v", statusErr.Code())
			}
		})

		t.Run("invalid_end_date_"+invalidDate, func(t *testing.T) {
			ctx := context.Background()
			request := &pb.GetOverallQualityScoreRequest{
				StartDate: "2024-01-01",
				EndDate:   invalidDate,
			}

			_, err := server.GetOverallQualityScore(ctx, request)
			if err == nil {
				t.Errorf("Expected error for invalid end_date %s, but got none", invalidDate)
				return
			}

			statusErr, ok := status.FromError(err)
			if !ok {
				t.Errorf("Expected gRPC status error, got %T: %v", err, err)
				return
			}

			if statusErr.Code() != codes.InvalidArgument {
				t.Errorf("Expected InvalidArgument error, got %v", statusErr.Code())
			}
		})
	}
}
