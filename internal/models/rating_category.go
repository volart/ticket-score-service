package models

type RatingCategory struct {
	ID     int     `json:"id" db:"id"`
	Name   string  `json:"name" db:"name"`
	Weight float64 `json:"weight" db:"weight"`
}
