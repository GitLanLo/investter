# Sprint 9: Promotion, Notifications And Monitoring

Status: completed.

## Objective

Turn Sprint 8 model artifacts and Sprint 4-6 operational loops into an operator-ready production workflow: explicit model activation, promotion/rollback, signal events, notification rules, and monitoring.

Sprint 9 starts from:

- active metadata artifact: `sprint8_gru_1h_h24_w96`;
- neural runtime status: `metadata_only`;
- selected ML configuration: timeframe `1h`, horizon `24`, window `96`;
- existing forward validation/outcome endpoints from Sprint 4;
- existing data refresh and signal jobs from Sprint 6.
- test matrix: `docs/sprint9_test_plan.md`.

## Non-Goals

- Do not add real brokerage order execution.
- Do not make Telegram/email delivery mandatory for closure; mock delivery is enough.
- Do not promote the Sprint 8 GRU automatically if runtime or forward metrics are blocked.
- Do not run full model retraining from the backend.

## Workstream 1: Model Runtime And Activation Contract

Goal:
make model activation explicit and prevent metadata-only artifacts from being used as executable inference.

Tasks:

- Add a model activation endpoint that can switch active model registry entry only after contract checks pass.
- Keep `metadata_only` neural artifacts visible but blocked from inference until runtime support exists.
- Add runtime capability checks for `available`, `metadata_only`, and `unavailable`.
- Add a clear fallback rule: tabular/baseline runtime can continue serving signals when the neural model is metadata-only.
- Add golden input/output fixtures for the active runtime path.

Files:

- `backend/internal/domain/models.go`
- `backend/internal/service/models.go`
- `backend/internal/service/analysis.go`
- `backend/internal/httpserver/router.go`
- `backend/internal/service/models_test.go`
- `backend/internal/service/analysis_test.go`
- `docs/contracts/api_v1.md`
- `docs/contracts/data_model_contracts_v1.md`

Tests:

- Keep `TestLoadManifest_RuntimeStatusMapping` green for `available`, `metadata_only`, and `unavailable`.
- Add activation tests for blocking `metadata_only` models and activating available models.
- Add analysis tests that prove explicit model requests never silently fall back to another model.
- Add router tests for activation blocker payloads.

Suggested endpoints:

- `GET /ml/models`
- `GET /ml/models/active`
- `GET /ml/models/{version}`
- `POST /ml/models/{version}/activate`

Done when:

- Activating a `metadata_only` neural artifact without runtime override returns a blocked response.
- Activating an available model writes the active registry state.
- `POST /analysis/run` never silently substitutes a different model when a specific blocked model is requested.

## Workstream 2: Policy Promotion And Rollback

Goal:
make promotion a controlled state transition tied to forward validation blockers.

Tasks:

- Normalize policy states:
  - `candidate`;
  - `shadow_live`;
  - `approved`;
  - `active`;
  - `blocked`;
  - `archived`.
- Add promotion action service that checks Sprint 4 blockers before promotion.
- Add rollback action that restores the previous active policy/model.
- Persist promotion events with actor, previous state, next state, blockers, and notes.
- Keep promotion state separate from raw research artifacts.

Files:

- `backend/internal/domain/models.go`
- `backend/internal/storage/migrations/006_policy_promotion_log.sql`
- `backend/internal/repository/repository.go`
- `backend/internal/repository/postgres/policy_promotion.go`
- `backend/internal/service/policy_validation.go`
- `backend/internal/service/policy_promotion.go`
- `backend/internal/httpserver/router.go`
- `backend/internal/service/policy_promotion_test.go`
- `backend/internal/httpserver/router_test.go`

Tests:

- Add service tests for approve, promote, blocked promote, archive previous active, rollback, and promotion log persistence.
- Add router tests for approve/promote/rollback endpoints and blocker responses.
- Add repository tests for promotion log ordering and previous-active lookup.

Suggested endpoints:

- `POST /ml/policy/validation-runs/{id}/approve`
- `POST /ml/policy/validation-runs/{id}/promote`
- `POST /ml/policy/rollback`
- `GET /ml/policy/promotion-log`

Done when:

- Promotion fails with explicit blockers when forward evidence is insufficient.
- A successful promotion archives or demotes the previous active policy.
- Rollback restores the previous active policy and records the action.

## Workstream 3: Signal Events

Goal:
create a durable event stream from signal generation for notifications and monitoring.

Tasks:

- Define signal event types:
  - `signal_generated`;
  - `signal_actionable`;
  - `signal_no_trade`;
  - `signal_suppressed`;
  - `notification_triggered`.
- Write signal events when `analysis/run` and scheduled signal jobs create signal runs.
- Add idempotency key by asset, as-of time, model version, and event type.
- Expose recent signal events for UI/debugging.

Files:

- `backend/internal/domain/models.go`
- `backend/internal/storage/migrations/007_signal_events_hardening.sql`
- `backend/internal/repository/repository.go`
- `backend/internal/repository/postgres/signal_events.go`
- `backend/internal/service/analysis.go`
- `backend/internal/service/watchlist_refresh.go`
- `backend/internal/service/signal_events.go`
- `backend/internal/httpserver/router.go`
- `backend/internal/service/signal_events_test.go`

Tests:

- Add service tests for generated/actionable/no-trade/suppressed events.
- Add idempotency tests by asset, as-of time, model version, and event type.
- Add router tests for `GET /signals/events`.
- Add job tests proving scheduled signal runs emit events without duplicates.

Suggested endpoints:

- `GET /signals/events?asset_id=&limit=`

Done when:

- Every newly generated signal has at least one persisted event.
- Re-running a job does not duplicate the same event.
- Signal event history is queryable by the operator UI.

## Workstream 4: Notification Rules And Evaluator

Goal:
let the operator define notification rules and evaluate them deterministically against signal events.

Tasks:

- Implement notification rule domain model:
  - asset/instrument filter;
  - direction filter;
  - probability threshold;
  - cooldown;
  - required policy/model state;
  - enabled flag;
  - mock channel target.
- Add CRUD repository and API endpoints.
- Implement evaluator that consumes signal events and returns triggered notifications.
- Enforce cooldown per rule and asset.
- Persist notification events and delivery attempts.

Files:

- `backend/internal/domain/models.go`
- `backend/internal/storage/migrations/008_notification_rules.sql`
- `backend/internal/repository/repository.go`
- `backend/internal/repository/postgres/notification_rules.go`
- `backend/internal/repository/postgres/notification_events.go`
- `backend/internal/service/notifications.go`
- `backend/internal/httpserver/router.go`
- `backend/internal/service/notifications_test.go`
- `backend/internal/httpserver/router_test.go`
- `docs/contracts/api_v1.md`

Tests:

- Add evaluator tests for direction, threshold, policy state, enabled flag, cooldown, and dedupe.
- Add repository tests for rule CRUD and notification event persistence.
- Add router tests for notification rule CRUD and notification history.
- Add mock delivery tests for delivered and failed statuses.

Suggested endpoints:

- `GET /notifications/rules`
- `POST /notifications/rules`
- `PATCH /notifications/rules/{id}`
- `DELETE /notifications/rules/{id}`
- `GET /notifications/events`

Done when:

- A rule can be created, updated, disabled, and deleted.
- Evaluator triggers only when direction, threshold, policy state, and cooldown all pass.
- Mock delivery status is persisted.

## Workstream 5: Monitoring Summary

Goal:
surface production degradation before the operator trusts stale or decaying signals.

Tasks:

- Add monitoring summary service with:
  - realized precision decay;
  - signal coverage decay;
  - calibration drift proxy;
  - data freshness;
  - factor freshness;
  - job failure counts;
  - notification delivery failures.
- Use existing outcome and job data where possible.
- Add severity levels: `ok`, `warning`, `critical`.
- Add blocker links to the affected model, policy, asset, or job.

Files:

- `backend/internal/service/monitoring.go`
- `backend/internal/httpserver/router.go`
- `backend/internal/service/monitoring_test.go`
- `frontend/src/App.tsx`
- `frontend/src/lib/types.ts`
- `frontend/src/lib/api.ts`
- `frontend/src/lib/mock.ts`

Tests:

- Add service tests for `ok`, `warning`, and `critical` summaries.
- Add stale-data, job-failure, precision-decay, notification-failure, and metadata-only-model cases.
- Add router tests for `GET /monitoring/summary`.

Suggested endpoint:

- `GET /monitoring/summary`

Done when:

- Monitoring summary reports data/model/job/notification health in one response.
- Critical stale-data or job-failure states are visible in API and UI.
- Tests cover at least one `ok`, `warning`, and `critical` response.

## Workstream 6: Frontend Operator Workflow

Goal:
make promotion, notification rules, and monitoring usable from the GUI.

Tasks:

- Add model activation controls to the Model Card.
- Add promotion/rollback actions to Policy Gate.
- Add notification rules panel.
- Add signal events and notification history view.
- Add monitoring status band with drilldowns to jobs, freshness, policy, and model runtime.
- Keep blocked actions disabled with clear backend-provided reasons.

Files:

- `frontend/src/App.tsx`
- `frontend/src/lib/types.ts`
- `frontend/src/lib/api.ts`
- `frontend/src/lib/mock.ts`

Done when:

- Operator can inspect active model/runtime status.
- Operator can define notification rules without editing config.
- Operator can see why promotion or activation is blocked.

Tests:

- Keep `frontend-build` as the required frontend gate unless a test runner is added.
- Add strict API/mock DTO shape coverage for model activation, promotion blockers, notification rules, signal events, and monitoring summary.
- Add smoke assertions that the UI-facing APIs return the fields needed by the new panels.

## Workstream 7: Automation And Smoke Checks

Goal:
make Sprint 9 closure reproducible.

Tasks:

- Add Make targets:
  - `sprint9-check`;
  - `sprint9-smoke`.
- Extend smoke to verify:
  - model active endpoint;
  - promotion blocker response;
  - notification rule CRUD;
  - signal event visibility;
  - monitoring summary;
  - frontend build.
- Keep notification delivery mocked in smoke.

Files:

- `Makefile`
- backend service/router tests
- frontend build checks
- `docs/sprint9_test_plan.md`

Expected checks:

```sh
make ml-test
make backend-test
make frontend-build
make sprint9-check
```

Done when:

- `make sprint9-check` fails if promotion, notification, signal-event, or monitoring contracts regress.
- Smoke validates response shape with `jq`, not only HTTP 200.

Test matrix:

- Detailed test cases and expected file names are tracked in `docs/sprint9_test_plan.md`.
- Checked-in tests should remain green; add workstream-specific tests together with the implementation they guard.
- Short-lived red/green TDD is acceptable locally, but incomplete failing tests should not be committed.

## Implementation Order

1. Keep the baseline Sprint 9 model metadata tests green.
2. Harden model runtime/activation semantics and add activation tests.
3. Add promotion log and rollback service with service/router/repository tests.
4. Add signal events repository/service and wire analysis/jobs with idempotency tests.
5. Add notification rules CRUD and evaluator with mock delivery tests.
6. Add monitoring summary endpoint with severity tests.
7. Add frontend controls and monitoring views with API shape smoke coverage.
8. Add `sprint9-check` and strict smoke assertions.
9. Close Sprint 9 with a completion note and updated contracts.

## Exit Criteria

- A model/policy can be approved, promoted, blocked, archived, and rolled back through API.
- Promotion uses forward-validation blockers instead of manual trust.
- Signal events are persisted and queryable.
- Notification rules can be created and evaluated with cooldown/deduplication.
- Mock notification delivery records status.
- Monitoring summary reports model, data, job, outcome, and notification health.
- Frontend exposes model activation, promotion blockers, notification rules, and monitoring status.
- `make sprint9-check` passes.

## Main Risks

- Native neural runtime may be too large for Sprint 9; activation must still protect metadata-only models.
- Promotion can become confusing if model registry state and policy validation state diverge; keep a single promotion service as the state-transition owner.
- Notification cooldown and idempotency need strict tests to avoid duplicate alerts.
- Monitoring can become noisy; start with a compact summary and explicit drilldowns.
