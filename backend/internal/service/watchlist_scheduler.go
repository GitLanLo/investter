package service

import (
	"context"
	"log"
	"time"
)

// WatchlistRefreshScheduler runs periodic watchlist data refresh jobs.
type WatchlistRefreshScheduler struct {
	svc      *WatchlistRefreshService
	interval time.Duration
	limit    int
	logger   *log.Logger
}

func NewWatchlistRefreshScheduler(
	svc *WatchlistRefreshService,
	interval time.Duration,
	limit int,
	logger *log.Logger,
) *WatchlistRefreshScheduler {
	return &WatchlistRefreshScheduler{
		svc:      svc,
		interval: interval,
		limit:    limit,
		logger:   logger,
	}
}

func (s *WatchlistRefreshScheduler) Start(ctx context.Context) {
	if s == nil || s.svc == nil || s.interval <= 0 {
		return
	}
	go s.run(ctx)
}

func (s *WatchlistRefreshScheduler) run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOnce(ctx, "tick")
		}
	}
}

func (s *WatchlistRefreshScheduler) runOnce(ctx context.Context, source string) {
	if ctx.Err() != nil {
		return
	}
	if s.logger != nil {
		s.logger.Printf("watchlist refresh scheduler: %s run started", source)
	}
	if _, err := s.svc.RunWatchlistRefresh(ctx, s.limit); err != nil {
		if s.logger != nil {
			s.logger.Printf("watchlist refresh scheduler: %s run failed: %v", source, err)
		}
		return
	}
	if s.logger != nil {
		s.logger.Printf("watchlist refresh scheduler: %s run completed", source)
	}
}

// WatchlistSignalScheduler runs periodic signal generation jobs for watchlist instruments.
type WatchlistSignalScheduler struct {
	svc      *WatchlistRefreshService
	interval time.Duration
	logger   *log.Logger
}

func NewWatchlistSignalScheduler(
	svc *WatchlistRefreshService,
	interval time.Duration,
	logger *log.Logger,
) *WatchlistSignalScheduler {
	return &WatchlistSignalScheduler{
		svc:      svc,
		interval: interval,
		logger:   logger,
	}
}

func (s *WatchlistSignalScheduler) Start(ctx context.Context) {
	if s == nil || s.svc == nil || s.interval <= 0 {
		return
	}
	go s.run(ctx)
}

func (s *WatchlistSignalScheduler) run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOnce(ctx, "tick")
		}
	}
}

func (s *WatchlistSignalScheduler) runOnce(ctx context.Context, source string) {
	if ctx.Err() != nil {
		return
	}
	if s.logger != nil {
		s.logger.Printf("watchlist signal scheduler: %s run started", source)
	}
	if _, err := s.svc.RunWatchlistSignalRefresh(ctx); err != nil {
		if s.logger != nil {
			s.logger.Printf("watchlist signal scheduler: %s run failed: %v", source, err)
		}
		return
	}
	if s.logger != nil {
		s.logger.Printf("watchlist signal scheduler: %s run completed", source)
	}
}
