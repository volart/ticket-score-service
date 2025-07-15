package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"ticket-score-service/internal/models"
)

// Additional mock for ScoreCalculator interface
type mockScoreCalculator struct {
	calculateFunc func([]models.Rating, []models.RatingCategory) (float64, error)
}

func (m *mockScoreCalculator) CalculateScore(ratings []models.Rating, categories []models.RatingCategory) (float64, error) {
	if m.calculateFunc != nil {
		return m.calculateFunc(ratings, categories)
	}
	return 0, nil
}

func TestGetTicketScores(t *testing.T) {
	startDate := time.Date(2019, 10, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2019, 10, 3, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name                string
		categories          []models.RatingCategory
		ratingsData         map[string][]models.Rating
		mockScoreCalculator func([]models.Rating, []models.RatingCategory) (float64, error)
		categoryRepoErr     error
		ratingsRepoErr      error
		expectedTicketCount int
		expectedError       bool
	}{
		{
			name: "successful ticket scores calculation",
			categories: []models.RatingCategory{
				{ID: 1, Name: "Spelling", Weight: 10},
				{ID: 2, Name: "Grammar", Weight: 5},
			},
			ratingsData: map[string][]models.Rating{
				"1-2019-10-01": {
					{ID: 1, TicketID: 1, RatingCategoryID: 1, Rating: 4, CreatedAt: startDate.Add(1 * time.Hour)},
					{ID: 2, TicketID: 1, RatingCategoryID: 1, Rating: 5, CreatedAt: startDate.Add(1 * time.Hour)},
				},
				"2-2019-10-01": {
					{ID: 3, TicketID: 1, RatingCategoryID: 2, Rating: 3, CreatedAt: startDate.Add(1 * time.Hour)},
				},
				"1-2019-10-02": {
					{ID: 4, TicketID: 2, RatingCategoryID: 1, Rating: 5, CreatedAt: startDate.Add(25 * time.Hour)},
				},
				"2-2019-10-02": {
					{ID: 5, TicketID: 2, RatingCategoryID: 2, Rating: 4, CreatedAt: startDate.Add(25 * time.Hour)},
				},
			},
			mockScoreCalculator: func(ratings []models.Rating, categories []models.RatingCategory) (float64, error) {
				if len(ratings) == 0 {
					return 0, nil
				}
				// Simple average for testing
				sum := 0.0
				for _, rating := range ratings {
					sum += float64(rating.Rating)
				}
				return (sum / float64(len(ratings))) * 20, nil // Convert to percentage
			},
			expectedTicketCount: 2,
			expectedError:       false,
		},
		{
			name: "no tickets found",
			categories: []models.RatingCategory{
				{ID: 1, Name: "Spelling", Weight: 10},
			},
			ratingsData: map[string][]models.Rating{},
			mockScoreCalculator: func(ratings []models.Rating, categories []models.RatingCategory) (float64, error) {
				return 0, nil
			},
			expectedTicketCount: 0,
			expectedError:       false,
		},
		{
			name: "error getting categories",
			categories: []models.RatingCategory{
				{ID: 1, Name: "Spelling", Weight: 10},
			},
			ratingsData:     map[string][]models.Rating{},
			categoryRepoErr: errors.New("category fetch error"),
			expectedError:   true,
		},
		{
			name: "score calculation error",
			categories: []models.RatingCategory{
				{ID: 1, Name: "Spelling", Weight: 10},
			},
			ratingsData: map[string][]models.Rating{
				"1-2019-10-01": {
					{ID: 1, TicketID: 1, RatingCategoryID: 1, Rating: 4, CreatedAt: startDate.Add(1 * time.Hour)},
				},
				"2-2019-10-01": {
					{ID: 2, TicketID: 1, RatingCategoryID: 2, Rating: 5, CreatedAt: startDate.Add(1 * time.Hour)},
				},
			},
			mockScoreCalculator: func(ratings []models.Rating, categories []models.RatingCategory) (float64, error) {
				return 0, errors.New("calculation error")
			},
			expectedTicketCount: 1,
			expectedError:       false, // Should still return ticket with N/A score
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockCategoryRepo := &mockCategoryRepo{
				categories: tt.categories,
				err:        tt.categoryRepoErr,
			}
			mockRatingsRepo := &mockRatingsRepo{
				ratingsByDate: tt.ratingsData,
				err:           tt.ratingsRepoErr,
			}
			mockScoreCalc := &mockScoreCalculator{
				calculateFunc: tt.mockScoreCalculator,
			}

			// Create service
			service := NewTicketScoresService(mockCategoryRepo, mockRatingsRepo, mockScoreCalc)

			// Execute
			ctx := context.Background()
			resultChan, errorChan := service.GetTicketScores(ctx, startDate, endDate)

			// Collect results
			var tickets []TicketScore
			var receivedError error

			for {
				select {
				case ticket, ok := <-resultChan:
					if !ok {
						resultChan = nil
						break
					}
					tickets = append(tickets, ticket)
				case err, ok := <-errorChan:
					if !ok {
						errorChan = nil
						break
					}
					if err != nil {
						receivedError = err
					}
				}
				if resultChan == nil && errorChan == nil {
					break
				}
			}

			// Verify results
			if tt.expectedError {
				if receivedError == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if receivedError != nil {
					t.Errorf("Unexpected error: %v", receivedError)
				}
			}

			if len(tickets) != tt.expectedTicketCount {
				t.Errorf("Expected %d tickets, got %d", tt.expectedTicketCount, len(tickets))
			}

			// Verify ticket structure
			for _, ticket := range tickets {
				if ticket.TicketID <= 0 {
					t.Errorf("Invalid ticket ID: %d", ticket.TicketID)
				}
				if len(ticket.Categories) != len(tt.categories) {
					t.Errorf("Expected %d categories for ticket %d, got %d",
						len(tt.categories), ticket.TicketID, len(ticket.Categories))
				}
				for _, category := range ticket.Categories {
					if category.CategoryName == "" {
						t.Errorf("Empty category name for ticket %d", ticket.TicketID)
					}
					if category.Score == "" {
						t.Errorf("Empty score for ticket %d, category %s", ticket.TicketID, category.CategoryName)
					}
				}
			}
		})
	}
}

func TestCalculateTicketScore(t *testing.T) {
	categories := []models.RatingCategory{
		{ID: 1, Name: "Spelling", Weight: 10},
		{ID: 2, Name: "Grammar", Weight: 5},
	}

	tests := []struct {
		name                string
		ticketID            int
		ratingsData         map[string][]models.Rating
		mockScoreCalculator func([]models.Rating, []models.RatingCategory) (float64, error)
		ratingsRepoErr      error
		expectedCategories  int
		expectedError       bool
	}{
		{
			name:     "successful calculation with ratings",
			ticketID: 1,
			ratingsData: map[string][]models.Rating{
				"1-2019-10-01": {
					{ID: 1, TicketID: 1, RatingCategoryID: 1, Rating: 4},
				},
				"2-2019-10-01": {
					{ID: 2, TicketID: 1, RatingCategoryID: 2, Rating: 5},
				},
			},
			mockScoreCalculator: func(ratings []models.Rating, categories []models.RatingCategory) (float64, error) {
				return 80.0, nil
			},
			expectedCategories: 2,
			expectedError:      false,
		},
		{
			name:     "no ratings for some categories",
			ticketID: 1,
			ratingsData: map[string][]models.Rating{
				"1-2019-10-01": {
					{ID: 1, TicketID: 1, RatingCategoryID: 1, Rating: 4},
				},
				// No ratings for category 2
			},
			mockScoreCalculator: func(ratings []models.Rating, categories []models.RatingCategory) (float64, error) {
				return 80.0, nil
			},
			expectedCategories: 2,
			expectedError:      false,
		},
		{
			name:     "error getting ratings",
			ticketID: 1,
			ratingsData: map[string][]models.Rating{
				"1-2019-10-01": {
					{ID: 1, TicketID: 1, RatingCategoryID: 1, Rating: 4},
				},
			},
			mockScoreCalculator: func(ratings []models.Rating, categories []models.RatingCategory) (float64, error) {
				return 80.0, nil
			},
			ratingsRepoErr: errors.New("database error"),
			expectedError:  true,
		},
		{
			name:     "score calculation error",
			ticketID: 1,
			ratingsData: map[string][]models.Rating{
				"1-2019-10-01": {
					{ID: 1, TicketID: 1, RatingCategoryID: 1, Rating: 4},
				},
				"2-2019-10-01": {
					{ID: 2, TicketID: 1, RatingCategoryID: 2, Rating: 5},
				},
			},
			mockScoreCalculator: func(ratings []models.Rating, categories []models.RatingCategory) (float64, error) {
				return 0, errors.New("calculation error")
			},
			expectedCategories: 2,
			expectedError:      false, // Should return N/A scores, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockCategoryRepo := &mockCategoryRepo{
				categories: categories,
			}
			mockRatingsRepo := &mockRatingsRepo{
				ratingsByDate: tt.ratingsData,
				err:           tt.ratingsRepoErr,
			}
			mockScoreCalc := &mockScoreCalculator{
				calculateFunc: tt.mockScoreCalculator,
			}

			// Create service
			service := NewTicketScoresService(mockCategoryRepo, mockRatingsRepo, mockScoreCalc)

			// Execute
			ctx := context.Background()
			ticketScore, err := service.calculateTicketScore(ctx, tt.ticketID, categories)

			// Verify results
			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if !tt.expectedError {
				if ticketScore.TicketID != tt.ticketID {
					t.Errorf("Expected ticket ID %d, got %d", tt.ticketID, ticketScore.TicketID)
				}
				if len(ticketScore.Categories) != tt.expectedCategories {
					t.Errorf("Expected %d categories, got %d", tt.expectedCategories, len(ticketScore.Categories))
				}
				for _, category := range ticketScore.Categories {
					if category.CategoryName == "" {
						t.Errorf("Empty category name")
					}
					if category.Score == "" {
						t.Errorf("Empty score for category %s", category.CategoryName)
					}
				}
			}
		})
	}
}

func TestTicketScoresService_ConcurrentProcessing(t *testing.T) {
	// Test with multiple tickets to verify concurrent processing works
	startDate := time.Date(2019, 10, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2019, 10, 3, 0, 0, 0, 0, time.UTC)

	categories := []models.RatingCategory{
		{ID: 1, Name: "Spelling", Weight: 10},
		{ID: 2, Name: "Grammar", Weight: 5},
	}

	// Create test data with multiple tickets
	ratingsData := make(map[string][]models.Rating)
	for i := 1; i <= 20; i++ {
		ratingsData[fmt.Sprintf("1-2019-10-01")] = append(ratingsData[fmt.Sprintf("1-2019-10-01")],
			models.Rating{ID: i, TicketID: i, RatingCategoryID: 1, Rating: 4, CreatedAt: startDate.Add(1 * time.Hour)})
		ratingsData[fmt.Sprintf("2-2019-10-01")] = append(ratingsData[fmt.Sprintf("2-2019-10-01")],
			models.Rating{ID: i + 20, TicketID: i, RatingCategoryID: 2, Rating: 5, CreatedAt: startDate.Add(1 * time.Hour)})
	}

	mockCategoryRepo := &mockCategoryRepo{
		categories: categories,
	}
	mockRatingsRepo := &mockRatingsRepo{
		ratingsByDate: ratingsData,
	}
	mockScoreCalc := &mockScoreCalculator{
		calculateFunc: func(ratings []models.Rating, categories []models.RatingCategory) (float64, error) {
			return 80.0, nil
		},
	}

	service := NewTicketScoresService(mockCategoryRepo, mockRatingsRepo, mockScoreCalc)

	ctx := context.Background()
	resultChan, errorChan := service.GetTicketScores(ctx, startDate, endDate)

	// Collect results
	var tickets []TicketScore
	var receivedError error

	for {
		select {
		case ticket, ok := <-resultChan:
			if !ok {
				resultChan = nil
				break
			}
			tickets = append(tickets, ticket)
		case err, ok := <-errorChan:
			if !ok {
				errorChan = nil
				break
			}
			if err != nil {
				receivedError = err
			}
		}
		if resultChan == nil && errorChan == nil {
			break
		}
	}

	// Verify results
	if receivedError != nil {
		t.Errorf("Unexpected error: %v", receivedError)
	}

	expectedTicketCount := 20
	if len(tickets) != expectedTicketCount {
		t.Errorf("Expected %d tickets, got %d", expectedTicketCount, len(tickets))
	}

	// Verify all tickets have correct structure
	for _, ticket := range tickets {
		if ticket.TicketID <= 0 || ticket.TicketID > 20 {
			t.Errorf("Invalid ticket ID: %d", ticket.TicketID)
		}
		if len(ticket.Categories) != 2 {
			t.Errorf("Expected 2 categories for ticket %d, got %d", ticket.TicketID, len(ticket.Categories))
		}
	}
}
