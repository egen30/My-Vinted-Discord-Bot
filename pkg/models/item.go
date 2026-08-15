package models

import "time"

// Seller contains marketplace-provided seller metadata. Catalog responses may
// omit profile fields, so every field is optional.
type Seller struct {
	Username    string  `json:"username,omitempty"`
	ProfileURL  string  `json:"profile_url,omitempty"`
	AvatarURL   string  `json:"avatar_url,omitempty"`
	Rating      float64 `json:"rating,omitempty"`
	ReviewCount int     `json:"review_count,omitempty"`
	Country     string  `json:"country,omitempty"`
}

type Item struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Brand       string   `json:"brand"`
	Size        string   `json:"size"`
	Condition   string   `json:"condition"`
	Description string   `json:"description"`
	Price       float64  `json:"price"`
	Currency    string   `json:"currency"`
	URL         string   `json:"url"`
	ImageURL    string   `json:"image_url"`
	ImageURLs   []string `json:"image_urls,omitempty"`
	Platform    string   `json:"platform"`
	Seller      Seller   `json:"seller,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	FoundBy     []string `json:"found_by,omitempty"`
}
