package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"ticket-score-service/internal/models"
)

type RatingsRepository struct {
	db *sql.DB
}

func NewRatingsRepository(db *sql.DB) *RatingsRepository {
	return &RatingsRepository{
		db: db,
	}
}

func (r *RatingsRepository) GetByCategoryIDAndDate(ctx context.Context, categoryID int, date time.Time) ([]models.Rating, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := `SELECT id, rating, ticket_id, rating_category_id, reviewer_id, reviewee_id, created_at 
			  FROM ratings 
			  WHERE rating_category_id = ? AND created_at >= ? AND created_at < ?
			  ORDER BY created_at`

	rows, err := r.db.QueryContext(ctx, query, categoryID, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("failed to query ratings: %w", err)
	}
	defer rows.Close()

	var ratings []models.Rating
	for rows.Next() {
		var rating models.Rating
		if err := rows.Scan(&rating.ID, &rating.Rating, &rating.TicketID, &rating.RatingCategoryID, &rating.ReviewerID, &rating.RevieweeID, &rating.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan rating: %w", err)
		}
		ratings = append(ratings, rating)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return ratings, nil
}