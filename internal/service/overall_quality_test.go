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
		name               string
		categories         []models.RatingCategory
		paginatedRatings   map[string][]models.Rating
		totalCount         int
		chunkSize          int
		maxGoroutines      int
		expectedScore      string
		expectedTotal      int
		expectedChunks     int
		expectedGoroutines int
		countErr           error
		paginationErr      error
		categoryErr        error
		expectError        bool
	}{
		{
			name: "successful overall quality calculation",
			categories: []models.RatingCategory{
				{ID: 1, Name: "Spelling", Weight: 10.0},
				{ID: 2, Name: "Grammar", Weight: 5.0},
			},
			paginatedRatings: map[string][]models.Rating{
				"5:0": {
					{ID: 1, RatingCategoryID: 1, Rating: 4},
					{ID: 2, RatingCategoryID: 1, Rating: 5},
					{ID: 3, RatingCategoryID: 2, Rating: 3},
					{ID: 4, RatingCategoryID: 2, Rating: 4},
					{ID: 5, RatingCategoryID: 1, Rating: 5},
				},
				"5:5": {
					{ID: 6, RatingCategoryID: 2, Rating: 4},
					{ID: 7, RatingCategoryID: 1, Rating: 5},
					{ID: 8, RatingCategoryID: 2, Rating: 3},
				},
			},
			totalCount:         8,
			chunkSize:          5,
			maxGoroutines:      3,
			expectedScore:      "88%", // Calculated based on weighted average
			expectedTotal:      5,     // Total ratings processed
			expectedChunks:     2,     // (8 + 5 - 1) / 5 = 2
			expectedGoroutines: 3,
			expectError:        false,
		},
		{
			name: "no ratings in period",
			categories: []models.RatingCategory{
				{ID: 1, Name: "Spelling", Weight: 10.0},
			},
			paginatedRatings:   map[string][]models.Rating{},
			totalCount:         0,
			chunkSize:          5,
			maxGoroutines:      3,
			expectedScore:      "N/A",
			expectedTotal:      0,
			expectedChunks:     0,
			expectedGoroutines: 0,
			expectError:        false,
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
			totalCount:         2,
			chunkSize:          10,
			maxGoroutines:      5,
			expectedScore:      "100%", // Perfect scores
			expectedTotal:      2,
			expectedChunks:     1,
			expectedGoroutines: 5,
			expectError:        false,
		},
		{
			name: "multiple chunks with different category weights",
			categories: []models.RatingCategory{
				{ID: 1, Name: "Spelling", Weight: 20.0},
				{ID: 2, Name: "Grammar", Weight: 10.0},
				{ID: 3, Name: "Clarity", Weight: 5.0},
			},
			paginatedRatings: map[string][]models.Rating{
				"3:0": {
					{ID: 1, RatingCategoryID: 1, Rating: 4}, // weight 20
					{ID: 2, RatingCategoryID: 2, Rating: 3}, // weight 10
					{ID: 3, RatingCategoryID: 3, Rating: 5}, // weight 5
				},
				"2:3": {
					{ID: 4, RatingCategoryID: 1, Rating: 5}, // weight 20
					{ID: 5, RatingCategoryID: 2, Rating: 4}, // weight 10
				},
			},
			totalCount:         5,
			chunkSize:          3,
			maxGoroutines:      2,
			expectedScore:      "85%", // Weighted calculation
			expectedTotal:      5,
			expectedChunks:     2,
			expectedGoroutines: 2,
			expectError:        false,
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
			chunkSize:        5,
			paginationErr:    errors.New("pagination query failed"),
			expectError:      true,
		},
		{
			name: "large dataset with multiple chunks",
			categories: []models.RatingCategory{
				{ID: 1, Name: "Spelling", Weight: 10.0},
			},
			paginatedRatings: map[string][]models.Rating{
				"100:0":   generateRatings(1, 100, 1, 4),
				"100:100": generateRatings(101, 100, 1, 5),
				"100:200": generateRatings(201, 50, 1, 3),
			},
			totalCount:         250,
			chunkSize:          100,
			maxGoroutines:      5,
			expectedScore:      "90%", // Mix of ratings 4, 5, 3
			expectedTotal:      200,
			expectedChunks:     3,
			expectedGoroutines: 5,
			expectError:        false,
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

			// Create service with custom configuration
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

			if result.TotalRatings != tt.expectedTotal {
				t.Errorf("Expected total ratings %d, got %d", tt.expectedTotal, result.TotalRatings)
			}

			if result.ChunksProcessed != tt.expectedChunks {
				t.Errorf("Expected chunks processed %d, got %d", tt.expectedChunks, result.ChunksProcessed)
			}

			if result.Goroutines != tt.expectedGoroutines {
				t.Errorf("Expected goroutines %d, got %d", tt.expectedGoroutines, result.Goroutines)
			}

			// All processing is done via paginated method

			// Verify period format
			expectedPeriod := "2019-10-01 to 2019-10-07"
			if result.Period != expectedPeriod {
				t.Errorf("Expected period %s, got %s", expectedPeriod, result.Period)
			}

			// Verify processing time is set
			if result.ProcessingTime == "" {
				t.Errorf("Expected processing time to be set")
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
		expectedTotal    int
		expectedChunks   int
		paginationErr    error
		expectError      bool
	}{
		{
			name:       "successful concurrent processing",
			totalCount: 6,
			paginatedRatings: map[string][]models.Rating{
				"3:0": {
					{ID: 1, RatingCategoryID: 1, Rating: 4},
					{ID: 2, RatingCategoryID: 2, Rating: 3},
					{ID: 3, RatingCategoryID: 1, Rating: 5},
				},
				"3:3": {
					{ID: 4, RatingCategoryID: 2, Rating: 4},
					{ID: 5, RatingCategoryID: 1, Rating: 5},
					{ID: 6, RatingCategoryID: 2, Rating: 5},
				},
			},
			expectedScore:  88.89, // Weighted calculation
			expectedTotal:  6,
			expectedChunks: 2,
			expectError:    false,
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
			expectedScore:  100.0,
			expectedTotal:  2,
			expectedChunks: 1,
			expectError:    false,
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
			expectedTotal:    0,
			expectedChunks:   0,
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

			score, totalRatings, chunksProcessed, err := service.processChunksConcurrently(
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

			// Allow for small floating point differences
			if abs(score-tt.expectedScore) > 0.1 {
				t.Errorf("Expected score %.2f, got %.2f", tt.expectedScore, score)
			}

			if totalRatings != tt.expectedTotal {
				t.Errorf("Expected total ratings %d, got %d", tt.expectedTotal, totalRatings)
			}

			if chunksProcessed != tt.expectedChunks {
				t.Errorf("Expected chunks processed %d, got %d", tt.expectedChunks, chunksProcessed)
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

			if abs(weightedSum-tt.expectedWeightedSum) > 0.01 {
				t.Errorf("Expected weighted sum %.2f, got %.2f", tt.expectedWeightedSum, weightedSum)
			}

			if abs(maxSum-tt.expectedMaxSum) > 0.01 {
				t.Errorf("Expected max sum %.2f, got %.2f", tt.expectedMaxSum, maxSum)
			}
		})
	}
}

func TestContextCancellation(t *testing.T) {
	categories := []models.RatingCategory{
		{ID: 1, Name: "Spelling", Weight: 10.0},
	}

	mockRatingsRepo := &mocks.MockRatingsRepo{
		Ratings: map[string][]models.Rating{
			"1:0": {{ID: 1, RatingCategoryID: 1, Rating: 4}},
		},
		Count: 1,
	}

	mockCategoryRepo := &mockCategoryRepo{
		categories: categories,
	}

	service := NewOverallQualityService(mockRatingsRepo, mockCategoryRepo)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	startDate := time.Date(2019, 10, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2019, 10, 7, 0, 0, 0, 0, time.UTC)

	// This should handle context cancellation gracefully
	_, _, _, err := service.processChunksConcurrently(ctx, startDate, endDate, 1, categories)

	if err == nil {
		t.Errorf("Expected error due to context cancellation, got none")
	}
}

// Helper functions

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

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
