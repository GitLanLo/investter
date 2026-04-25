# Sprint 4 Plan

Status: started.

Sprint 4 переводит Sprint 3 policy candidate из простого shadow/live счётчика в forward-validation контур с realised outcomes.

## Goal

- Считать фактический outcome для persisted policy signals после завершения `horizon_bars`.
- Разделять matured и pending shadow/live signals.
- Показать оператору realized precision и action return до promotion.
- Сохранить воспроизводимую проверку через `make sprint4-check`.

## Initial Scope

- `GET /ml/policy/outcomes` returns forward-validation summary for the active `shadow_live` or `promoted` validation run.
- Outcome summary is computed from persisted `signal_runs` and available future candles.
- Directional hit-rate is counted for actionable `up/down` signals only.
- Pending signals remain visible until enough future candles exist.

## Current Slice

- Backend computed outcome summary: implemented.
- API contract for `/ml/policy/outcomes`: implemented.
- Persisted matured outcomes in `signal_outcomes`: implemented.
- Manual materialization endpoint `POST /ml/policy/outcomes`: implemented.
- Promotion blockers from forward validation summary: implemented.
- Tracked materialization job `POST /jobs/outcomes/materialize`: implemented.
- Recurring backend scheduler for outcome materialization: implemented.
- Outcome audit endpoint `GET /ml/policy/outcomes/history`: implemented.
- Promotion blockers for stale validation and overdue pending outcomes: implemented.
- GUI forward outcomes card with matured/pending and realized precision: implemented.
- GUI scheduler status card in Policy Gate: implemented.
- GUI outcome history view in Policy Gate: implemented.
- Smoke-check extension `make sprint4-check`: implemented.

## Next Slice

- Add drift/calibration decay views after outcome persistence is stable.

## Exit Criteria

- `make sprint4-check` passes against a running local stack.
- Backend exposes matured/pending/hit/miss/precision metrics for the active policy validation run.
- GUI shows realized outcome metrics and still blocks automated trading/promotion when forward validation is insufficient.
