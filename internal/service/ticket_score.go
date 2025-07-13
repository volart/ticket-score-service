package service

import (
	"fmt"
	"ticket-score-service/internal/models"
)

type TicketScoreService struct{}

func NewTicketScoreService() *TicketScoreService {
	return &TicketScoreService{}
}

// The algorithm:
// Calculates weighted scores: rating × weight for each category
// Normalizes against maximum possible score: weight × 5
// Returns percentage => (total weighted score / total max possible score) * 100
func (s *TicketScoreService) CalculateScore(ratings []models.Rating,
	categories []models.RatingCategory) (float64, error) {
	if len(ratings) == 0 {
		return 0, fmt.Errorf("no ratings provided")
	}

	categoryWeights := make(map[int]int)
	for _, category := range categories {
		categoryWeights[category.ID] = category.Weight
	}

	var totalWeightedScore float64
	var totalMaxPossibleScore float64

	for _, rating := range ratings {
		weight, exists := categoryWeights[rating.RatingCategoryID]
		if !exists {
			return 0, fmt.Errorf("rating category %d not found",
				rating.RatingCategoryID)
		}

		if rating.Rating < 0 || rating.Rating > 5 {
			return 0, fmt.Errorf("rating value %d is out of range (0-5)",
				rating.Rating)
		}

		totalWeightedScore += float64(rating.Rating * weight)
		totalMaxPossibleScore += float64(weight * 5)
	}

	if totalMaxPossibleScore == 0 {
		return 0, fmt.Errorf("total possible score is zero")
	}

	score := (totalWeightedScore / totalMaxPossibleScore) * 100
	return score, nil
}
