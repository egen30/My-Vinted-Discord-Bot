package store

import (
	"context"
	"fmt"

	"github.com/2spy/vinted-discord-bot/pkg/history"
	"github.com/2spy/vinted-discord-bot/pkg/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore owns durable listing and evaluation records. Redis remains the
// short-lived delivery claim store; business state belongs here.
type PostgresStore struct {
	db *pgxpool.Pool
}

func (s *PostgresStore) CreateSearch(ctx context.Context, search models.Search) (models.Search, error) {
	const query = `INSERT INTO searches (name, url, enabled, priority, notes) VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, url, enabled, priority, notes, created_at, last_attempted_at, last_successful_at, last_error`
	return scanSearch(s.db.QueryRow(ctx, query, search.Name, search.URL, search.Enabled, search.Priority, search.Notes))
}

func (s *PostgresStore) ListSearches(ctx context.Context, enabledOnly bool) ([]models.Search, error) {
	query := `SELECT id, name, url, enabled, priority, notes, created_at, last_attempted_at, last_successful_at, last_error FROM searches`
	if enabledOnly {
		query += ` WHERE enabled = TRUE`
	}
	query += ` ORDER BY priority DESC, id ASC`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list searches: %w", err)
	}
	defer rows.Close()
	var searches []models.Search
	for rows.Next() {
		search, scanErr := scanSearch(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan search: %w", scanErr)
		}
		searches = append(searches, search)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate searches: %w", err)
	}
	return searches, nil
}

func (s *PostgresStore) SetSearchEnabled(ctx context.Context, id int64, enabled bool) error {
	commandTag, err := s.db.Exec(ctx, `UPDATE searches SET enabled = $1 WHERE id = $2`, enabled, id)
	if err != nil {
		return fmt.Errorf("update search: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return fmt.Errorf("search %d not found", id)
	}
	return nil
}

func (s *PostgresStore) DeleteSearch(ctx context.Context, id int64) error {
	commandTag, err := s.db.Exec(ctx, `DELETE FROM searches WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete search: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return fmt.Errorf("search %d not found", id)
	}
	return nil
}

func (s *PostgresStore) UpdateSearch(ctx context.Context, search models.Search) (models.Search, error) {
	const query = `UPDATE searches SET name = $1, url = $2, enabled = $3, priority = $4, notes = $5 WHERE id = $6
RETURNING id, name, url, enabled, priority, notes, created_at, last_attempted_at, last_successful_at, last_error`
	updated, err := scanSearch(s.db.QueryRow(ctx, query, search.Name, search.URL, search.Enabled, search.Priority, search.Notes, search.ID))
	if err != nil {
		return models.Search{}, fmt.Errorf("update search: %w", err)
	}
	return updated, nil
}

func (s *PostgresStore) RecordSearchAttempt(ctx context.Context, id int64, runErr error) error {
	if runErr == nil {
		_, err := s.db.Exec(ctx, `UPDATE searches SET last_attempted_at = now(), last_successful_at = now(), last_error = '' WHERE id = $1`, id)
		return err
	}
	_, err := s.db.Exec(ctx, `UPDATE searches SET last_attempted_at = now(), last_error = $2 WHERE id = $1`, id, runErr.Error())
	return err
}

// ReplaceSalesHistory atomically publishes the latest validated sheet snapshot.
// It intentionally stores only business history, never raw Vinted discoveries.
func (s *PostgresStore) ReplaceSalesHistory(ctx context.Context, sales []history.Sale) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin history replacement: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM sales_history`); err != nil {
		return fmt.Errorf("clear sales history: %w", err)
	}
	for _, sale := range sales {
		if _, err := tx.Exec(ctx, `INSERT INTO sales_history (model, size, condition, purchase_price_cents, sale_price_cents, costs_cents, purchased_at, sold_at, source) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, sale.Model, sale.Size, sale.Condition, sale.PurchaseCents, sale.SaleCents, sale.CostsCents, sale.PurchasedAt, sale.SoldAt, sale.Source); err != nil {
			return fmt.Errorf("insert sales history: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit history replacement: %w", err)
	}
	return nil
}

type searchRow interface {
	Scan(dest ...any) error
}

func scanSearch(row searchRow) (models.Search, error) {
	var search models.Search
	err := row.Scan(&search.ID, &search.Name, &search.URL, &search.Enabled, &search.Priority, &search.Notes, &search.CreatedAt, &search.LastAttemptedAt, &search.LastSuccessfulAt, &search.LastError)
	return search, err
}

func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database URL is required")
	}
	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}
	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Close() {
	if s != nil && s.db != nil {
		s.db.Close()
	}
}
