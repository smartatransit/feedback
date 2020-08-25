package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const (
	//SaveFeedbackSQL a prepared Postgres statements for saving a new feedback record
	SaveFeedbackSQL = `
INSERT INTO feedbacks
  (session_id, role, kind, message, value, email)
  VALUES ($1, $2, $3, $4, $5, $6)`

	//GetRecentOutagesSQL a prepared Postgres statements for getting recent outages
	GetRecentOutagesSQL = `
SELECT id, session_id, role, kind, message, received_moment, silenced FROM feedbacks
  WHERE kind = 'outage'
    AND received_moment > $1
    AND NOT silenced`
)

//Feedback represents a user feedback record
type Feedback struct {
	ID         string
	SessionID  string
	Role       string
	ReceivedAt time.Time
	Kind       string
	Silenced   bool
	Message    *string
	Value      *string
	Email      *string
}

//Client implements DB
type Client struct {
	db       DBDriver
	migrator Migrator
}

//New returns a new Client with the speficied dependencies
func New(
	db DBDriver,
	migrator Migrator,
) Client {
	return Client{
		db:       db,
		migrator: migrator,
	}
}

//DB exposes basic database operations
//go:generate counterfeiter . DB
type DB interface {
	Migrate(ctx context.Context) error
	SaveFeedback(ctx context.Context, fb Feedback) error
	GetRecentOutages(ctx context.Context, since time.Time) ([]Feedback, error)
}

//Migrate runs any pending migrations
func (c Client) Migrate(ctx context.Context) error {
	if err := c.migrator.Up(); err != nil {
		return fmt.Errorf("failed migrating database: %w", err)
	}

	return nil
}

//SaveFeedback saves a single new feedback record
func (c Client) SaveFeedback(ctx context.Context, fb Feedback) error {
	_, err := c.db.ExecContext(ctx, SaveFeedbackSQL,
		fb.SessionID, fb.Role, fb.Kind, fb.Message, fb.Value, fb.Email,
	)
	if err != nil {
		return fmt.Errorf("failed saving feedback: %w", err)
	}

	return nil
}

//GetRecentOutages returns all user-submitted outages since `since`
func (c Client) GetRecentOutages(ctx context.Context, since time.Time) ([]Feedback, error) {
	rows, err := c.db.QueryContext(ctx, GetRecentOutagesSQL, since)
	if err != nil {
		return nil, fmt.Errorf("failed saving feedback: %w", err)
	}

	result := []Feedback{}
	for rows.Next() {
		var fb Feedback
		err = rows.Scan(
			&fb.ID,
			&fb.SessionID,
			&fb.Role,
			&fb.Kind,
			&fb.Message,
			&fb.ReceivedAt,
			&fb.Silenced,
		)
		if err != nil {
			return nil, fmt.Errorf("failed scanning feedback results: %w", err)
		}

		result = append(result, fb)
	}

	return result, nil
}

//Migrator is for generating fakes
//go:generate counterfeiter . Migrator
type Migrator interface {
	Up() error
}

//DBDriver is for generating fakes
//go:generate counterfeiter . DBDriver
type DBDriver interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}
