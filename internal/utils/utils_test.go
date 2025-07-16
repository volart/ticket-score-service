package utils

import (
	"testing"
	"time"
)

func TestFormatScore(t *testing.T) {
	tests := []struct {
		name     string
		score    float64
		expected string
	}{
		{
			name:     "regular score",
			score:    80.0,
			expected: "80%",
		},
		{
			name:     "rounded score",
			score:    75.5,
			expected: "76%",
		},
		{
			name:     "zero score",
			score:    0.0,
			expected: "0%",
		},
		{
			name:     "perfect score",
			score:    100.0,
			expected: "100%",
		},
		{
			name:     "decimal score",
			score:    85.7,
			expected: "86%",
		},
		{
			name:     "low decimal score",
			score:    85.3,
			expected: "85%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatScore(tt.score)
			if result != tt.expected {
				t.Errorf("FormatScore(%.1f) = %s, expected %s", tt.score, result, tt.expected)
			}
		})
	}
}

func TestFormatDateRange(t *testing.T) {
	tests := []struct {
		name      string
		startDate time.Time
		endDate   time.Time
		expected  string
	}{
		{
			name:      "same date",
			startDate: time.Date(2019, 10, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2019, 10, 1, 0, 0, 0, 0, time.UTC),
			expected:  "2019-10-01",
		},
		{
			name:      "different dates",
			startDate: time.Date(2019, 10, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2019, 10, 7, 0, 0, 0, 0, time.UTC),
			expected:  "2019-10-01 to 2019-10-07",
		},
		{
			name:      "cross month range",
			startDate: time.Date(2019, 9, 28, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2019, 10, 5, 0, 0, 0, 0, time.UTC),
			expected:  "2019-09-28 to 2019-10-05",
		},
		{
			name:      "cross year range",
			startDate: time.Date(2019, 12, 30, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2020, 1, 3, 0, 0, 0, 0, time.UTC),
			expected:  "2019-12-30 to 2020-01-03",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDateRange(tt.startDate, tt.endDate)
			if result != tt.expected {
				t.Errorf("FormatDateRange(%v, %v) = %s, expected %s",
					tt.startDate.Format("2006-01-02"), tt.endDate.Format("2006-01-02"),
					result, tt.expected)
			}
		})
	}
}
