package models

type RatingCategory struct {
	ID     int    `json:"id" db:"id"`
	Name   string `json:"name" db:"name"`
	Weight int    `json:"weight" db:"weight"`
}