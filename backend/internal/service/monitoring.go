package service

import (
	"context"
	"fmt"
	"time"

	"invest/backend/internal/domain"
	"invest/backend/internal/repository"
)

type MonitoringService struct {
	assetRepo    repository.AssetRepository
	modelRepo    repository.ModelRegistryRepository
	signalRepo   repository.SignalRunRepository
	policyRepo   repository.PolicyValidationRunRepository
	notification repository.NotificationRepository
	jobRepo      repository.JobRunRepository
	validation   *PolicyValidationService
}

func NewMonitoringService(
	assetRepo repository.AssetRepository,
	modelRepo repository.ModelRegistryRepository,
	signalRepo repository.SignalRunRepository,
	policyRepo repository.PolicyValidationRunRepository,
	notification repository.NotificationRepository,
	jobRepo repository.JobRunRepository,
	validation *PolicyValidationService,
) *MonitoringService {
	return &MonitoringService{
		assetRepo:    assetRepo,
		modelRepo:    modelRepo,
		signalRepo:   signalRepo,
		policyRepo:   policyRepo,
		notification: notification,
		jobRepo:      jobRepo,
		validation:   validation,
	}
}

type HealthItem struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type SeverityCounts struct {
	Info     int `json:"info"`
	Warning  int `json:"warning"`
	Critical int `json:"critical"`
}

type MonitoringSummary struct {
	GeneratedAt time.Time      `json:"generated_at"`
	Status      string         `json:"status"`
	Items       []HealthItem   `json:"items"`
	Severity    SeverityCounts `json:"severity"`

	Models struct {
		ActiveVersion string `json:"active_version"`
		TotalCount    int    `json:"total_count"`
	} `json:"models"`
	Freshness struct {
		LastSignalAt time.Time `json:"last_signal_at"`
		StaleAssets  int       `json:"stale_assets"`
	} `json:"freshness"`
	Policy struct {
		ActiveRunID       int64   `json:"active_run_id"`
		RealizedPrecision float64 `json:"realized_precision"`
		MaturedSignals    int     `json:"matured_signals"`
	} `json:"policy"`
	Notifications struct {
		RecentWarnings int `json:"recent_warnings"`
		RecentCritical int `json:"recent_critical"`
	} `json:"notifications"`
	Jobs struct {
		FailedLast24h int `json:"failed_last_24h"`
	} `json:"jobs"`
}

func (s *MonitoringService) GetSummary(ctx context.Context) (MonitoringSummary, error) {
	summary := MonitoringSummary{
		GeneratedAt: time.Now().UTC(),
		Status:      "ok",
		Items:       []HealthItem{},
	}

	// 1. Models
	active, err := s.modelRepo.GetActive(ctx)
	if err != nil {
		summary.Items = append(summary.Items, HealthItem{Name: "model_registry", Status: "error", Message: "active model not found or registry error: " + err.Error()})
		summary.Status = "error"
		summary.Severity.Critical++
	} else {
		summary.Models.ActiveVersion = active.ModelVersion
		manifest, err := LoadManifest(active.ManifestPath)
		if err != nil {
			summary.Items = append(summary.Items, HealthItem{Name: "model_manifest", Status: "warn", Message: "failed to load active model manifest: " + err.Error()})
			summary.updateStatus("warn")
			summary.Severity.Warning++
		} else if manifest.RuntimeStatus != RuntimeStatusAvailable {
			summary.Items = append(summary.Items, HealthItem{Name: "model_runtime", Status: "warn", Message: fmt.Sprintf("active model %s is %s", active.ModelVersion, manifest.RuntimeStatus)})
			summary.updateStatus("warn")
			summary.Severity.Warning++
		}
	}
	if allModels, err := s.modelRepo.List(ctx); err == nil {
		summary.Models.TotalCount = len(allModels)
	}

	// 2. Signals & Freshness
	latestSignals, err := s.signalRepo.ListLatest(ctx, 0, 1)
	if err != nil {
		summary.Items = append(summary.Items, HealthItem{Name: "signal_repository", Status: "error", Message: "failed to list latest signals: " + err.Error()})
		summary.updateStatus("error")
		summary.Severity.Critical++
	} else if len(latestSignals) > 0 {
		summary.Freshness.LastSignalAt = latestSignals[0].AsOfTime
		if time.Since(latestSignals[0].AsOfTime) > 24*time.Hour {
			summary.Items = append(summary.Items, HealthItem{Name: "data_freshness", Status: "warn", Message: "no signals generated in last 24h"})
			summary.updateStatus("warn")
			summary.Severity.Warning++
		}
	} else {
		summary.Items = append(summary.Items, HealthItem{Name: "data_freshness", Status: "warn", Message: "no signals found in repository"})
		summary.updateStatus("warn")
		summary.Severity.Warning++
	}

	assets, err := s.assetRepo.List(ctx)
	if err != nil {
		summary.Items = append(summary.Items, HealthItem{Name: "asset_repository", Status: "error", Message: "failed to list assets: " + err.Error()})
		summary.updateStatus("error")
		summary.Severity.Critical++
	} else {
		for _, a := range assets {
			if !a.IsActive {
				continue
			}
			sig, err := s.signalRepo.ListByAsset(ctx, 0, a.ID, 1)
			if err != nil || len(sig) == 0 || time.Since(sig[0].AsOfTime) > 24*time.Hour {
				summary.Freshness.StaleAssets++
			}
		}
		if summary.Freshness.StaleAssets > 0 {
			summary.Items = append(summary.Items, HealthItem{Name: "asset_freshness", Status: "warn", Message: fmt.Sprintf("%d active assets are stale (no signals in last 24h)", summary.Freshness.StaleAssets)})
			summary.updateStatus("warn")
			summary.Severity.Warning++
		}
	}

	// 3. Policy
	if s.validation != nil {
		outcomeSummary, err := s.validation.LoadOutcomeSummary(ctx, 1000)
		if err != nil {
			summary.Items = append(summary.Items, HealthItem{Name: "policy_validation", Status: "warn", Message: "failed to load outcome summary: " + err.Error()})
			summary.updateStatus("warn")
			summary.Severity.Warning++
		} else {
			summary.Policy.ActiveRunID = outcomeSummary.ValidationRunID
			summary.Policy.RealizedPrecision = outcomeSummary.RealizedPrecision
			summary.Policy.MaturedSignals = outcomeSummary.MaturedSignals

			if outcomeSummary.MaturedSignals >= 20 && outcomeSummary.RealizedPrecision < 0.35 {
				summary.Items = append(summary.Items, HealthItem{Name: "policy_performance", Status: "warn", Message: fmt.Sprintf("low realized precision: %.3f (threshold 0.35)", outcomeSummary.RealizedPrecision)})
				summary.updateStatus("warn")
				summary.Severity.Warning++
			}
		}
	}

	// 4. Notifications
	if s.notification != nil {
		recentEvents, err := s.notification.ListLatestEvents(ctx, 0, 100)
		if err != nil {
			summary.Items = append(summary.Items, HealthItem{Name: "notification_repository", Status: "warn", Message: "failed to list notification events: " + err.Error()})
			summary.updateStatus("warn")
			summary.Severity.Warning++
		} else {
			for _, e := range recentEvents {
				if time.Since(e.CreatedAt) < 24*time.Hour {
					if e.Severity == domain.SeverityWarning {
						summary.Notifications.RecentWarnings++
					} else if e.Severity == domain.SeverityCritical {
						summary.Notifications.RecentCritical++
					}
				}
			}
			if summary.Notifications.RecentCritical > 0 {
				summary.Items = append(summary.Items, HealthItem{Name: "notifications", Status: "error", Message: fmt.Sprintf("%d critical notification events in last 24h", summary.Notifications.RecentCritical)})
				summary.updateStatus("error")
				summary.Severity.Critical++
			} else if summary.Notifications.RecentWarnings > 0 {
				summary.Items = append(summary.Items, HealthItem{Name: "notifications", Status: "warn", Message: fmt.Sprintf("%d warning notification events in last 24h", summary.Notifications.RecentWarnings)})
				summary.updateStatus("warn")
				summary.Severity.Warning++
			}
		}
	}

	// 5. Jobs
	if s.jobRepo != nil {
		allJobs, err := s.jobRepo.ListLatest(ctx, 100)
		if err != nil {
			summary.Items = append(summary.Items, HealthItem{Name: "job_repository", Status: "warn", Message: "failed to list jobs: " + err.Error()})
			summary.updateStatus("warn")
			summary.Severity.Warning++
		} else {
			for _, j := range allJobs {
				if j.Status == domain.JobStatusFailed && time.Since(j.StartedAt) < 24*time.Hour {
					summary.Jobs.FailedLast24h++
				}
			}
			if summary.Jobs.FailedLast24h > 0 {
				summary.Items = append(summary.Items, HealthItem{Name: "job_execution", Status: "error", Message: fmt.Sprintf("%d jobs failed in last 24h", summary.Jobs.FailedLast24h)})
				summary.updateStatus("error")
				summary.Severity.Critical++
			}
		}
	}

	return summary, nil
}

func (s *MonitoringSummary) updateStatus(newStatus string) {
	if s.Status == "error" {
		return
	}
	if newStatus == "error" {
		s.Status = "error"
	} else if newStatus == "warn" && s.Status == "ok" {
		s.Status = "warn"
	}
}
