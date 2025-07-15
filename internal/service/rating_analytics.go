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
	GetByDateRangePaginated(ctx context.Context, startDate, endDate time.Time, limit, offset int) ([]models.Rating, error)
	CountByDateRange(ctx context.Context, startDate, endDate time.Time) (int, error)
	GetDistinctTicketIDsByDateRange(ctx context.Context, startDate, endDate time.Time) ([]int, error)
	GetByTicketIDAndCategoryID(ctx context.Context, ticketID, categoryID int) ([]models.Rating, error)
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

	scores, totalRatings, err := s.calculateScores(ctx, category, startDate, endDate)
	if err != nil {
		return analytics, err
	}

	analytics.Dates = scores
	analytics.Ratings = len(totalRatings)
	analytics.Score = s.calculateOverallScore(totalRatings, category)

	return analytics, nil
}

func (s *RatingAnalyticsService) calculateScores(ctx context.Context, category models.RatingCategory, startDate, endDate time.Time) ([]DailyScore, []models.Rating, error) {
	if s.shouldUseWeeklyAggregation(startDate, endDate) {
		return s.calculateWeeklyScores(ctx, category, startDate, endDate)
	}
	return s.calculateDailyScores(ctx, category, startDate, endDate)
}

func (s *RatingAnalyticsService) calculateDailyScores(ctx context.Context, category models.RatingCategory, startDate, endDate time.Time) ([]DailyScore, []models.Rating, error) {
	var scores []DailyScore
	var totalRatings []models.Rating

	currentDate := startDate
	for !currentDate.After(endDate) {
		dailyRatings, err := s.ratingsRepo.GetByCategoryIDAndDate(ctx, category.ID, currentDate)
		if err != nil {
			return nil, nil, err
		}

		dateStr := currentDate.Format("2006-01-02")
		dailyScore := s.calculateDailyScore(dailyRatings, category, dateStr)
		scores = append(scores, dailyScore)

		if len(dailyRatings) > 0 {
			totalRatings = append(totalRatings, dailyRatings...)
		}

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return scores, totalRatings, nil
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

	score, err := s.ticketScoreServ.CalculateScore(totalRatings, []models.RatingCategory{category})
	if err != nil {
		return "N/A"
	}

	return formatScore(score)
}

func (s *RatingAnalyticsService) shouldUseWeeklyAggregation(startDate, endDate time.Time) bool {
	duration := endDate.Sub(startDate)
	return duration > 30*24*time.Hour // More than 30 days
}

func (s *RatingAnalyticsService) calculateWeeklyScores(ctx context.Context, category models.RatingCategory, startDate, endDate time.Time) ([]DailyScore, []models.Rating, error) {
	var weeklyScores []DailyScore
	var totalRatings []models.Rating

	currentWeekStart := s.getWeekStart(startDate)

	for !currentWeekStart.After(endDate) {
		weekEnd := currentWeekStart.AddDate(0, 0, 6)
		if weekEnd.After(endDate) {
			weekEnd = endDate
		}

		weeklyRatings, err := s.getRatingsForDateRange(ctx, category.ID, currentWeekStart, weekEnd)
		if err != nil {
			return nil, nil, err
		}

		weekStr := fmt.Sprintf("%s to %s", currentWeekStart.Format("2006-01-02"), weekEnd.Format("2006-01-02"))
		weeklyScore := s.calculatePeriodScore(weeklyRatings, category, weekStr)
		weeklyScores = append(weeklyScores, weeklyScore)

		if len(weeklyRatings) > 0 {
			totalRatings = append(totalRatings, weeklyRatings...)
		}

		currentWeekStart = currentWeekStart.AddDate(0, 0, 7)
	}

	return weeklyScores, totalRatings, nil
}

func (s *RatingAnalyticsService) getWeekStart(date time.Time) time.Time {
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7
	}
	return date.AddDate(0, 0, -(weekday - 1))
}

func (s *RatingAnalyticsService) getRatingsForDateRange(ctx context.Context, categoryID int, startDate, endDate time.Time) ([]models.Rating, error) {
	var allRatings []models.Rating

	currentDate := startDate
	for !currentDate.After(endDate) {
		dailyRatings, err := s.ratingsRepo.GetByCategoryIDAndDate(ctx, categoryID, currentDate)
		if err != nil {
			return nil, err
		}
		allRatings = append(allRatings, dailyRatings...)
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return allRatings, nil
}

func (s *RatingAnalyticsService) calculatePeriodScore(ratings []models.Rating, category models.RatingCategory, periodStr string) DailyScore {
	if len(ratings) == 0 {
		return DailyScore{
			Date:  periodStr,
			Score: "N/A",
		}
	}

	score, err := s.ticketScoreServ.CalculateScore(ratings, []models.RatingCategory{category})
	if err != nil {
		return DailyScore{
			Date:  periodStr,
			Score: "N/A",
		}
	}

	return DailyScore{
		Date:  periodStr,
		Score: formatScore(score),
	}
}

func formatScore(score float64) string {
	return fmt.Sprintf("%.0f%%", score)
}
