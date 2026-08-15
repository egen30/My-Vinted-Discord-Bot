package models

import "time"

type Search struct {
	ID               int64      `json:"id"`
	Name             string     `json:"name"`
	URL              string     `json:"url"`
	Enabled          bool       `json:"enabled"`
	Priority         int        `json:"priority"`
	Notes            string     `json:"notes,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	LastAttemptedAt  *time.Time `json:"last_attempted_at,omitempty"`
	LastSuccessfulAt *time.Time `json:"last_successful_at,omitempty"`
	LastError        string     `json:"last_error,omitempty"`
}
