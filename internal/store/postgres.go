package store

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

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

func (s *PostgresStore) GetHistorySource(ctx context.Context) (models.HistorySource, error) {
	var source models.HistorySource
	err := s.db.QueryRow(ctx, `SELECT spreadsheet_url, worksheet, enabled, updated_at, last_sync_at, last_error, accepted_rows, rejected_rows FROM history_source WHERE id = 1`).Scan(
		&source.SpreadsheetURL, &source.Worksheet, &source.Enabled, &source.UpdatedAt, &source.LastSyncAt, &source.LastError, &source.AcceptedRows, &source.RejectedRows)
	if err != nil {
		return models.HistorySource{}, fmt.Errorf("get history source: %w", err)
	}
	return source, nil
}

func (s *PostgresStore) RecordHistorySync(ctx context.Context, accepted, rejected int, syncErr error) error {
	lastError := ""
	if syncErr != nil {
		lastError = syncErr.Error()
	}
	_, err := s.db.Exec(ctx, `UPDATE history_source SET last_sync_at=CASE WHEN $1 = '' THEN now() ELSE last_sync_at END, last_error=$1, accepted_rows=CASE WHEN $1 = '' THEN $2 ELSE accepted_rows END, rejected_rows=CASE WHEN $1 = '' THEN $3 ELSE rejected_rows END WHERE id=1`, lastError, accepted, rejected)
	if err != nil {
		return fmt.Errorf("record history sync: %w", err)
	}
	return nil
}

func (s *PostgresStore) SaveHistorySource(ctx context.Context, source models.HistorySource) (models.HistorySource, error) {
	const query = `INSERT INTO history_source (id, spreadsheet_url, worksheet, enabled, updated_at) VALUES (1,$1,$2,$3,now())
ON CONFLICT (id) DO UPDATE SET spreadsheet_url=EXCLUDED.spreadsheet_url, worksheet=EXCLUDED.worksheet, enabled=EXCLUDED.enabled, updated_at=now()
RETURNING spreadsheet_url, worksheet, enabled, updated_at, last_sync_at, last_error, accepted_rows, rejected_rows`
	var saved models.HistorySource
	err := s.db.QueryRow(ctx, query, source.SpreadsheetURL, source.Worksheet, source.Enabled).Scan(
		&saved.SpreadsheetURL, &saved.Worksheet, &saved.Enabled, &saved.UpdatedAt, &saved.LastSyncAt, &saved.LastError, &saved.AcceptedRows, &saved.RejectedRows)
	if err != nil {
		return models.HistorySource{}, fmt.Errorf("save history source: %w", err)
	}
	return saved, nil
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

// UpsertListing stores the latest marketplace representation and all search
// attributions in one transaction. A listing identity is scoped by platform
// so another marketplace can reuse the same external ID safely.
func (s *PostgresStore) UpsertListing(ctx context.Context, item models.Item, searchIDs []int64) (int64, error) {
	if len(searchIDs) == 0 {
		return 0, fmt.Errorf("at least one search ID is required")
	}
	images, err := json.Marshal(item.ImageURLs)
	if err != nil {
		return 0, fmt.Errorf("marshal listing images: %w", err)
	}
	priceCents := int64(math.Round(item.Price * 100))
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin listing upsert: %w", err)
	}
	defer tx.Rollback(ctx)
	const upsert = `INSERT INTO listings
 (platform, external_id, url, title, description, brand, size, condition, purchase_price_cents, currency,
  seller_username, seller_profile_url, seller_avatar_url, seller_rating, seller_review_count, country, image_urls,
  published_at, updated_at, last_seen_at)
 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,now())
 ON CONFLICT (platform, external_id) DO UPDATE SET
  url=EXCLUDED.url, title=EXCLUDED.title, description=EXCLUDED.description, brand=EXCLUDED.brand,
  size=EXCLUDED.size, condition=EXCLUDED.condition, purchase_price_cents=EXCLUDED.purchase_price_cents,
  currency=EXCLUDED.currency, seller_username=EXCLUDED.seller_username, seller_profile_url=EXCLUDED.seller_profile_url,
  seller_avatar_url=EXCLUDED.seller_avatar_url, seller_rating=EXCLUDED.seller_rating,
  seller_review_count=EXCLUDED.seller_review_count, country=EXCLUDED.country, image_urls=EXCLUDED.image_urls,
  published_at=EXCLUDED.published_at, updated_at=EXCLUDED.updated_at, last_seen_at=now()
 RETURNING id`
	var listingID int64
	err = tx.QueryRow(ctx, upsert, item.Platform, item.ID, item.URL, item.Title, item.Description, item.Brand,
		item.Size, item.Condition, priceCents, item.Currency, item.Seller.Username, item.Seller.ProfileURL,
		item.Seller.AvatarURL, nullableRating(item.Seller.Rating), nullableInt(item.Seller.ReviewCount),
		item.Seller.Country, images, item.PublishedAt, item.UpdatedAt).Scan(&listingID)
	if err != nil {
		return 0, fmt.Errorf("upsert listing: %w", err)
	}
	seenSearchIDs := make(map[int64]struct{}, len(searchIDs))
	for _, searchID := range searchIDs {
		if _, seen := seenSearchIDs[searchID]; seen {
			continue
		}
		seenSearchIDs[searchID] = struct{}{}
		_, err = tx.Exec(ctx, `INSERT INTO listing_searches (listing_id, search_id) VALUES ($1,$2)
ON CONFLICT (listing_id, search_id) DO UPDATE SET last_discovered_at=now(), discovery_count=listing_searches.discovery_count+1`, listingID, searchID)
		if err != nil {
			return 0, fmt.Errorf("record listing attribution: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit listing upsert: %w", err)
	}
	return listingID, nil
}

func nullableRating(value float64) any {
	if value == 0 {
		return nil
	}
	return value
}

func nullableInt(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

func (s *PostgresStore) RecentListings(ctx context.Context, limit int) ([]models.ListingSummary, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `SELECT l.id, l.external_id, l.title, l.url, l.purchase_price_cents, l.currency,
 l.seller_username, l.first_seen_at, COALESCE(array_agg(DISTINCT se.name) FILTER (WHERE se.name IS NOT NULL), '{}')
 FROM listings l LEFT JOIN listing_searches ls ON ls.listing_id=l.id LEFT JOIN searches se ON se.id=ls.search_id
 GROUP BY l.id ORDER BY l.first_seen_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent listings: %w", err)
	}
	defer rows.Close()
	var result []models.ListingSummary
	for rows.Next() {
		var listing models.ListingSummary
		if err := rows.Scan(&listing.ID, &listing.ExternalID, &listing.Title, &listing.URL, &listing.PriceCents,
			&listing.Currency, &listing.Seller, &listing.FirstSeenAt, &listing.SearchNames); err != nil {
			return nil, fmt.Errorf("scan recent listing: %w", err)
		}
		result = append(result, listing)
	}
	return result, rows.Err()
}

func (s *PostgresStore) RecordNotification(ctx context.Context, listingID int64, channel string, sendErr error) error {
	message := ""
	succeeded := sendErr == nil
	if sendErr != nil {
		message = sendErr.Error()
	}
	_, err := s.db.Exec(ctx, `INSERT INTO notification_deliveries (listing_id, channel, succeeded, error) VALUES ($1,$2,$3,$4)`, listingID, channel, succeeded, message)
	return err
}

// ReplaceSalesHistory atomically publishes the latest validated sheet snapshot.
// It intentionally stores only business history, never raw Vinted discoveries.
func (s *PostgresStore) ReplaceSalesHistory(ctx context.Context, sales []history.Sale) error {
	return s.ReplaceSalesHistorySnapshot(ctx, sales, nil, "legacy")
}

// ReplaceSalesHistorySnapshot publishes a new immutable history snapshot.
// Existing snapshots remain available for auditability and future comparison.
func (s *PostgresStore) ReplaceSalesHistorySnapshot(ctx context.Context, sales []history.Sale, rejected []string, source string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin history replacement: %w", err)
	}
	defer tx.Rollback(ctx)
	var snapshotID int64
	if err := tx.QueryRow(ctx, `INSERT INTO history_snapshots (source, accepted_rows, rejected_rows) VALUES ($1,$2,$3) RETURNING id`, source, len(sales), len(rejected)).Scan(&snapshotID); err != nil {
		return fmt.Errorf("create history snapshot: %w", err)
	}
	for _, sale := range sales {
		if _, err := tx.Exec(ctx, `INSERT INTO sales_history (snapshot_id, source_row, model, original_model, brand, normalized_model, size, normalized_size, condition, normalized_condition, purchase_price_cents, sale_price_cents, costs_cents, purchased_at, sold_at, days_to_sell, source) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`, snapshotID, nullableSourceRow(sale.SourceRow), sale.Model, sale.OriginalModel, sale.Brand, sale.NormalizedModel, sale.Size, sale.NormalizedSize, sale.Condition, sale.NormalizedCondition, sale.PurchaseCents, sale.SaleCents, sale.CostsCents, sale.PurchasedAt, sale.SoldAt, sale.DaysToSell, sale.Source); err != nil {
			return fmt.Errorf("insert sales history: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit history replacement: %w", err)
	}
	return nil
}

func nullableSourceRow(value int) any {
	if value == 0 {
		return nil
	}
	return value
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
	store := &PostgresStore{db: db}
	if err := store.ensureMVPListingSchema(ctx); err != nil {
		store.Close()
		return nil, err
	}
	return store, nil
}

func (s *PostgresStore) Close() {
	if s != nil && s.db != nil {
		s.db.Close()
	}
}

// ensureMVPListingSchema is intentionally additive. Compose initializes fresh
// databases from migrations, while this guard also upgrades an already-running
// installation whose PostgreSQL volume predates the MVP listing migration.
func (s *PostgresStore) ensureMVPListingSchema(ctx context.Context) error {
	statements := []string{
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS seller_username TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS seller_profile_url TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS seller_avatar_url TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS seller_rating NUMERIC(8, 3)`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS seller_review_count INTEGER`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS country TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS image_urls JSONB NOT NULL DEFAULT '[]'::jsonb`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS published_at TIMESTAMPTZ`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ`,
		`CREATE TABLE IF NOT EXISTS notification_deliveries (id BIGSERIAL PRIMARY KEY, listing_id BIGINT NOT NULL REFERENCES listings(id) ON DELETE CASCADE, channel TEXT NOT NULL, attempted_at TIMESTAMPTZ NOT NULL DEFAULT now(), succeeded BOOLEAN NOT NULL, error TEXT NOT NULL DEFAULT '')`,
		`CREATE INDEX IF NOT EXISTS listings_first_seen_idx ON listings (first_seen_at DESC)`,
		`CREATE INDEX IF NOT EXISTS notification_deliveries_attempted_idx ON notification_deliveries (attempted_at DESC)`,
		`CREATE TABLE IF NOT EXISTS history_snapshots (id BIGSERIAL PRIMARY KEY, source TEXT NOT NULL, synced_at TIMESTAMPTZ NOT NULL DEFAULT now(), accepted_rows INTEGER NOT NULL, rejected_rows INTEGER NOT NULL DEFAULT 0)`,
		`ALTER TABLE sales_history ADD COLUMN IF NOT EXISTS snapshot_id BIGINT REFERENCES history_snapshots(id)`,
		`ALTER TABLE sales_history ADD COLUMN IF NOT EXISTS source_row INTEGER`,
		`ALTER TABLE sales_history ADD COLUMN IF NOT EXISTS original_model TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sales_history ADD COLUMN IF NOT EXISTS brand TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sales_history ADD COLUMN IF NOT EXISTS normalized_model TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sales_history ADD COLUMN IF NOT EXISTS normalized_size TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sales_history ADD COLUMN IF NOT EXISTS normalized_condition TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sales_history ADD COLUMN IF NOT EXISTS days_to_sell INTEGER`,
		`CREATE INDEX IF NOT EXISTS sales_history_snapshot_idx ON sales_history (snapshot_id, normalized_model, normalized_size)`,
		`CREATE TABLE IF NOT EXISTS history_source (id INTEGER PRIMARY KEY CHECK (id = 1), spreadsheet_url TEXT NOT NULL DEFAULT '', worksheet TEXT NOT NULL DEFAULT 'Sales', enabled BOOLEAN NOT NULL DEFAULT FALSE, updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), last_sync_at TIMESTAMPTZ, last_error TEXT NOT NULL DEFAULT '', accepted_rows INTEGER NOT NULL DEFAULT 0, rejected_rows INTEGER NOT NULL DEFAULT 0)`,
		`INSERT INTO history_source (id) VALUES (1) ON CONFLICT (id) DO NOTHING`,
	}
	for _, statement := range statements {
		if _, err := s.db.Exec(ctx, statement); err != nil {
			return fmt.Errorf("ensure MVP listing schema: %w", err)
		}
	}
	return nil
}
