package service

import (
	"context"
	"fmt"
	"time"

	"ticket-score-service/internal/models"
)

type DailyScore struct {
	Date  string `json:"date"`
	Score string `json:"score"`
}

type CategoryAnalytics struct {
	Category string       `json:"category"`
	Ratings  int          `json:"ratings"`
	Dates    []DailyScore `json:"dates"`
	Score    string       `json:"score"`
}

type CategoryRepository interface {
	GetAll(ctx context.Context) ([]models.RatingCategory, error)
}

type RatingsRepository interface {
	GetByCategoryIDAndDate(ctx context.Context, categoryID int, date time.Time) ([]models.Rating, error)
}

type ScoreCalculator interface {
	CalculateScore(ratings []models.Rating, categories []models.RatingCategory) (float64, error)
}

type RatingAnalyticsService struct {
	categoryRepo    CategoryRepository
	ratingsRepo     RatingsRepository
	ticketScoreServ ScoreCalculator
}

func NewRatingAnalyticsService(
	categoryRepo CategoryRepository,
	ratingsRepo RatingsRepository,
	ticketScoreServ ScoreCalculator,
) *RatingAnalyticsService {
	return &RatingAnalyticsService{
		categoryRepo:    categoryRepo,
		ratingsRepo:     ratingsRepo,
		ticketScoreServ: ticketScoreServ,
	}
}

func (s *RatingAnalyticsService) GetCategoryAnalytics(ctx context.Context, startDate, endDate time.Time) ([]CategoryAnalytics, error) {
	categories, err := s.categoryRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	var results []CategoryAnalytics
	for _, category := range categories {
		analytics, err := s.processCategoryAnalytics(ctx, category, startDate, endDate)
		if err != nil {
			return nil, err
		}
		results = append(results, analytics)
	}

	return results, nil
}

func (s *RatingAnalyticsService) processCategoryAnalytics(ctx context.Context, category models.RatingCategory, startDate, endDate time.Time) (CategoryAnalytics, error) {
	analytics := CategoryAnalytics{
		Category: category.Name,
		Ratings:  0,
		Dates:    []DailyScore{},
	}

	dailyScores, totalRatings, err := s.calculateDailyScores(ctx, category, startDate, endDate)
	if err != nil {
		return analytics, err
	}

	analytics.Dates = dailyScores
	analytics.Ratings = len(totalRatings)
	analytics.Score = s.calculateOverallScore(totalRatings, category)

	return analytics, nil
}

func (s *RatingAnalyticsService) calculateDailyScores(ctx context.Context, category models.RatingCategory, startDate, endDate time.Time) ([]DailyScore, []models.Rating, error) {
	var dailyScores []DailyScore
	var totalRatings []models.Rating

	currentDate := startDate
	for !currentDate.After(endDate) {
		dailyRatings, err := s.ratingsRepo.GetByCategoryIDAndDate(ctx, category.ID, currentDate)
		if err != nil {
			return nil, nil, err
		}

		dateStr := currentDate.Format("2006-01-02")
		dailyScore := s.calculateDailyScore(dailyRatings, category, dateStr)
		dailyScores = append(dailyScores, dailyScore)

		if len(dailyRatings) > 0 {
			totalRatings = append(totalRatings, dailyRatings...)
		}

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return dailyScores, totalRatings, nil
}

func (s *RatingAnalyticsService) calculateDailyScore(dailyRatings []models.Rating, category models.RatingCategory, dateStr string) DailyScore {
	if len(dailyRatings) == 0 {
		return DailyScore{
			Date:  dateStr,
			Score: "N/A",
		}
	}

	score, err := s.ticketScoreServ.CalculateScore(dailyRatings, []models.RatingCategory{category})
	if err != nil {
		return DailyScore{
			Date:  dateStr,
			Score: "N/A",
		}
	}

	return DailyScore{
		Date:  dateStr,
		Score: formatScore(score),
	}
}

func (s *RatingAnalyticsService) calculateOverallScore(totalRatings []models.Rating, category models.RatingCategory) string {
	if len(totalRatings) == 0 {
		return "N/A"
	}

	var totalWeightedScore float64
	var totalMaxPossibleScore float64

	for _, rating := range totalRatings {
		totalWeightedScore += float64(rating.Rating * category.Weight)
		totalMaxPossibleScore += float64(category.Weight * 5)
	}

	if totalMaxPossibleScore == 0 {
		return "N/A"
	}

	totalScore := (totalWeightedScore / totalMaxPossibleScore) * 100
	return formatScore(totalScore)
}

func formatScore(score float64) string {
	return fmt.Sprintf("%.0f%%", score)
}