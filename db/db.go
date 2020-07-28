package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	migrate "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	//SaveFeedbackSQL a prepared Postgres statements for saving a new feedback record
	SaveFeedbackSQL = `
INSERT INTO feedbacks
  (session_id, role, kind, message)
  VALUES ($1, $2, $3, $4)`

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
	Message    string
}

//Client implements DB
type Client struct {
	migrationsPath     string
	pgConnectionString string
	pg                 *sql.DB
}

//New returns a new Client
func New(
	migrationsPath string,
	pgConnectionString string,
	pg *sql.DB,
) Client {
	return Client{
		migrationsPath:     migrationsPath,
		pgConnectionString: pgConnectionString,
		pg:                 pg,
	}
}

//DB exposes basic database operations
//go:generate counterfeiter . DB
type DB interface {
	Migrate(ctx context.Context) error
	SaveFeedback(ctx context.Context, fb Feedback) error
	GetRecentOutages(ctx context.Context, since time.Time) ([]Feedback, error)
	TestConnection(ctx context.Context) error
}

//Migrate runs any pending migrations
func (c Client) Migrate(ctx context.Context) error {
	m, err := migrate.New("file://"+c.migrationsPath, c.pgConnectionString)
	if err != nil {
		return fmt.Errorf("failed initializing migration client: %w", err)
	}
	if err := m.Up(); err != nil {
		return fmt.Errorf("failed migrating database: %w", err)
	}

	return nil
}

//SaveFeedback saves a single new feedback record
func (c Client) SaveFeedback(ctx context.Context, fb Feedback) error {
	_, err := c.pg.ExecContext(ctx, SaveFeedbackSQL,
		fb.SessionID, fb.Role, fb.Kind, fb.Message,
	)
	if err != nil {
		return fmt.Errorf("failed saving feedback: %w", err)
	}

	return nil
}

//GetRecentOutages returns all user-submitted outages since `since`
func (c Client) GetRecentOutages(ctx context.Context, since time.Time) ([]Feedback, error) {
	rows, err := c.pg.QueryContext(ctx, GetRecentOutagesSQL, since)
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

func (c Client) TestConnection(ctx context.Context) error {
	return c.pg.PingContext(ctx)
}
