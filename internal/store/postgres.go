package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/2spy/vinted-discord-bot/pkg/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore owns durable listing and evaluation records. Redis remains the
// short-lived delivery claim store; business state belongs here.
type PostgresStore struct {
	db *pgxpool.Pool
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

// UpsertListing persists the latest source observation and returns its ID.
func (s *PostgresStore) UpsertListing(ctx context.Context, item models.Item) (int64, error) {
	rawPayload, err := json.Marshal(item)
	if err != nil {
		return 0, fmt.Errorf("marshal listing payload: %w", err)
	}
	const query = `
INSERT INTO listings (platform, external_id, url, title, brand, size, condition, purchase_price_cents, currency, raw_payload)
VALUES ($1, $2, $3, $4, $5, $6, $7, ROUND($8 * 100), $9, $10)
ON CONFLICT (platform, external_id) DO UPDATE SET
  url = EXCLUDED.url, title = EXCLUDED.title, brand = EXCLUDED.brand,
  size = EXCLUDED.size, condition = EXCLUDED.condition,
  purchase_price_cents = EXCLUDED.purchase_price_cents, currency = EXCLUDED.currency,
  raw_payload = EXCLUDED.raw_payload, last_seen_at = now()
RETURNING id`
	var id int64
	if err := s.db.QueryRow(ctx, query, item.Platform, item.ID, item.URL, item.Title, item.Brand, item.Size, item.Condition, item.Price, item.Currency, rawPayload).Scan(&id); err != nil {
		return 0, fmt.Errorf("upsert listing: %w", err)
	}
	return id, nil
}
