package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"invest/backend/internal/domain"
)

type testOutcomeJobRunner struct {
	mu     sync.Mutex
	calls  int
	limits []int
}

func (r *testOutcomeJobRunner) RunOutcomeMaterialization(_ context.Context, limit int) (domain.JobRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	r.limits = append(r.limits, limit)
	return domain.JobRun{}, nil
}

func TestOutcomeMaterializationSchedulerRunLoop(t *testing.T) {
	runner := &testOutcomeJobRunner{}
	scheduler := &OutcomeMaterializationScheduler{
		jobs:       runner,
		limit:      77,
		runOnStart: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	ticks := make(chan time.Time, 1)
	done := make(chan struct{})
	go func() {
		scheduler.runLoop(ctx, ticks)
		close(done)
	}()

	ticks <- time.Now()

	deadline := time.After(2 * time.Second)
	for {
		runner.mu.Lock()
		calls := runner.calls
		limits := append([]int(nil), runner.limits...)
		runner.mu.Unlock()

		if calls >= 2 {
			if limits[0] != 77 || limits[1] != 77 {
				t.Fatalf("unexpected scheduler limits: %+v", limits)
			}
			break
		}

		select {
		case <-deadline:
			t.Fatalf("scheduler did not execute expected runs, got %d", calls)
		case <-time.After(10 * time.Millisecond):
		}
	}

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not stop after context cancellation")
	}
}
