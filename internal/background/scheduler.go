// Package background runs periodic maintenance tasks (sitemap generation,
// search-index refresh) for the life of the server process.
package background

import (
	"context"
	"log"
	"time"
)

// RunEvery calls task once per interval until ctx is cancelled. It does not
// run task immediately on start — callers that want an initial run before
// serving traffic should call task once themselves first.
func RunEvery(ctx context.Context, interval time.Duration, name string, task func(context.Context) error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := task(ctx); err != nil {
				log.Printf("background task %q failed: %v", name, err)
			}
		}
	}
}
