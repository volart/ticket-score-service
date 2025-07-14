package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"ticket-score-service/internal/models"
)

type TicketRepository struct {
	db *sql.DB
}

func NewTicketRepository(db *sql.DB) *TicketRepository {
	return &TicketRepository{
		db: db,
	}
}

func (r *TicketRepository) GetByCreatedDateRange(ctx context.Context, startDate, endDate time.Time) ([]models.Ticket, error) {
	query := `SELECT id, subject, created_at
			  FROM tickets
			  WHERE created_at >= ? AND created_at < ?
			  ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query tickets: %w", err)
	}
	defer rows.Close()

	var tickets []models.Ticket
	for rows.Next() {
		var ticket models.Ticket
		if err := rows.Scan(&ticket.ID, &ticket.Subject, &ticket.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan ticket: %w", err)
		}
		tickets = append(tickets, ticket)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return tickets, nil
}
