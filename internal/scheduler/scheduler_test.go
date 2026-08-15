package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/2spy/vinted-discord-bot/pkg/models"
)

func TestRunOnceLimitsConcurrency(t *testing.T) {
	var active, maxActive int32
	searches := []models.Search{{Name: "one"}, {Name: "two"}, {Name: "three"}, {Name: "four"}}
	err := RunOnce(context.Background(), searches, 2, func(context.Context, models.Search) error {
		current := atomic.AddInt32(&active, 1)
		for {
			old := atomic.LoadInt32(&maxActive)
			if current <= old || atomic.CompareAndSwapInt32(&maxActive, old, current) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt32(&active, -1)
		return nil
	})
	if err != nil || maxActive > 2 {
		t.Fatalf("unexpected scheduler result: err=%v max=%d", err, maxActive)
	}
}
