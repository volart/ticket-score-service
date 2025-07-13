package service

import (
	"testing"
	"ticket-score-service/internal/models"
)

func TestCalculateScore(t *testing.T) {
	service := NewTicketScoreService()

	t.Run("valid ratings with equal weights", func(t *testing.T) {
		ratings := []models.Rating{
			{Rating: 4, RatingCategoryID: 1},
			{Rating: 2, RatingCategoryID: 2},
		}
		categories := []models.RatingCategory{
			{ID: 1, Weight: 1},
			{ID: 2, Weight: 1},
		}

		score, err := service.CalculateScore(ratings, categories)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expected := 60.0 // (4*1 + 2*1) / (1*5 + 1*5) * 100 = 6/10 * 100 = 60
		if score != expected {
			t.Errorf("Expected score %f, got %f", expected, score)
		}
	})

	t.Run("valid ratings with different weights", func(t *testing.T) {
		ratings := []models.Rating{
			{Rating: 4, RatingCategoryID: 1},
			{Rating: 2, RatingCategoryID: 2},
		}
		categories := []models.RatingCategory{
			{ID: 1, Weight: 3},
			{ID: 2, Weight: 1},
		}

		score, err := service.CalculateScore(ratings, categories)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expected := 70.0 // (4*3 + 2*1) / (3*5 + 1*5) * 100 = 14/20 * 100 = 70
		if score != expected {
			t.Errorf("Expected score %f, got %f", expected, score)
		}
	})

	t.Run("perfect score", func(t *testing.T) {
		ratings := []models.Rating{
			{Rating: 5, RatingCategoryID: 1},
			{Rating: 5, RatingCategoryID: 2},
		}
		categories := []models.RatingCategory{
			{ID: 1, Weight: 2},
			{ID: 2, Weight: 3},
		}

		score, err := service.CalculateScore(ratings, categories)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expected := 100.0
		if score != expected {
			t.Errorf("Expected score %f, got %f", expected, score)
		}
	})

	t.Run("zero score", func(t *testing.T) {
		ratings := []models.Rating{
			{Rating: 0, RatingCategoryID: 1},
			{Rating: 0, RatingCategoryID: 2},
		}
		categories := []models.RatingCategory{
			{ID: 1, Weight: 1},
			{ID: 2, Weight: 1},
		}

		score, err := service.CalculateScore(ratings, categories)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expected := 0.0
		if score != expected {
			t.Errorf("Expected score %f, got %f", expected, score)
		}
	})

	t.Run("empty ratings", func(t *testing.T) {
		ratings := []models.Rating{}
		categories := []models.RatingCategory{
			{ID: 1, Weight: 1},
		}

		_, err := service.CalculateScore(ratings, categories)
		if err == nil {
			t.Error("Expected error for empty ratings")
		}
	})

	t.Run("rating out of range - too high", func(t *testing.T) {
		ratings := []models.Rating{
			{Rating: 6, RatingCategoryID: 1},
		}
		categories := []models.RatingCategory{
			{ID: 1, Weight: 1},
		}

		_, err := service.CalculateScore(ratings, categories)
		if err == nil {
			t.Error("Expected error for rating out of range")
		}
	})

	t.Run("rating out of range - negative", func(t *testing.T) {
		ratings := []models.Rating{
			{Rating: -1, RatingCategoryID: 1},
		}
		categories := []models.RatingCategory{
			{ID: 1, Weight: 1},
		}

		_, err := service.CalculateScore(ratings, categories)
		if err == nil {
			t.Error("Expected error for negative rating")
		}
	})

	t.Run("category not found", func(t *testing.T) {
		ratings := []models.Rating{
			{Rating: 4, RatingCategoryID: 99},
		}
		categories := []models.RatingCategory{
			{ID: 1, Weight: 1},
		}

		_, err := service.CalculateScore(ratings, categories)
		if err == nil {
			t.Error("Expected error for missing category")
		}
	})

	t.Run("zero total max possible score", func(t *testing.T) {
		ratings := []models.Rating{
			{Rating: 4, RatingCategoryID: 1},
		}
		categories := []models.RatingCategory{
			{ID: 1, Weight: 0},
		}

		_, err := service.CalculateScore(ratings, categories)
		if err == nil {
			t.Error("Expected error for zero total possible score")
		}
	})

	t.Run("zero weight", func(t *testing.T) {
		ratings := []models.Rating{
			{Rating: 4, RatingCategoryID: 1},
			{Rating: 2, RatingCategoryID: 2},
		}
		categories := []models.RatingCategory{
			{ID: 1, Weight: 1},
			{ID: 2, Weight: 0},
		}

		score, err := service.CalculateScore(ratings, categories)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expected := 80.0 // (4*1 + 2*0) / (1*5 + 0*5) * 100 = 4/5 * 100 = 80
		if score != expected {
			t.Errorf("Expected score %f, got %f", expected, score)
		}
	})
}
