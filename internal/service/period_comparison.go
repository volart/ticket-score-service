package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// PeriodComparisonResult represents the result of comparing two periods
type PeriodComparisonResult struct {
	StartPeriod string `json:"start_period"`
	StartScore  string `json:"start_score"`
	EndPeriod   string `json:"end_period"`
	EndScore    string `json:"end_score"`
	Difference  string `json:"difference"`
}

// PeriodComparisonService handles period over period comparisons
type PeriodComparisonService struct {
	overallQualityService *OverallQualityService
}

// NewPeriodComparisonService creates a new period comparison service instance
func NewPeriodComparisonService(overallQualityService *OverallQualityService) *PeriodComparisonService {
	return &PeriodComparisonService{
		overallQualityService: overallQualityService,
	}
}

// GetPeriodComparison compares overall quality scores between two time periods
func (s *PeriodComparisonService) GetPeriodComparison(
	ctx context.Context,
	firstStartDate, firstEndDate, secondStartDate, secondEndDate time.Time,
) (*PeriodComparisonResult, error) {
	// Get overall quality score for first period
	firstPeriodScore, err := s.overallQualityService.GetOverallQualityScore(ctx, firstStartDate, firstEndDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get first period score: %w", err)
	}

	// Get overall quality score for second period
	secondPeriodScore, err := s.overallQualityService.GetOverallQualityScore(ctx, secondStartDate, secondEndDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get second period score: %w", err)
	}

	// Calculate difference (from first to second period)
	difference := s.calculateDifference(firstPeriodScore.Score, secondPeriodScore.Score)

	return &PeriodComparisonResult{
		StartPeriod: secondPeriodScore.Period, // Most recent period (second)
		StartScore:  secondPeriodScore.Score,  // Most recent score (second)
		EndPeriod:   firstPeriodScore.Period,  // Older period (first)
		EndScore:    firstPeriodScore.Score,   // Older score (first)
		Difference:  difference,
	}, nil
}

// Calculates relative percentage change
// Returns the relative change as a formatted string with proper sign
func (s *PeriodComparisonService) calculateDifference(firstScore, secondScore string) string {
	// Handle N/A cases
	if firstScore == "N/A" || secondScore == "N/A" {
		return "N/A"
	}

	// Parse first score (remove % sign)
	firstScoreStr := strings.TrimSuffix(firstScore, "%")
	value1, err := strconv.ParseFloat(firstScoreStr, 64)
	if err != nil {
		return "N/A"
	}

	// Parse second score (remove % sign)
	secondScoreStr := strings.TrimSuffix(secondScore, "%")
	value2, err := strconv.ParseFloat(secondScoreStr, 64)
	if err != nil {
		return "N/A"
	}

	// Handle division by zero
	if value1 == 0 {
		if value2 == 0 {
			return "0.0%"
		}
		return "N/A" // Cannot calculate percentage change from zero
	}

	// Calculate relative percentage change: ((value2 - value1) / value1) * 100
	relativeChange := ((value2 - value1) / value1) * 100

	// Format the relative change with proper sign to 1 decimal place
	if relativeChange > 0 {
		return fmt.Sprintf("+%.1f%%", relativeChange)
	} else if relativeChange < 0 {
		return fmt.Sprintf("%.1f%%", relativeChange) // negative sign included automatically
	} else {
		return "0.0%"
	}
}
