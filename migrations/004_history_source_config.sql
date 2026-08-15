CREATE TABLE IF NOT EXISTS history_source (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    spreadsheet_url TEXT NOT NULL DEFAULT '',
    worksheet TEXT NOT NULL DEFAULT 'Sales',
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_sync_at TIMESTAMPTZ,
    last_error TEXT NOT NULL DEFAULT '',
    accepted_rows INTEGER NOT NULL DEFAULT 0,
    rejected_rows INTEGER NOT NULL DEFAULT 0
);

INSERT INTO history_source (id) VALUES (1) ON CONFLICT (id) DO NOTHING;
