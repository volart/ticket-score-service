package mocks

import (
	"context"
	"fmt"
	"ticket-score-service/internal/models"
	"time"
)

type MockRatingsRepo struct {
	Ratings       map[string][]models.Rating
	Count         int
	PaginationErr error
	CountErr      error
	Err           error
}

func (m *MockRatingsRepo) GetByCategoryIDAndDate(ctx context.Context, categoryID int, date time.Time) ([]models.Rating, error) {
	if m.Err != nil {
		return nil, m.Err
	}

	dateStr := date.Format("2006-01-02")
	key := fmt.Sprintf("%d-%s", categoryID, dateStr)

	if ratings, exists := m.Ratings[key]; exists {
		return ratings, nil
	}

	return []models.Rating{}, nil
}

func (m *MockRatingsRepo) GetDistinctTicketIDsByDateRange(ctx context.Context, startDate, endDate time.Time) ([]int, error) {
	if m.Err != nil {
		return nil, m.Err
	}

	ticketIDMap := make(map[int]bool)
	for _, ratings := range m.Ratings {
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

func (m *MockRatingsRepo) GetByTicketIDAndCategoryID(ctx context.Context, ticketID, categoryID int) ([]models.Rating, error) {
	if m.Err != nil {
		return nil, m.Err
	}

	var results []models.Rating
	for _, ratings := range m.Ratings {
		for _, rating := range ratings {
			if rating.TicketID == ticketID && rating.RatingCategoryID == categoryID {
				results = append(results, rating)
			}
		}
	}

	return results, nil
}

func (m *MockRatingsRepo) GetByDateRangePaginated(ctx context.Context, startDate, endDate time.Time, limit, offset int) ([]models.Rating, error) {
	if m.PaginationErr != nil {
		return nil, m.PaginationErr
	}

	key := fmt.Sprintf("%d:%d", limit, offset)
	if ratings, exists := m.Ratings[key]; exists {
		return ratings, nil
	}
	return []models.Rating{}, nil
}

func (m *MockRatingsRepo) CountByDateRange(ctx context.Context, startDate, endDate time.Time) (int, error) {
	if m.CountErr != nil {
		return 0, m.CountErr
	}
	return m.Count, nil
}
