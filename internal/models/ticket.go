package models

import "time"

type Ticket struct {
	ID        int       `json:"id" db:"id"`
	Subject   string    `json:"subject" db:"subject"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}