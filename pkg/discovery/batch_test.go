package discovery

import (
	"testing"

	"github.com/2spy/vinted-discord-bot/pkg/models"
)

func TestBatchDeduplicatesByMarketplaceIdentityAndRetainsSources(t *testing.T) {
	batch := NewBatch()
	item := models.Item{ID: "42", Platform: "vinted", Title: "P-6000"}
	batch.Add(models.Search{ID: 1, Name: "Germany"}, []models.Item{item})
	batch.Add(models.Search{ID: 2, Name: "Broad"}, []models.Item{item})
	batch.Add(models.Search{ID: 1, Name: "Germany"}, []models.Item{item})

	listings := batch.Listings()
	if len(listings) != 1 {
		t.Fatalf("got %d listings, want 1", len(listings))
	}
	if len(listings[0].SearchIDs) != 2 {
		t.Fatalf("got %d search IDs, want 2", len(listings[0].SearchIDs))
	}
	if len(listings[0].Item.FoundBy) != 2 {
		t.Fatalf("got %d source names, want 2", len(listings[0].Item.FoundBy))
	}
}
