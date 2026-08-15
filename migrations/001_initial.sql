CREATE TABLE IF NOT EXISTS listings (
    id BIGSERIAL PRIMARY KEY,
    platform TEXT NOT NULL,
    external_id TEXT NOT NULL,
    url TEXT NOT NULL,
    title TEXT NOT NULL,
    brand TEXT,
    size TEXT,
    condition TEXT,
    purchase_price_cents BIGINT NOT NULL,
    currency CHAR(3) NOT NULL,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    raw_payload JSONB,
    UNIQUE (platform, external_id)
);

CREATE TABLE IF NOT EXISTS searches (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    url TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    priority INTEGER NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_attempted_at TIMESTAMPTZ,
    last_successful_at TIMESTAMPTZ,
    last_error TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS listing_searches (
    listing_id BIGINT NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    search_id BIGINT NOT NULL REFERENCES searches(id) ON DELETE CASCADE,
    first_discovered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_discovered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    discovery_count INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (listing_id, search_id)
);

CREATE TABLE IF NOT EXISTS sales_history (
    id BIGSERIAL PRIMARY KEY,
    model TEXT NOT NULL,
    size TEXT,
    condition TEXT,
    purchase_price_cents BIGINT NOT NULL,
    sale_price_cents BIGINT NOT NULL,
    costs_cents BIGINT NOT NULL DEFAULT 0,
    purchased_at DATE,
    sold_at DATE,
    source TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS evaluations (
    id BIGSERIAL PRIMARY KEY,
    listing_id BIGINT NOT NULL REFERENCES listings(id),
    model TEXT,
    size TEXT,
    condition TEXT NOT NULL,
    expected_resale_cents BIGINT,
    applicable_costs_cents BIGINT NOT NULL DEFAULT 0,
    expected_profit_cents BIGINT,
    maximum_purchase_cents BIGINT,
    roi_percent NUMERIC(12, 4),
    qualified BOOLEAN NOT NULL,
    reason TEXT NOT NULL,
    estimator_version TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS listing_status_events (
    id BIGSERIAL PRIMARY KEY,
    listing_id BIGINT NOT NULL REFERENCES listings(id),
    status TEXT NOT NULL,
    actor TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS evaluations_qualified_idx ON evaluations (qualified, created_at DESC);
CREATE INDEX IF NOT EXISTS status_events_listing_idx ON listing_status_events (listing_id, created_at DESC);
