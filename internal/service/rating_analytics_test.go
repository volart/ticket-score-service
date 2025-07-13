package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ticket-score-service/internal/models"
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
				{ID: 1, Name: "Quality", Weight: 10},
				{ID: 2, Name: "Speed", Weight: 5},
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
				{ID: 1, Name: "Quality", Weight: 10},
			},
			ratings: map[string][]models.Rating{
				"1-2024-01-01": {{ID: 1, Rating: 4, RatingCategoryID: 1}},
			},
			startDate:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:       time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
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
			}
		})
	}
}

func TestCalculateDailyScore(t *testing.T) {
	ticketScoreServ := &mockTicketScoreService{score: 75.0}
	service := &RatingAnalyticsService{
		ticketScoreServ: ticketScoreServ,
	}

	category := models.RatingCategory{ID: 1, Name: "Quality", Weight: 10}

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
	service := &RatingAnalyticsService{}
	category := models.RatingCategory{ID: 1, Name: "Quality", Weight: 10}

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
			name: "single rating",
			ratings: []models.Rating{
				{ID: 1, Rating: 4, RatingCategoryID: 1},
			},
			expectedScore: "80%",
		},
		{
			name: "multiple ratings",
			ratings: []models.Rating{
				{ID: 1, Rating: 4, RatingCategoryID: 1},
				{ID: 2, Rating: 5, RatingCategoryID: 1},
			},
			expectedScore: "90%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		result := formatScore(tt.score)
		if result != tt.expected {
			t.Errorf("formatScore(%.1f) = %s, expected %s", tt.score, result, tt.expected)
		}
	}
}
