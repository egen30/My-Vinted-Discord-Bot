package models

import "time"

// HistorySource is the non-secret Google Sheets configuration managed by the
// admin page. Credentials remain environment-provided and are never persisted.
type HistorySource struct {
	SpreadsheetURL string     `json:"spreadsheet_url"`
	Worksheet      string     `json:"worksheet"`
	Enabled        bool       `json:"enabled"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
	LastSyncAt     *time.Time `json:"last_sync_at,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
	AcceptedRows   int        `json:"accepted_rows"`
	RejectedRows   int        `json:"rejected_rows"`
}
