package utils

import (
	"fmt"
	"time"
)

// FormatScore formats a float score as a percentage string
func FormatScore(score float64) string {
	if score == 0 {
		return "0%"
	}
	return fmt.Sprintf("%.0f%%", score)
}

// FormatDateRange formats a date range for display
func FormatDateRange(startDate, endDate time.Time) string {
	if startDate.Equal(endDate) {
		return startDate.Format("2006-01-02")
	}
	return fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
}
