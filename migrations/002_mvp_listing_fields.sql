ALTER TABLE listings ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';
ALTER TABLE listings ADD COLUMN IF NOT EXISTS seller_username TEXT NOT NULL DEFAULT '';
ALTER TABLE listings ADD COLUMN IF NOT EXISTS seller_profile_url TEXT NOT NULL DEFAULT '';
ALTER TABLE listings ADD COLUMN IF NOT EXISTS seller_avatar_url TEXT NOT NULL DEFAULT '';
ALTER TABLE listings ADD COLUMN IF NOT EXISTS seller_rating NUMERIC(8, 3);
ALTER TABLE listings ADD COLUMN IF NOT EXISTS seller_review_count INTEGER;
ALTER TABLE listings ADD COLUMN IF NOT EXISTS country TEXT NOT NULL DEFAULT '';
ALTER TABLE listings ADD COLUMN IF NOT EXISTS image_urls JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE listings ADD COLUMN IF NOT EXISTS published_at TIMESTAMPTZ;
ALTER TABLE listings ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS notification_deliveries (
    id BIGSERIAL PRIMARY KEY,
    listing_id BIGINT NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    channel TEXT NOT NULL,
    attempted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    succeeded BOOLEAN NOT NULL,
    error TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS listings_first_seen_idx ON listings (first_seen_at DESC);
CREATE INDEX IF NOT EXISTS notification_deliveries_attempted_idx ON notification_deliveries (attempted_at DESC);
