package historysync

import (
	"context"
	"testing"

	"github.com/2spy/vinted-discord-bot/pkg/history"
)

type fakeRows struct{ rows [][]string }

func (f fakeRows) ReadRows(context.Context) ([][]string, error) { return f.rows, nil }

func TestSyncPublishesValidRowsAndDiagnostics(t *testing.T) {
	syncer := New(fakeRows{rows: [][]string{
		{"model", "size", "condition", "purchase_price", "sale_price"},
		{"Nike P-6000", "42", "good", "15", "35"},
		{"Nike P-6000", "42", "good", "bad", "35"},
	}})
	snapshot, err := syncer.Sync(context.Background())
	if err != nil || len(snapshot.Sales) != 1 || len(snapshot.Rejected) != 1 {
		t.Fatalf("unexpected sync: %+v, %v", snapshot, err)
	}
	if snapshot.Sales[0].SaleCents != 3500 {
		t.Fatalf("unexpected sale: %+v", snapshot.Sales[0])
	}
	if len(syncer.Current().Sales) != 1 {
		t.Fatal("expected published snapshot")
	}
	if snapshot.SyncedAt.IsZero() {
		t.Fatal("expected sync timestamp")
	}
	_ = history.Sale{}
}
