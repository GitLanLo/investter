package service

import (
	"context"
	"log"
	"time"

	"invest/backend/internal/domain"
)

type outcomeJobRunner interface {
	RunOutcomeMaterialization(ctx context.Context, limit int) (domain.JobRun, error)
}

type OutcomeMaterializationScheduler struct {
	jobs       outcomeJobRunner
	interval   time.Duration
	limit      int
	runOnStart bool
	logger     *log.Logger
}

func NewOutcomeMaterializationScheduler(
	jobs *JobService,
	interval time.Duration,
	limit int,
	runOnStart bool,
	logger *log.Logger,
) *OutcomeMaterializationScheduler {
	return &OutcomeMaterializationScheduler{
		jobs:       jobs,
		interval:   interval,
		limit:      limit,
		runOnStart: runOnStart,
		logger:     logger,
	}
}

func (s *OutcomeMaterializationScheduler) Start(ctx context.Context) {
	if s == nil || s.jobs == nil || s.interval <= 0 || s.limit <= 0 {
		return
	}

	go s.run(ctx)
}

func (s *OutcomeMaterializationScheduler) run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	s.runLoop(ctx, ticker.C)
}

func (s *OutcomeMaterializationScheduler) runLoop(ctx context.Context, ticks <-chan time.Time) {
	if s.runOnStart {
		s.runOnce(ctx, "startup")
	}

	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ticks:
			if !ok {
				return
			}
			s.runOnce(ctx, "tick")
		}
	}
}

func (s *OutcomeMaterializationScheduler) runOnce(ctx context.Context, source string) {
	if ctx.Err() != nil {
		return
	}
	if s.logger != nil {
		s.logger.Printf("outcome scheduler: %s run started", source)
	}
	if _, err := s.jobs.RunOutcomeMaterialization(ctx, s.limit); err != nil {
		if s.logger != nil {
			s.logger.Printf("outcome scheduler: %s run failed: %v", source, err)
		}
		return
	}
	if s.logger != nil {
		s.logger.Printf("outcome scheduler: %s run completed", source)
	}
}
