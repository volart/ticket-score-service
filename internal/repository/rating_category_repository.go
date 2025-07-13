package repository

import (
	"context"
	"database/sql"
	"fmt"

	"ticket-score-service/internal/models"
)

type RatingCategoryRepository struct {
	db *sql.DB
}

func NewRatingCategoryRepository(db *sql.DB) *RatingCategoryRepository {
	return &RatingCategoryRepository{
		db: db,
	}
}

func (r *RatingCategoryRepository) GetAll(ctx context.Context) ([]models.RatingCategory, error) {
	query := `SELECT id, name, weight FROM rating_categories ORDER BY id`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rating categories: %w", err)
	}
	defer rows.Close()

	var categories []models.RatingCategory
	for rows.Next() {
		var category models.RatingCategory
		if err := rows.Scan(&category.ID, &category.Name, &category.Weight); err != nil {
			return nil, fmt.Errorf("failed to scan rating category: %w", err)
		}
		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return categories, nil
}
