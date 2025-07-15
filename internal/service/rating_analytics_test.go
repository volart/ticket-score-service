package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"ticket-score-service/internal/models"
	"ticket-score-service/internal/utils"
)

type mockCategoryRepo struct {
	categories []models.RatingCategory
	err        error
}

func (m *mockCategoryRepo) GetAll(ctx context.Context) ([]models.RatingCategory, error) {
	return m.categories, m.err
}

type mockRatingsRepo struct {
	ratingsByDate map[string][]models.Rating
	err           error
}

func (m *mockRatingsRepo) GetByCategoryIDAndDate(ctx context.Context, categoryID int, date time.Time) ([]models.Rating, error) {
	if m.err != nil {
		return nil, m.err
	}

	dateStr := date.Format("2006-01-02")
	key := fmt.Sprintf("%d-%s", categoryID, dateStr)

	if ratings, exists := m.ratingsByDate[key]; exists {
		return ratings, nil
	}

	return []models.Rating{}, nil
}

func (m *mockRatingsRepo) GetDistinctTicketIDsByDateRange(ctx context.Context, startDate, endDate time.Time) ([]int, error) {
	if m.err != nil {
		return nil, m.err
	}

	ticketIDMap := make(map[int]bool)
	for _, ratings := range m.ratingsByDate {
		for _, rating := range ratings {
			if rating.CreatedAt.After(startDate) && rating.CreatedAt.Before(endDate.Add(24*time.Hour)) {
				ticketIDMap[rating.TicketID] = true
			}
		}
	}

	var ticketIDs []int
	for id := range ticketIDMap {
		ticketIDs = append(ticketIDs, id)
	}

	return ticketIDs, nil
}

func (m *mockRatingsRepo) GetByTicketIDAndCategoryID(ctx context.Context, ticketID, categoryID int) ([]models.Rating, error) {
	if m.err != nil {
		return nil, m.err
	}

	var results []models.Rating
	for _, ratings := range m.ratingsByDate {
		for _, rating := range ratings {
			if rating.TicketID == ticketID && rating.RatingCategoryID == categoryID {
				results = append(results, rating)
			}
		}
	}

	return results, nil
}

func (m *mockRatingsRepo) GetByDateRangePaginated(ctx context.Context, startDate, endDate time.Time, limit, offset int) ([]models.Rating, error) {
	if m.err != nil {
		return nil, m.err
	}
	
	// For testing, collect all ratings within date range and apply pagination
	var allRatings []models.Rating
	for _, ratings := range m.ratingsByDate {
		for _, rating := range ratings {
			if rating.CreatedAt.After(startDate) && rating.CreatedAt.Before(endDate.Add(24*time.Hour)) {
				allRatings = append(allRatings, rating)
			}
		}
	}
	
	// Apply pagination
	if offset >= len(allRatings) {
		return []models.Rating{}, nil
	}
	
	end := offset + limit
	if end > len(allRatings) {
		end = len(allRatings)
	}
	
	return allRatings[offset:end], nil
}

func (m *mockRatingsRepo) CountByDateRange(ctx context.Context, startDate, endDate time.Time) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	
	count := 0
	for _, ratings := range m.ratingsByDate {
		for _, rating := range ratings {
			if rating.CreatedAt.After(startDate) && rating.CreatedAt.Before(endDate.Add(24*time.Hour)) {
				count++
			}
		}
	}
	
	return count, nil
}

type mockTicketScoreService struct {
	score float64
	err   error
}

func (m *mockTicketScoreService) CalculateScore(ratings []models.Rating, categories []models.RatingCategory) (float64, error) {
	return m.score, m.err
}

func TestGetCategoryAnalytics(t *testing.T) {
	tests := []struct {
		name          string
		categories    []models.RatingCategory
		ratings       map[string][]models.Rating
		startDate     time.Time
		endDate       time.Time
		expectedCount int
		expectError   bool
	}{
		{
			name: "successful analysis with ratings",
			categories: []models.RatingCategory{
				{ID: 1, Name: "Spelling", Weight: 10},
				{ID: 2, Name: "Grammar", Weight: 5},
			},
			ratings: map[string][]models.Rating{
				"1-2024-01-01": {{ID: 1, Rating: 4, RatingCategoryID: 1}},
				"2-2024-01-01": {{ID: 2, Rating: 5, RatingCategoryID: 2}},
			},
			startDate:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:          "no categories",
			categories:    []models.RatingCategory{},
			ratings:       map[string][]models.Rating{},
			startDate:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "multiple days with mixed data",
			categories: []models.RatingCategory{
				{ID: 1, Name: "Spelling", Weight: 10},
			},
			ratings: map[string][]models.Rating{
				"1-2024-01-01": {{ID: 1, Rating: 4, RatingCategoryID: 1}},
			},
			startDate:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:       time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
			expectedCount: 1,
			expectError:   false,
		},
		{
			name: "long date range - weekly aggregation",
			categories: []models.RatingCategory{
				{ID: 1, Name: "Spelling", Weight: 10},
			},
			ratings: map[string][]models.Rating{
				"1-2024-01-01": {{ID: 1, Rating: 4, RatingCategoryID: 1}},
				"1-2024-02-15": {{ID: 2, Rating: 5, RatingCategoryID: 1}},
			},
			startDate:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:       time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC),
			expectedCount: 1,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			categoryRepo := &mockCategoryRepo{categories: tt.categories}
			ratingsRepo := &mockRatingsRepo{ratingsByDate: tt.ratings}
			ticketScoreServ := &mockTicketScoreService{score: 80.0}

			service := NewRatingAnalyticsService(categoryRepo, ratingsRepo, ticketScoreServ)

			result, err := service.GetCategoryAnalytics(context.Background(), tt.startDate, tt.endDate)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(result) != tt.expectedCount {
				t.Errorf("expected %d categories, got %d", tt.expectedCount, len(result))
			}

			for _, analytics := range result {
				if analytics.Category == "" {
					t.Errorf("category name should not be empty")
				}
				if len(analytics.Dates) == 0 {
					t.Errorf("dates should not be empty")
				}

				// Check if long date ranges use weekly aggregation
				if tt.name == "long date range - weekly aggregation" {
					for _, date := range analytics.Dates {
						if !strings.Contains(date.Date, " to ") {
							t.Errorf("expected weekly format with 'to' separator, got %s", date.Date)
						}
					}
				}
			}
		})
	}
}

func TestCalculateScores(t *testing.T) {
	tests := []struct {
		name                string
		startDate           time.Time
		endDate             time.Time
		expectedAggregation string // "daily" or "weekly"
	}{
		{
			name:                "short range - daily aggregation",
			startDate:           time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:             time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
			expectedAggregation: "daily",
		},
		{
			name:                "long range - weekly aggregation",
			startDate:           time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:             time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC),
			expectedAggregation: "weekly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			categoryRepo := &mockCategoryRepo{}
			ratingsRepo := &mockRatingsRepo{ratingsByDate: map[string][]models.Rating{}}
			ticketScoreServ := &mockTicketScoreService{score: 75.0}
			service := NewRatingAnalyticsService(categoryRepo, ratingsRepo, ticketScoreServ)

			category := models.RatingCategory{ID: 1, Name: "Spelling", Weight: 10}
			scores, _, err := service.calculateScores(context.Background(), category, tt.startDate, tt.endDate)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(scores) == 0 {
				t.Errorf("expected scores to be returned")
			}

			// Check aggregation type based on date format
			if tt.expectedAggregation == "weekly" {
				for _, score := range scores {
					if !strings.Contains(score.Date, " to ") {
						t.Errorf("expected weekly format with 'to' separator, got %s", score.Date)
					}
				}
			} else {
				for _, score := range scores {
					if strings.Contains(score.Date, " to ") {
						t.Errorf("expected daily format without 'to' separator, got %s", score.Date)
					}
				}
			}
		})
	}
}

func TestCalculateDailyScore(t *testing.T) {
	ticketScoreServ := &mockTicketScoreService{score: 75.0}
	service := &RatingAnalyticsService{
		ticketScoreServ: ticketScoreServ,
	}

	category := models.RatingCategory{ID: 1, Name: "Spelling", Weight: 10}

	tests := []struct {
		name          string
		ratings       []models.Rating
		expectedScore string
	}{
		{
			name:          "no ratings",
			ratings:       []models.Rating{},
			expectedScore: "N/A",
		},
		{
			name: "with ratings",
			ratings: []models.Rating{
				{ID: 1, Rating: 4, RatingCategoryID: 1},
			},
			expectedScore: "75%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateDailyScore(tt.ratings, category, "2024-01-01")

			if result.Score != tt.expectedScore {
				t.Errorf("expected score %s, got %s", tt.expectedScore, result.Score)
			}
			if result.Date != "2024-01-01" {
				t.Errorf("expected date 2024-01-01, got %s", result.Date)
			}
		})
	}
}

func TestCalculateOverallScore(t *testing.T) {
	ticketScoreServ := &mockTicketScoreService{}
	service := &RatingAnalyticsService{
		ticketScoreServ: ticketScoreServ,
	}
	category := models.RatingCategory{ID: 1, Name: "Spelling", Weight: 10}

	tests := []struct {
		name          string
		ratings       []models.Rating
		mockScore     float64
		mockError     error
		expectedScore string
	}{
		{
			name:          "no ratings",
			ratings:       []models.Rating{},
			expectedScore: "N/A",
		},
		{
			name: "single rating",
			ratings: []models.Rating{
				{ID: 1, Rating: 4, RatingCategoryID: 1},
			},
			mockScore:     80.0,
			expectedScore: "80%",
		},
		{
			name: "multiple ratings",
			ratings: []models.Rating{
				{ID: 1, Rating: 4, RatingCategoryID: 1},
				{ID: 2, Rating: 5, RatingCategoryID: 1},
			},
			mockScore:     90.0,
			expectedScore: "90%",
		},
		{
			name: "calculation error",
			ratings: []models.Rating{
				{ID: 1, Rating: 4, RatingCategoryID: 1},
			},
			mockError:     fmt.Errorf("calculation error"),
			expectedScore: "N/A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set mock values for this test
			ticketScoreServ.score = tt.mockScore
			ticketScoreServ.err = tt.mockError
			
			result := service.calculateOverallScore(tt.ratings, category)

			if result != tt.expectedScore {
				t.Errorf("expected score %s, got %s", tt.expectedScore, result)
			}
		})
	}
}

func TestFormatScore(t *testing.T) {
	tests := []struct {
		score    float64
		expected string
	}{
		{80.0, "80%"},
		{75.5, "76%"},
		{0.0, "0%"},
		{100.0, "100%"},
	}

	for _, tt := range tests {
		result := utils.FormatScore(tt.score)
		if result != tt.expected {
			t.Errorf("FormatScore(%.1f) = %s, expected %s", tt.score, result, tt.expected)
		}
	}
}
