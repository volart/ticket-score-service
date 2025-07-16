package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ticket-score-service/internal/models"
	"ticket-score-service/internal/utils"
)

// OverallQualityScore represents the aggregate quality score for a period
type OverallQualityScore struct {
	Period string `json:"period"`
	Score  string `json:"score"`
}

// ChunkResult represents the result of processing a single chunk
type ChunkResult struct {
	WeightedScore float64
	MaxScore      float64
	RatingCount   int
	ChunkID       int
	Error         error
}

// ChunkWork represents work to be done by a goroutine
type ChunkWork struct {
	ChunkID    int
	StartDate  time.Time
	EndDate    time.Time
	Limit      int
	Offset     int
	Categories []models.RatingCategory
}

// OverallQualityService handles overall quality score calculations using concurrent pagination
type OverallQualityService struct {
	ratingsRepo   RatingsRepository
	categoryRepo  CategoryRepository
	maxGoroutines int
	chunkSize     int
}

// NewOverallQualityService creates a new overall quality service instance
func NewOverallQualityService(
	ratingsRepo RatingsRepository,
	categoryRepo CategoryRepository,
) *OverallQualityService {
	return &OverallQualityService{
		ratingsRepo:   ratingsRepo,
		categoryRepo:  categoryRepo,
		maxGoroutines: 10,   // Default concurrency limit
		chunkSize:     1000, // Default chunk size
	}
}

// GetOverallQualityScore calculates overall quality score using concurrent pagination processing
func (s *OverallQualityService) GetOverallQualityScore(ctx context.Context, startDate, endDate time.Time) (*OverallQualityScore, error) {
	// Get total count
	totalCount, err := s.ratingsRepo.CountByDateRange(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to count ratings: %w", err)
	}

	if totalCount == 0 {
		return &OverallQualityScore{
			Period: utils.FormatDateRange(startDate, endDate),
			Score:  "N/A",
		}, nil
	}

	// Get categories for weighting
	categories, err := s.categoryRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	// Process chunks concurrently
	score, err := s.processChunksConcurrently(ctx, startDate, endDate, totalCount, categories)
	if err != nil {
		return nil, fmt.Errorf("failed to process chunks: %w", err)
	}

	return &OverallQualityScore{
		Period: utils.FormatDateRange(startDate, endDate),
		Score:  utils.FormatScore(score),
	}, nil
}

// processChunksConcurrently processes rating chunks using goroutines
func (s *OverallQualityService) processChunksConcurrently(
	ctx context.Context,
	startDate, endDate time.Time,
	totalCount int,
	categories []models.RatingCategory,
) (float64, error) {

	// Calculate number of chunks
	numChunks := (totalCount + s.chunkSize - 1) / s.chunkSize

	// Create channels for results
	resultChan := make(chan ChunkResult, numChunks)

	// Start worker goroutines with semaphore for concurrency control
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, s.maxGoroutines)

	// Process each chunk
	for i := 0; i < numChunks; i++ {
		offset := i * s.chunkSize
		limit := s.chunkSize
		if offset+limit > totalCount {
			limit = totalCount - offset
		}

		work := ChunkWork{
			ChunkID:    i,
			StartDate:  startDate,
			EndDate:    endDate,
			Limit:      limit,
			Offset:     offset,
			Categories: categories,
		}

		wg.Add(1)
		go s.processChunk(ctx, work, semaphore, resultChan, &wg)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Aggregate results
	return s.aggregateChunkResults(resultChan, numChunks)
}

// processChunk processes a single chunk of ratings
func (s *OverallQualityService) processChunk(
	ctx context.Context,
	work ChunkWork,
	semaphore chan struct{},
	resultChan chan<- ChunkResult,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	// Acquire semaphore
	select {
	case semaphore <- struct{}{}:
	case <-ctx.Done():
		resultChan <- ChunkResult{ChunkID: work.ChunkID, Error: ctx.Err()}
		return
	}
	defer func() { <-semaphore }()

	// Get ratings for this chunk
	ratings, err := s.ratingsRepo.GetByDateRangePaginated(ctx, work.StartDate, work.EndDate, work.Limit, work.Offset)
	if err != nil {
		resultChan <- ChunkResult{ChunkID: work.ChunkID, Error: err}
		return
	}

	// Calculate weighted score for this chunk
	weightedScore, maxScore := s.calculateChunkWeightedScore(ratings, work.Categories)

	resultChan <- ChunkResult{
		ChunkID:       work.ChunkID,
		WeightedScore: weightedScore,
		MaxScore:      maxScore,
		RatingCount:   len(ratings),
		Error:         nil,
	}
}

// calculateChunkWeightedScore calculates weighted score for a chunk of ratings
func (s *OverallQualityService) calculateChunkWeightedScore(ratings []models.Rating, categories []models.RatingCategory) (float64, float64) {
	// Create category weight map for quick lookup
	categoryWeights := make(map[int]float64)
	for _, cat := range categories {
		categoryWeights[cat.ID] = cat.Weight
	}

	var weightedSum, maxSum float64
	for _, rating := range ratings {
		weight := categoryWeights[rating.RatingCategoryID]
		maxRating := 5.0 // Assuming 1-5 scale

		weightedSum += float64(rating.Rating) * weight
		maxSum += maxRating * weight
	}

	return weightedSum, maxSum
}

// aggregateChunkResults combines results from all chunks
func (s *OverallQualityService) aggregateChunkResults(resultChan <-chan ChunkResult, expectedChunks int) (float64, error) {
	var (
		totalWeightedScore = 0.0
		totalMaxScore      = 0.0
		errors             []error
	)

	// Collect all results
	for result := range resultChan {
		if result.Error != nil {
			errors = append(errors, fmt.Errorf("chunk %d failed: %w", result.ChunkID, result.Error))
			continue
		}

		totalWeightedScore += result.WeightedScore
		totalMaxScore += result.MaxScore
	}

	// Check if we have any errors
	if len(errors) > 0 {
		return 0, fmt.Errorf("chunk processing errors: %v", errors)
	}

	// Calculate final percentage
	var finalScore float64
	if totalMaxScore > 0 {
		finalScore = (totalWeightedScore / totalMaxScore) * 100
	}

	return finalScore, nil
}
