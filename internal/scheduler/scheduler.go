// Package scheduler runs independent searches with bounded concurrency.
package scheduler

import (
	"context"
	"fmt"
	"sync"

	"github.com/2spy/vinted-discord-bot/pkg/models"
)

type RunFunc func(context.Context, models.Search) error

func RunOnce(ctx context.Context, searches []models.Search, concurrency int, run RunFunc) error {
	if concurrency < 1 {
		concurrency = 1
	}
	if len(searches) == 0 {
		return nil
	}
	sem := make(chan struct{}, concurrency)
	var wait sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	for _, search := range searches {
		search := search
		wait.Add(1)
		go func() {
			defer wait.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			if err := run(ctx, search); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("search %q: %w", search.Name, err)
				}
				mu.Unlock()
			}
		}()
	}
	wait.Wait()
	return firstErr
}
