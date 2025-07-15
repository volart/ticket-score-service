package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ticket-score-service/internal/models"
)

// OverallQualityScore represents the aggregate quality score for a period
type OverallQualityScore struct {
	Period          string `json:"period"`
	Score           string `json:"score"`
	TotalRatings    int    `json:"totalRatings"`
	ProcessingTime  string `json:"processingTime"`
	ChunksProcessed int    `json:"chunksProcessed"`
	Goroutines      int    `json:"goroutines"`
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
	startTime := time.Now()

	// Get total count
	totalCount, err := s.ratingsRepo.CountByDateRange(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to count ratings: %w", err)
	}

	if totalCount == 0 {
		return &OverallQualityScore{
			Period:          formatDateRange(startDate, endDate),
			Score:           "N/A",
			TotalRatings:    0,
			ProcessingTime:  fmt.Sprintf("%dms", time.Since(startTime).Milliseconds()),
			ChunksProcessed: 0,
			Goroutines:      0,
		}, nil
	}

	// Get categories for weighting
	categories, err := s.categoryRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	// Process chunks concurrently
	score, processedRatings, chunksProcessed, err := s.processChunksConcurrently(ctx, startDate, endDate, totalCount, categories)
	if err != nil {
		return nil, fmt.Errorf("failed to process chunks: %w", err)
	}

	return &OverallQualityScore{
		Period:          formatDateRange(startDate, endDate),
		Score:           formatScorePercent(score),
		TotalRatings:    processedRatings,
		ProcessingTime:  fmt.Sprintf("%dms", time.Since(startTime).Milliseconds()),
		ChunksProcessed: chunksProcessed,
		Goroutines:      s.maxGoroutines,
	}, nil
}

// processChunksConcurrently processes rating chunks using goroutines
func (s *OverallQualityService) processChunksConcurrently(
	ctx context.Context,
	startDate, endDate time.Time,
	totalCount int,
	categories []models.RatingCategory,
) (float64, int, int, error) {

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
func (s *OverallQualityService) aggregateChunkResults(resultChan <-chan ChunkResult, expectedChunks int) (float64, int, int, error) {
	var (
		totalWeightedScore = 0.0
		totalMaxScore      = 0.0
		totalRatings       = 0
		chunksProcessed    = 0
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
		totalRatings += result.RatingCount
		chunksProcessed++
	}

	// Check if we have any errors
	if len(errors) > 0 {
		return 0, 0, 0, fmt.Errorf("chunk processing errors: %v", errors)
	}

	// Calculate final percentage
	var finalScore float64
	if totalMaxScore > 0 {
		finalScore = (totalWeightedScore / totalMaxScore) * 100
	}

	return finalScore, totalRatings, chunksProcessed, nil
}

// formatDateRange formats the date range for display
func formatDateRange(startDate, endDate time.Time) string {
	if startDate.Equal(endDate) {
		return startDate.Format("2006-01-02")
	}
	return fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
}

// formatScorePercent formats a float score as a percentage string
func formatScorePercent(score float64) string {
	if score == 0 {
		return "0%"
	}
	return fmt.Sprintf("%.0f%%", score)
}
