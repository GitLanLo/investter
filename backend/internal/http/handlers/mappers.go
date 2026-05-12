package handlers

import (
	"context"
	"time"

	"invest/backend/internal/app"
	"invest/backend/internal/domain"
	"invest/backend/internal/service"
)

func ToAssetDTO(asset domain.Asset) AssetDTO {
	out := AssetDTO{
		ID:                asset.ID,
		Ticker:            asset.Ticker,
		Name:              asset.Name,
		Exchange:          asset.Exchange,
		Timeframe:         asset.Timeframe,
		IsActive:          asset.IsActive,
		Figi:              asset.Figi,
		InstrumentUID:     asset.InstrumentUID,
		ClassCode:         asset.ClassCode,
		InstrumentType:    asset.InstrumentType,
		Lot:               asset.Lot,
		Currency:          asset.Currency,
		APITradeAvailable: asset.APITradeAvailable,
		ModelSupported:    asset.ModelSupported,
	}
	if asset.First1MinCandleDate != nil {
		out.First1MinCandleDate = asset.First1MinCandleDate.UTC().Format(time.RFC3339)
	}
	if asset.First1DayCandleDate != nil {
		out.First1DayCandleDate = asset.First1DayCandleDate.UTC().Format(time.RFC3339)
	}
	return out
}

func ToInstrumentDTO(inst domain.TinkoffInstrument) InstrumentDTO {
	dto := InstrumentDTO{
		UID:               inst.UID,
		Figi:              inst.Figi,
		Ticker:            inst.Ticker,
		ClassCode:         inst.ClassCode,
		Isin:              inst.Isin,
		Lot:               inst.Lot,
		Currency:          inst.Currency,
		Name:              inst.Name,
		Exchange:          inst.Exchange,
		InstrumentType:    inst.InstrumentType,
		APITradeAvailable: inst.APITradeAvailable,
	}
	if inst.First1MinCandleDate != nil {
		dto.First1MinCandleDate = inst.First1MinCandleDate.UTC().Format(time.RFC3339)
	}
	if inst.First1DayCandleDate != nil {
		dto.First1DayCandleDate = inst.First1DayCandleDate.UTC().Format(time.RFC3339)
	}
	return dto
}

func ToCandleDTO(c domain.Candle) CandleDTO {
	return CandleDTO{
		Timestamp: c.Timestamp.UTC().Format(time.RFC3339),
		Open:      c.Open,
		High:      c.High,
		Low:       c.Low,
		Close:     c.Close,
		Volume:    c.Volume,
	}
}

func ToFactorDTO(f domain.FactorBar) FactorDTO {
	return FactorDTO{
		Factor:    f.Factor,
		Timestamp: f.Timestamp.UTC().Format(time.RFC3339),
		Close:     f.Close,
	}
}

func ToWatchlistItemDTO(ctx context.Context, container app.Container, item domain.WatchlistItem) WatchlistItemDTO {
	out := WatchlistItemDTO{
		AssetID:  item.AssetID,
		Position: item.Position,
	}
	if container.Services.Assets == nil {
		return out
	}
	asset, err := container.Services.Assets.GetByID(ctx, item.AssetID)
	if err != nil {
		return out
	}
	dto := ToAssetDTO(asset)
	out.Asset = &dto
	return out
}

func ToSignalDTO(run domain.SignalRun) SignalDTO {
	var policyDTO *SignalPolicyDTO
	if run.Policy != nil {
		policyDTO = &SignalPolicyDTO{
			PolicyStatus:      run.Policy.PolicyStatus,
			ModelName:         run.Policy.ModelName,
			ScenarioName:      run.Policy.ScenarioName,
			CalibrationMethod: run.Policy.CalibrationMethod,
			Threshold:         run.Policy.Threshold,
			DatasetVersion:    run.Policy.DatasetVersion,
		}
	}

	probs := make(map[string]float64)
	probs["up"] = run.ClassProbabilities.Up
	probs["down"] = run.ClassProbabilities.Down
	probs["no_trade"] = run.ClassProbabilities.NoTrade

	var idPtr *int64
	if run.ID != 0 {
		idCopy := run.ID
		idPtr = &idCopy
	}

	return SignalDTO{
		ID:                 idPtr,
		AssetID:            run.AssetID,
		AsOfTime:           run.AsOfTime.UTC().Format(time.RFC3339),
		SignalState:        run.SignalState,
		SignalDirection:    run.SignalDirection,
		SignalProbability:  run.SignalProbability,
		ClassProbabilities: probs,
		Threshold:          run.Threshold,
		Timeframe:          run.Timeframe,
		ModelVersion:       run.ModelVersion,
		HorizonBars:        run.HorizonBars,
		Policy:             policyDTO,
	}
}

func ToSignalEventDTO(e domain.SignalEvent) SignalEventDTO {
	return SignalEventDTO{
		ID:             e.ID,
		SignalRunID:    e.SignalRunID,
		EventType:      e.EventType,
		ModelVersion:   e.ModelVersion,
		Ticker:         e.Ticker,
		IdempotencyKey: e.IdempotencyKey,
		Payload:        e.Payload,
		CreatedAt:      e.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func ToNotificationRuleDTO(r domain.NotificationRule) NotificationRuleDTO {
	out := NotificationRuleDTO{
		ID:                 r.ID,
		Ticker:             r.Ticker,
		EventType:          r.EventType,
		TargetIndicator:    r.TargetIndicator,
		Operator:           r.Operator,
		Threshold:          r.Threshold,
		SecondaryThreshold: r.SecondaryThreshold,
		Severity:           r.Severity,
		Direction:          r.Direction,
		ModelVersion:       r.ModelVersion,
		IsEnabled:          r.IsEnabled,
		CooldownMinutes:    r.CooldownMinutes,
		TriggerMode:        r.TriggerMode,
		DeliveryChannels:   r.DeliveryChannels,
	}
	if r.ExpiresAt != nil {
		out.ExpiresAt = r.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return out
}

func ToJobRunDTO(item domain.JobRun) JobRunDTO {
	out := JobRunDTO{
		ID:        item.ID,
		JobType:   item.JobType,
		Status:    item.Status,
		StartedAt: item.StartedAt.UTC().Format(time.RFC3339),
		Payload:   item.Payload,
	}
	if !item.FinishedAt.IsZero() {
		out.FinishedAt = item.FinishedAt.UTC().Format(time.RFC3339)
	}
	if item.ErrorMessage != "" {
		out.ErrorMessage = item.ErrorMessage
	}
	return out
}

func ToResearchOverviewDTO(item service.ResearchArtifactsOverview) MLResearchOverviewResponse {
	return MLResearchOverviewResponse{
		GeneratedAt:        formatOptionalTime(item.GeneratedAt),
		DatasetManifest:    item.DatasetManifest,
		ResearchSummary:    item.ResearchSummary,
		CalibrationSummary: item.CalibrationSummary,
		GridMatrix:         item.GridMatrix,
		SourcePaths:        item.SourcePaths,
		Warnings:           item.Warnings,
	}
}

func ToResearchDocumentDTO(item service.ResearchArtifactDocument) MLResearchDocumentDTO {
	return MLResearchDocumentDTO{
		Key:         item.Key,
		Title:       item.Title,
		Path:        item.Path,
		ContentType: item.ContentType,
		Content:     item.Content,
	}
}

func ToProductionPolicyDTO(item service.ProductionPolicySnapshot) MLProductionPolicyResponse {
	return MLProductionPolicyResponse{
		GeneratedAt:       formatOptionalTime(item.GeneratedAt),
		PolicyStatus:      item.PolicyStatus,
		ModelName:         item.ModelName,
		ModelVersion:      item.ModelVersion,
		ScenarioName:      item.ScenarioName,
		CalibrationMethod: item.CalibrationMethod,
		Threshold:         item.Threshold,
		Timeframe:         item.Timeframe,
		HorizonBars:       item.HorizonBars,
		DatasetVersion:    item.DatasetVersion,
		FeatureSchema:     item.FeatureSchema,
		TrainRows:         item.TrainRows,
		ValidationRows:    item.ValidationRows,
		TestRows:          item.TestRows,
		Validation: MLProductionPolicyMetricsDTO{
			ActionableF1:  item.Validation.ActionableF1,
			Precision:     item.Validation.Precision,
			Coverage:      item.Validation.Coverage,
			ActionableECE: item.Validation.ActionableECE,
		},
		Test: MLProductionPolicyMetricsDTO{
			ActionableF1:  item.Test.ActionableF1,
			Precision:     item.Test.Precision,
			Coverage:      item.Test.Coverage,
			ActionableECE: item.Test.ActionableECE,
		},
		SourcePaths: item.SourcePaths,
		Warnings:    item.Warnings,
	}
}

func ToModelManifestDTO(m domain.ModelManifest) MLModelManifestDTO {
	return MLModelManifestDTO{
		ModelVersion:              m.ModelVersion,
		ModelType:                 m.ModelType,
		ModelFamily:               m.ModelFamily,
		Classes:                   m.Classes,
		Timeframe:                 m.Timeframe,
		HorizonBars:               m.HorizonBars,
		FeatureSchemaVersion:      m.FeatureSchemaVersion,
		FeatureColumns:            m.FeatureColumns,
		NormalizationArtifactPath: m.NormalizationArtifactPath,
		ExportFormat:              m.ExportFormat,
		ModelArtifactPath:         m.ModelArtifactPath,
		ArtifactSHA256:            m.ArtifactSHA256,
		Metrics:                   m.Metrics,
		DecisionThreshold:         m.DecisionThreshold,
		Calibration:               m.Calibration,
		CreatedAt:                 m.CreatedAt.UTC().Format(time.RFC3339),
		SourceDatasetVersion:      m.SourceDatasetVersion,
		InputWindowBars:           m.InputWindowBars,
		InputTensorShape:          m.InputTensorShape,
		RuntimeStatus:             m.RuntimeStatus,
	}
}

func ToPolicyShadowSummaryDTO(summary domain.PolicyShadowSummary) MLPolicyShadowSummaryDTO {
	return MLPolicyShadowSummaryDTO{
		ValidationRunID:   summary.ValidationRunID,
		DecisionState:     summary.DecisionState,
		ModelName:         summary.ModelName,
		CalibrationMethod: summary.CalibrationMethod,
		Threshold:         summary.Threshold,
		DatasetVersion:    summary.DatasetVersion,
		SignalsTotal:      summary.SignalsTotal,
		ActionableSignals: summary.ActionableSignals,
		NoTradeSignals:    summary.NoTradeSignals,
		UpSignals:         summary.UpSignals,
		DownSignals:       summary.DownSignals,
		ObservedCoverage:  summary.ObservedCoverage,
		FirstSignalAt:     formatOptionalTime(summary.FirstSignalAt),
		LastSignalAt:      formatOptionalTime(summary.LastSignalAt),
	}
}

func ToPolicyOutcomeSummaryDTO(summary domain.PolicyOutcomeSummary) MLPolicyOutcomeSummaryDTO {
	blockers := make([]PolicyPromotionBlockerDTO, 0, len(summary.PromotionBlockers))
	for _, blocker := range summary.PromotionBlockers {
		blockers = append(blockers, PolicyPromotionBlockerDTO{
			Code:    blocker.Code,
			Message: blocker.Message,
		})
	}
	return MLPolicyOutcomeSummaryDTO{
		ValidationRunID:        summary.ValidationRunID,
		DecisionState:          summary.DecisionState,
		ModelName:              summary.ModelName,
		CalibrationMethod:      summary.CalibrationMethod,
		Threshold:              summary.Threshold,
		DatasetVersion:         summary.DatasetVersion,
		SignalsTotal:           summary.SignalsTotal,
		ActionableSignals:      summary.ActionableSignals,
		MaturedSignals:         summary.MaturedSignals,
		PendingSignals:         summary.PendingSignals,
		OverduePendingSignals:  summary.OverduePendingSignals,
		HitSignals:             summary.HitSignals,
		MissSignals:            summary.MissSignals,
		RealizedPrecision:      summary.RealizedPrecision,
		AverageReturnPct:       summary.AverageReturnPct,
		AverageActionReturnPct: summary.AverageActionReturnPct,
		LastSignalAt:           formatOptionalTime(summary.LastSignalAt),
		FirstMaturedAt:         formatOptionalTime(summary.FirstMaturedAt),
		LastMaturedAt:          formatOptionalTime(summary.LastMaturedAt),
		CanPromote:             summary.CanPromote,
		PromotionBlockers:      blockers,
	}
}

func ToPolicyOutcomeRecordDTO(record domain.PolicyOutcomeRecord) MLPolicyOutcomeRecordDTO {
	return MLPolicyOutcomeRecordDTO{
		SignalRunID:       record.SignalRunID,
		AssetID:           record.AssetID,
		AsOfTime:          formatOptionalTime(record.AsOfTime),
		SignalState:       record.SignalState,
		SignalDirection:   record.SignalDirection,
		SignalProbability: record.SignalProbability,
		Timeframe:         record.Timeframe,
		HorizonBars:       record.HorizonBars,
		MaturedAt:         formatOptionalTime(record.MaturedAt),
		EntryPrice:        record.EntryPrice,
		ExitPrice:         record.ExitPrice,
		RawReturnPct:      record.RawReturnPct,
		ActionReturnPct:   record.ActionReturnPct,
		IsHit:             record.IsHit,
	}
}

func ToPolicyValidationRunDTO(run domain.PolicyValidationRun) MLPolicyValidationRunDTO {
	return MLPolicyValidationRunDTO{
		ID:                run.ID,
		PolicyStatus:      run.PolicyStatus,
		ModelName:         run.ModelName,
		ModelVersion:      run.ModelVersion,
		ScenarioName:      run.ScenarioName,
		CalibrationMethod: run.CalibrationMethod,
		Threshold:         run.Threshold,
		DatasetVersion:    run.DatasetVersion,
		Validation: MLProductionPolicyMetricsDTO{
			ActionableF1:  run.Validation.ActionableF1,
			Precision:     run.Validation.Precision,
			Coverage:      run.Validation.Coverage,
			ActionableECE: run.Validation.ActionableECE,
		},
		Test: MLProductionPolicyMetricsDTO{
			ActionableF1:  run.Test.ActionableF1,
			Precision:     run.Test.Precision,
			Coverage:      run.Test.Coverage,
			ActionableECE: run.Test.ActionableECE,
		},
		DecisionState: run.DecisionState,
		Notes:         run.Notes,
		CreatedAt:     run.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func ToNotificationEventDTO(e domain.NotificationEvent) NotificationEventDTO {
	return NotificationEventDTO{
		ID:               e.ID,
		RuleID:           e.RuleID,
		SignalEventID:    e.SignalEventID,
		EventType:        e.EventType,
		Severity:         e.Severity,
		ModelVersion:     e.ModelVersion,
		Ticker:           e.Ticker,
		Message:          e.Message,
		Payload:          e.Payload,
		DeliveryStatus:   e.DeliveryStatus,
		DeliveryAttempts: e.DeliveryAttempts,
		CreatedAt:        e.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
