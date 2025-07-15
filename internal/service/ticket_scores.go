package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ticket-score-service/internal/models"
)

// TicketCategoryScore represents a score for a specific category within a ticket
type TicketCategoryScore struct {
	CategoryName string `json:"categoryName"`
	Score        string `json:"score"`
}

// TicketScore represents all category scores for a single ticket
type TicketScore struct {
	TicketID   int                   `json:"ticketId"`
	Categories []TicketCategoryScore `json:"categories"`
}

// TicketScoresService handles ticket score calculations
type TicketScoresService struct {
	categoryRepo    CategoryRepository
	ratingsRepo     RatingsRepository
	ticketScoreServ ScoreCalculator
}

// NewTicketScoresService creates a new ticket scores service instance
func NewTicketScoresService(
	categoryRepo CategoryRepository,
	ratingsRepo RatingsRepository,
	ticketScoreServ ScoreCalculator,
) *TicketScoresService {
	return &TicketScoresService{
		categoryRepo:    categoryRepo,
		ratingsRepo:     ratingsRepo,
		ticketScoreServ: ticketScoreServ,
	}
}

// GetTicketScores gets scores for all tickets within a date range, streaming results
func (s *TicketScoresService) GetTicketScores(ctx context.Context, startDate, endDate time.Time) (<-chan TicketScore, <-chan error) {
	resultChan := make(chan TicketScore, 100)
	errorChan := make(chan error, 1)

	go func() {
		defer close(resultChan)
		defer close(errorChan)

		// Get distinct ticket IDs from ratings table
		ticketIDs, err := s.ratingsRepo.GetDistinctTicketIDsByDateRange(ctx, startDate, endDate)
		if err != nil {
			errorChan <- fmt.Errorf("failed to get ticket IDs: %w", err)
			return
		}

		// Get all categories
		categories, err := s.categoryRepo.GetAll(ctx)
		if err != nil {
			errorChan <- fmt.Errorf("failed to get categories: %w", err)
			return
		}

		// Process tickets concurrently
		semaphore := make(chan struct{}, 10) // Limit concurrent goroutines
		var wg sync.WaitGroup

		for _, ticketID := range ticketIDs {
			wg.Add(1)
			go func(tID int) {
				defer wg.Done()
				semaphore <- struct{}{}        // Acquire
				defer func() { <-semaphore }() // Release

				ticketScore, err := s.calculateTicketScore(ctx, tID, categories)
				if err != nil {
					select {
					case errorChan <- fmt.Errorf("failed to calculate score for ticket %d: %w", tID, err):
					case <-ctx.Done():
					}
					return
				}

				select {
				case resultChan <- ticketScore:
				case <-ctx.Done():
					return
				}
			}(ticketID)
		}

		wg.Wait()
	}()

	return resultChan, errorChan
}

// calculateTicketScore calculates scores for all categories for a single ticket
func (s *TicketScoresService) calculateTicketScore(ctx context.Context, ticketID int, categories []models.RatingCategory) (TicketScore, error) {
	ticketScore := TicketScore{
		TicketID:   ticketID,
		Categories: make([]TicketCategoryScore, 0, len(categories)),
	}

	// Use a channel to collect category scores concurrently
	type categoryResult struct {
		categoryName string
		score        string
		err          error
	}

	resultChan := make(chan categoryResult, len(categories))
	var wg sync.WaitGroup

	// Calculate scores for each category concurrently
	for _, category := range categories {
		wg.Add(1)
		go func(cat models.RatingCategory) {
			defer wg.Done()

			ratings, err := s.ratingsRepo.GetByTicketIDAndCategoryID(ctx, ticketID, cat.ID)
			if err != nil {
				resultChan <- categoryResult{
					categoryName: cat.Name,
					score:        "N/A",
					err:          err,
				}
				return
			}

			var score string
			if len(ratings) == 0 {
				score = "N/A"
			} else {
				calculatedScore, err := s.ticketScoreServ.CalculateScore(ratings, []models.RatingCategory{cat})
				if err != nil {
					score = "N/A"
				} else {
					score = formatScore(calculatedScore)
				}
			}

			resultChan <- categoryResult{
				categoryName: cat.Name,
				score:        score,
				err:          nil,
			}
		}(category)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for result := range resultChan {
		if result.err != nil {
			return ticketScore, fmt.Errorf("failed to calculate score for category %s: %w", result.categoryName, result.err)
		}

		ticketScore.Categories = append(ticketScore.Categories, TicketCategoryScore{
			CategoryName: result.categoryName,
			Score:        result.score,
		})
	}

	return ticketScore, nil
}
