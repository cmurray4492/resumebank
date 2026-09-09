package background

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunEvery_CallsTaskOnEachTickUntilCancelled(t *testing.T) {
	var calls int32
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		RunEvery(ctx, 10*time.Millisecond, "test task", func(context.Context) error {
			atomic.AddInt32(&calls, 1)
			return nil
		})
		close(done)
	}()

	time.Sleep(55 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("RunEvery did not return after context cancellation")
	}

	got := atomic.LoadInt32(&calls)
	if got < 2 {
		t.Errorf("expected at least 2 calls in ~55ms with a 10ms interval, got %d", got)
	}
}

func TestRunEvery_DoesNotCallTaskImmediately(t *testing.T) {
	var calls int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go RunEvery(ctx, time.Hour, "test task", func(context.Context) error {
		atomic.AddInt32(&calls, 1)
		return nil
	})

	time.Sleep(20 * time.Millisecond)
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Errorf("expected no calls before the first tick, got %d", got)
	}
}

func TestRunEvery_ReturnsPromptlyOnCancelDuringWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		RunEvery(ctx, time.Hour, "test task", func(context.Context) error { return nil })
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("RunEvery did not return promptly for an already-cancelled context")
	}
}
