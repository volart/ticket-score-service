package models

import "time"

type Rating struct {
	ID               int       `json:"id" db:"id"`
	Rating           int       `json:"rating" db:"rating"`
	TicketID         int       `json:"ticket_id" db:"ticket_id"`
	RatingCategoryID int       `json:"rating_category_id" db:"rating_category_id"`
	ReviewerID       int       `json:"reviewer_id" db:"reviewer_id"`
	RevieweeID       int       `json:"reviewee_id" db:"reviewee_id"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}
