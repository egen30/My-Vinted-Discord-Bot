CREATE TABLE IF NOT EXISTS history_snapshots (
    id BIGSERIAL PRIMARY KEY,
    source TEXT NOT NULL,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    accepted_rows INTEGER NOT NULL,
    rejected_rows INTEGER NOT NULL DEFAULT 0
);

ALTER TABLE sales_history ADD COLUMN IF NOT EXISTS snapshot_id BIGINT REFERENCES history_snapshots(id);
ALTER TABLE sales_history ADD COLUMN IF NOT EXISTS source_row INTEGER;
ALTER TABLE sales_history ADD COLUMN IF NOT EXISTS original_model TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_history ADD COLUMN IF NOT EXISTS brand TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_history ADD COLUMN IF NOT EXISTS normalized_model TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_history ADD COLUMN IF NOT EXISTS normalized_size TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_history ADD COLUMN IF NOT EXISTS normalized_condition TEXT NOT NULL DEFAULT '';
ALTER TABLE sales_history ADD COLUMN IF NOT EXISTS days_to_sell INTEGER;

CREATE INDEX IF NOT EXISTS sales_history_snapshot_idx ON sales_history (snapshot_id, normalized_model, normalized_size);
