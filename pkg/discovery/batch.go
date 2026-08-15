// Package discovery contains framework-independent listing collection logic.
package discovery

import (
	"sync"

	"github.com/2spy/vinted-discord-bot/pkg/models"
)

type Listing struct {
	Item      models.Item
	SearchIDs []int64
}

// Batch combines overlapping search results by marketplace identity while
// retaining every search that discovered the listing.
type Batch struct {
	mu       sync.Mutex
	listings map[string]*Listing
}

func NewBatch() *Batch { return &Batch{listings: make(map[string]*Listing)} }

func (b *Batch) Add(search models.Search, items []models.Item) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, item := range items {
		listing, ok := b.listings[item.Platform+":"+item.ID]
		if !ok {
			item.FoundBy = []string{search.Name}
			listing = &Listing{Item: item}
			b.listings[item.Platform+":"+item.ID] = listing
		}
		if !contains(listing.SearchIDs, search.ID) {
			listing.SearchIDs = append(listing.SearchIDs, search.ID)
			if len(listing.Item.FoundBy) == 0 || listing.Item.FoundBy[len(listing.Item.FoundBy)-1] != search.Name {
				listing.Item.FoundBy = append(listing.Item.FoundBy, search.Name)
			}
		}
	}
}

func (b *Batch) Listings() []Listing {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make([]Listing, 0, len(b.listings))
	for _, listing := range b.listings {
		result = append(result, *listing)
	}
	return result
}

func contains(ids []int64, wanted int64) bool {
	for _, id := range ids {
		if id == wanted {
			return true
		}
	}
	return false
}
