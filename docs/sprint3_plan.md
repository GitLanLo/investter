# Sprint 3 Plan

Status: completed. Closure note: `docs/sprint3_completion.md`.

Sprint 3 фокусируется на production policy и переходе от research cockpit к контролируемому runtime-кандидату.

## Goals

- Нормализовать production-candidate policy поверх Sprint 2 research/calibration artifacts.
- Показать policy quality в GUI без чтения raw JSON пользователем.
- Подготовить backend contract для следующего шага: live validation loop and production-candidate promotion.
- Сохранить воспроизводимую проверку через `make sprint3-check`.

## Initial Scope

- `GET /ml/policy/production` returns model, scenario, calibration method, threshold, dataset metadata and validation/test metrics.
- Frontend Research Board includes a Sprint 3 policy card.
- Smoke-check extends Sprint 2 verification with production policy endpoint.

## Next Slice

- Add live validation run records for candidate policy. Status: implemented.
- Add policy decision states: `candidate`, `shadow_live`, `promoted`, `blocked`. Status: implemented.
- Add operator transition API/UI for validation run states. Status: implemented.
- Persist policy snapshot directly in `signal_runs`. Status: implemented for new signals.
- Add shadow summary over persisted policy signals. Status: implemented.
- Attach policy status to `/analysis/run` responses. Status: implemented and persisted for new `signal_runs`.
- Add GUI comparison between current live signal and selected production policy. Status: implemented.

## Current Data Refresh

- Latest dataset: `mvp_live_20260422`.
- Latest refresh note: `docs/ml/data_refresh_20260422.md`.
- Baseline production-gated candidate after refresh: `logreg_multiclass`.
- Calibration audit after refresh: `calibration_audit_logreg_mvp_live_20260422`.
- Current Sprint 3 policy: `production_candidate`, model `logreg_multiclass`, method `identity`, threshold `0.30`.
- Latest saved validation run: `id=2`, decision state `shadow_live`.
- Promotion blocker: requires accumulated shadow/live validation outcomes before production promotion.

## Exit Criteria

- `make sprint3-check` passes against a running local stack. Status: passed.
- GUI shows policy identity and core reliability metrics. Status: implemented.
- Backend returns a stable production policy contract independent of raw artifact document layout. Status: implemented.
