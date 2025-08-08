package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"ticket-score-service/internal/mocks"
	"ticket-score-service/internal/models"
)

func TestGetOverallQualityScore(t *testing.T) {
	startDate := time.Date(2019, 10, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2019, 10, 7, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name             string
		categories       []models.RatingCategory
		paginatedRatings map[string][]models.Rating
		totalCount       int
		expectedScore    string
		countErr         error
		paginationErr    error
		categoryErr      error
		expectError      bool
	}{
		{
			name: "successful overall quality calculation",
			categories: []models.RatingCategory{
				{ID: 1, Name: "Spelling", Weight: 10.0},
				{ID: 2, Name: "Grammar", Weight: 5.0},
			},
			paginatedRatings: map[string][]models.Rating{
				"8:0": {
					{ID: 1, RatingCategoryID: 1, Rating: 4},
					{ID: 2, RatingCategoryID: 1, Rating: 5},
					{ID: 3, RatingCategoryID: 2, Rating: 3},
					{ID: 4, RatingCategoryID: 2, Rating: 4},
					{ID: 5, RatingCategoryID: 1, Rating: 5},
				},
			},
			totalCount:    8,
			expectedScore: "88%", // Calculated based on weighted average
			expectError:   false,
		},
		{
			name: "no ratings in period",
			categories: []models.RatingCategory{
				{ID: 1, Name: "Spelling", Weight: 10.0},
			},
			paginatedRatings: map[string][]models.Rating{},
			totalCount:       0,
			expectedScore:    "N/A",
			expectError:      false,
		},
		{
			name: "single chunk processing",
			categories: []models.RatingCategory{
				{ID: 1, Name: "Spelling", Weight: 10.0},
			},
			paginatedRatings: map[string][]models.Rating{
				"2:0": {
					{ID: 1, RatingCategoryID: 1, Rating: 5},
					{ID: 2, RatingCategoryID: 1, Rating: 5},
				},
			},
			totalCount:    2,
			expectedScore: "100%", // Perfect scores
			expectError:   false,
		},
		{
			name: "multiple chunks with different category weights",
			categories: []models.RatingCategory{
				{ID: 1, Name: "Spelling", Weight: 20.0},
				{ID: 2, Name: "Grammar", Weight: 10.0},
				{ID: 3, Name: "Clarity", Weight: 5.0},
			},
			paginatedRatings: map[string][]models.Rating{
				"5:0": {
					{ID: 1, RatingCategoryID: 1, Rating: 4}, // weight 20
					{ID: 2, RatingCategoryID: 2, Rating: 3}, // weight 10
					{ID: 3, RatingCategoryID: 3, Rating: 5}, // weight 5
					{ID: 4, RatingCategoryID: 1, Rating: 5}, // weight 20
					{ID: 5, RatingCategoryID: 2, Rating: 4}, // weight 10
				},
			},
			totalCount:    5,
			expectedScore: "85%", // Weighted calculation
			expectError:   false,
		},
		{
			name: "error counting ratings",
			categories: []models.RatingCategory{
				{ID: 1, Name: "Spelling", Weight: 10.0},
			},
			paginatedRatings: map[string][]models.Rating{},
			totalCount:       0,
			countErr:         errors.New("database connection failed"),
			expectError:      true,
		},
		{
			name: "error getting categories",
			categories: []models.RatingCategory{
				{ID: 1, Name: "Spelling", Weight: 10.0},
			},
			paginatedRatings: map[string][]models.Rating{},
			totalCount:       5,
			categoryErr:      errors.New("category fetch failed"),
			expectError:      true,
		},
		{
			name: "error getting paginated ratings",
			categories: []models.RatingCategory{
				{ID: 1, Name: "Spelling", Weight: 10.0},
			},
			paginatedRatings: map[string][]models.Rating{},
			totalCount:       5,
			paginationErr:    errors.New("pagination query failed"),
			expectError:      true,
		},
		{
			name: "large dataset with multiple chunks",
			categories: []models.RatingCategory{
				{ID: 1, Name: "Spelling", Weight: 10.0},
			},
			paginatedRatings: map[string][]models.Rating{
				"250:0": append(append(generateRatings(1, 100, 1, 4), generateRatings(101, 100, 1, 5)...), generateRatings(201, 50, 1, 3)...),
			},
			totalCount:    250,
			expectedScore: "84%", // Mix of ratings 4, 5, 3: (100*4 + 100*5 + 50*3) / (250*5) = 1050/1250 = 0.84 = 84%
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockRatingsRepo := &mocks.MockRatingsRepo{
				Ratings:       tt.paginatedRatings,
				Count:         tt.totalCount,
				PaginationErr: tt.paginationErr,
				CountErr:      tt.countErr,
			}

			mockCategoryRepo := &mockCategoryRepo{
				categories: tt.categories,
				err:        tt.categoryErr,
			}

			// Create service
			service := NewOverallQualityService(mockRatingsRepo, mockCategoryRepo)

			// Execute
			ctx := context.Background()
			result, err := service.GetOverallQualityScore(ctx, startDate, endDate)

			// Verify error handling
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify results
			if result.Score != tt.expectedScore {
				t.Errorf("Expected score %s, got %s", tt.expectedScore, result.Score)
			}

			// All processing is done via paginated method

			// Verify period format
			expectedPeriod := "2019-10-01 to 2019-10-07"
			if result.Period != expectedPeriod {
				t.Errorf("Expected period %s, got %s", expectedPeriod, result.Period)
			}
		})
	}
}

func TestProcessChunksConcurrently(t *testing.T) {
	categories := []models.RatingCategory{
		{ID: 1, Name: "Spelling", Weight: 10.0},
		{ID: 2, Name: "Grammar", Weight: 5.0},
	}

	tests := []struct {
		name             string
		totalCount       int
		paginatedRatings map[string][]models.Rating
		expectedScore    float64
		paginationErr    error
		expectError      bool
	}{
		{
			name:       "successful concurrent processing",
			totalCount: 6,
			paginatedRatings: map[string][]models.Rating{
				"6:0": {
					{ID: 1, RatingCategoryID: 1, Rating: 4},
					{ID: 2, RatingCategoryID: 2, Rating: 3},
					{ID: 3, RatingCategoryID: 1, Rating: 5},
					{ID: 4, RatingCategoryID: 2, Rating: 4},
					{ID: 5, RatingCategoryID: 1, Rating: 5},
					{ID: 6, RatingCategoryID: 2, Rating: 5},
				},
			},
			expectedScore: 200.0 / 225.0 * 100, // Weighted calculation: (140+60)/(150+75)*100 = 88.888...%
			expectError:   false,
		},
		{
			name:       "single chunk processing",
			totalCount: 2,
			paginatedRatings: map[string][]models.Rating{
				"2:0": {
					{ID: 1, RatingCategoryID: 1, Rating: 5},
					{ID: 2, RatingCategoryID: 1, Rating: 5},
				},
			},
			expectedScore: 100.0,
			expectError:   false,
		},
		{
			name:       "error in chunk processing",
			totalCount: 3,
			paginatedRatings: map[string][]models.Rating{
				"2:0": {
					{ID: 1, RatingCategoryID: 1, Rating: 4},
				},
			},
			paginationErr: errors.New("chunk processing failed"),
			expectError:   true,
		},
		{
			name:             "empty chunks",
			totalCount:       0,
			paginatedRatings: map[string][]models.Rating{},
			expectedScore:    0.0,
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRatingsRepo := &mocks.MockRatingsRepo{
				Ratings:       tt.paginatedRatings,
				Count:         tt.totalCount,
				PaginationErr: tt.paginationErr,
			}

			mockCategoryRepo := &mockCategoryRepo{
				categories: categories,
			}

			service := NewOverallQualityService(mockRatingsRepo, mockCategoryRepo)

			ctx := context.Background()
			startDate := time.Date(2019, 10, 1, 0, 0, 0, 0, time.UTC)
			endDate := time.Date(2019, 10, 7, 0, 0, 0, 0, time.UTC)

			score, err := service.processChunksConcurrently(
				ctx, startDate, endDate, tt.totalCount, categories)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Allow for small floating point differences due to division
			if score != tt.expectedScore {
				t.Errorf("Expected score %.6f, got %.6f", tt.expectedScore, score)
			}
		})
	}
}

func TestCalculateChunkWeightedScore(t *testing.T) {
	categories := []models.RatingCategory{
		{ID: 1, Name: "Spelling", Weight: 10.0},
		{ID: 2, Name: "Grammar", Weight: 5.0},
	}

	tests := []struct {
		name                string
		ratings             []models.Rating
		expectedWeightedSum float64
		expectedMaxSum      float64
	}{
		{
			name: "mixed ratings with different weights",
			ratings: []models.Rating{
				{ID: 1, RatingCategoryID: 1, Rating: 4}, // 4 * 10 = 40
				{ID: 2, RatingCategoryID: 2, Rating: 3}, // 3 * 5 = 15
				{ID: 3, RatingCategoryID: 1, Rating: 5}, // 5 * 10 = 50
			},
			expectedWeightedSum: 105.0, // 40 + 15 + 50
			expectedMaxSum:      125.0, // (5*10) + (5*5) + (5*10)
		},
		{
			name: "single category ratings",
			ratings: []models.Rating{
				{ID: 1, RatingCategoryID: 1, Rating: 5},
				{ID: 2, RatingCategoryID: 1, Rating: 4},
			},
			expectedWeightedSum: 90.0,  // (5*10) + (4*10)
			expectedMaxSum:      100.0, // (5*10) + (5*10)
		},
		{
			name:                "empty ratings",
			ratings:             []models.Rating{},
			expectedWeightedSum: 0.0,
			expectedMaxSum:      0.0,
		},
		{
			name: "unknown category (zero weight)",
			ratings: []models.Rating{
				{ID: 1, RatingCategoryID: 999, Rating: 5}, // Unknown category
			},
			expectedWeightedSum: 0.0, // 5 * 0 = 0
			expectedMaxSum:      0.0, // 5 * 0 = 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRatingsRepo := &mocks.MockRatingsRepo{}
			mockCategoryRepo := &mockCategoryRepo{categories: categories}

			service := NewOverallQualityService(mockRatingsRepo, mockCategoryRepo)

			weightedSum, maxSum := service.calculateChunkWeightedScore(tt.ratings, categories)

			if weightedSum != tt.expectedWeightedSum {
				t.Errorf("Expected weighted sum %.2f, got %.2f", tt.expectedWeightedSum, weightedSum)
			}

			if maxSum != tt.expectedMaxSum {
				t.Errorf("Expected max sum %.2f, got %.2f", tt.expectedMaxSum, maxSum)
			}
		})
	}
}

// generateRatings creates a slice of test ratings
func generateRatings(startID, count, categoryID, rating int) []models.Rating {
	ratings := make([]models.Rating, count)
	for i := 0; i < count; i++ {
		ratings[i] = models.Rating{
			ID:               startID + i,
			RatingCategoryID: categoryID,
			Rating:           rating,
		}
	}
	return ratings
}
