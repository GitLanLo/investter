# Sprint 9 Test Plan

Status: completed.

This document defines the Sprint 9 test set. Tests should be added with the workstream they protect; checked-in tests should stay green unless a task explicitly starts with a short-lived red/green TDD step.

## Current Baseline Tests

These tests are already present before Sprint 9 implementation work begins:

- `backend/internal/service/models_test.go`
  - `TestLoadManifest_Neural`
  - `TestLoadManifest_NaiveCreatedAt`
  - `TestLoadManifest_RuntimeStatusMapping`
- `backend/internal/httpserver/router_test.go`
  - `TestModelManifestEndpointsExposeSprint9RuntimeState`

They protect the Sprint 9 starting point:

- neural TorchScript artifacts are `metadata_only`;
- available tabular runtimes can be distinguished from metadata-only neural artifacts;
- unknown export formats are `unavailable`;
- `/ml/models/active` and `/ml/models/{version}` expose neural shape and runtime metadata.

## Workstream 1: Model Runtime And Activation

Backend service tests:

- `backend/internal/service/models_test.go`
  - `TestActivateModel_BlocksMetadataOnlyWithoutOverride`
  - `TestActivateModel_AllowsAvailableModel`
  - `TestActivateModel_DemotesPreviousActiveModel`
  - `TestActivateModel_RecordsBlockerReason`
- `backend/internal/service/analysis_test.go`
  - `TestRunAnalysis_SpecificMetadataOnlyModelReturnsBlocked`
  - `TestRunAnalysis_ActiveMetadataOnlyModelDoesNotFallbackSilently`
  - `TestRunAnalysis_AvailableModelUsesManifestFeatureOrder`

Router tests:

- `backend/internal/httpserver/router_test.go`
  - `TestModelActivationEndpointBlocksMetadataOnly`
  - `TestModelActivationEndpointActivatesAvailableModel`
  - `TestModelActivationEndpointReturnsRuntimeBlockers`

Smoke assertions:

- `GET /ml/models/active` returns active model metadata.
- `POST /ml/models/sprint8_gru_1h_h24_w96/activate` returns blocked while runtime is `metadata_only`.

## Workstream 2: Policy Promotion And Rollback

Service tests:

- `backend/internal/service/policy_promotion_test.go`
  - `TestApproveValidationRun_FromCandidate`
  - `TestPromotePolicy_BlockedByForwardValidation`
  - `TestPromotePolicy_SucceedsWhenBlockersClear`
  - `TestPromotePolicy_ArchivesPreviousActive`
  - `TestRollbackPolicy_RestoresPreviousActive`
  - `TestPromotionLog_RecordsActorNotesAndBlockers`

Repository tests:

- `backend/internal/repository/postgres/policy_promotion_test.go`
  - migration applies cleanly;
  - insert/list promotion log preserves ordering;
  - rollback lookup returns the previous active policy.

Router tests:

- `backend/internal/httpserver/router_test.go`
  - `POST /ml/policy/validation-runs/{id}/approve`
  - `POST /ml/policy/validation-runs/{id}/promote`
  - `POST /ml/policy/rollback`
  - `GET /ml/policy/promotion-log`

Smoke assertions:

- promote returns explicit blockers when `can_promote=false`;
- rollback endpoint returns the restored policy id/version.

## Workstream 3: Signal Events

Service tests:

- `backend/internal/service/signal_events_test.go`
  - `TestSignalEventRecordedForGeneratedSignal`
  - `TestSignalEventActionableVsNoTradeClassification`
  - `TestSignalEventSuppressedForBlockedRuntime`
  - `TestSignalEventIdempotencyPreventsDuplicateEvents`
  - `TestWatchlistSignalJobEmitsEvents`

Repository tests:

- `backend/internal/repository/postgres/signal_events_test.go`
  - upsert by idempotency key;
  - list by asset and time descending;
  - limit clamping.

Router tests:

- `backend/internal/httpserver/router_test.go`
  - `GET /signals/events?asset_id=SBER&limit=10`

Smoke assertions:

- running `POST /jobs/signals/run` creates queryable signal events;
- rerunning the same job does not duplicate the same idempotency key.

## Workstream 4: Notification Rules And Evaluator

Service tests:

- `backend/internal/service/notifications_test.go`
  - `TestNotificationEvaluator_MatchesDirectionAndThreshold`
  - `TestNotificationEvaluator_RespectsPolicyStateRequirement`
  - `TestNotificationEvaluator_RespectsCooldown`
  - `TestNotificationEvaluator_DeduplicatesSignalEvent`
  - `TestNotificationEvaluator_IgnoresDisabledRule`
  - `TestMockDelivery_RecordsDeliveredStatus`
  - `TestMockDelivery_RecordsFailedStatus`

Repository tests:

- `backend/internal/repository/postgres/notification_rules_test.go`
  - CRUD round trip;
  - disabled rules are retained but not evaluated;
  - rule list filters by asset.
- `backend/internal/repository/postgres/notification_events_test.go`
  - delivery attempt persistence;
  - event history ordering;
  - duplicate prevention by rule/signal event.

Router tests:

- `backend/internal/httpserver/router_test.go`
  - `GET /notifications/rules`
  - `POST /notifications/rules`
  - `PATCH /notifications/rules/{id}`
  - `DELETE /notifications/rules/{id}`
  - `GET /notifications/events`

Smoke assertions:

- create a mock rule;
- generate a matching signal event;
- evaluator records one notification event;
- immediate rerun is suppressed by cooldown.

## Workstream 5: Monitoring Summary

Service tests:

- `backend/internal/service/monitoring_test.go`
  - `TestMonitoringSummary_OK`
  - `TestMonitoringSummary_WarningForStaleData`
  - `TestMonitoringSummary_CriticalForJobFailures`
  - `TestMonitoringSummary_WarningForPrecisionDecay`
  - `TestMonitoringSummary_ReportsNotificationFailures`
  - `TestMonitoringSummary_ReportsMetadataOnlyActiveModel`

Router tests:

- `backend/internal/httpserver/router_test.go`
  - `GET /monitoring/summary`

Smoke assertions:

- `/monitoring/summary` returns `status`, `items`, and severity counts;
- at least one controlled degraded fixture returns `warning` or `critical`.

## Workstream 6: Frontend Operator Workflow

Current frontend has no dedicated test runner. Sprint 9 frontend verification is:

- `frontend-build` for type-safety;
- strict smoke responses with `jq`;
- mock data shape coverage in `frontend/src/lib/mock.ts`;
- API DTO mapping checks in `frontend/src/lib/api.ts`.

If a frontend test runner is added, add:

- model activation disabled state for `metadata_only`;
- promotion blockers rendering;
- notification rule form validation;
- monitoring severity rendering.

## Workstream 7: Sprint 9 Check

`make sprint9-check` should include:

- `ml-test`;
- `backend-test`;
- `frontend-build`;
- `sprint9-smoke`.

`sprint9-smoke` must validate response shape, not just HTTP 200:

- active model and runtime status;
- activation blocked response;
- promotion blocked response;
- notification rule CRUD;
- signal event visibility;
- monitoring summary severity fields.
