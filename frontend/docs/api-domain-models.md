# Frontend Domain Models and API Mapping

## Asset list

| Backend DTO field | Frontend field | Notes |
| --- | --- | --- |
| `id` | `id` | canonical asset id |
| `ticker` | `ticker` | shown in compact cards |
| `name` | `name` | human-readable title |
| `exchange` | `venue` | renamed for UI semantics |
| `timeframe` | `timeframe` | directly rendered |
| `is_active` | `active` | boolean status badge |

## Signal card

| Backend DTO field | Frontend field | Notes |
| --- | --- | --- |
| `id` | `id` | list key |
| `asset_id` | `assetId` | joins with asset cards |
| `signal_direction` | `direction` | `up/down/flat` UI badge |
| `signal_state` | `state` | actionable/watch/no-trade semantics |
| `signal_probability` | `probability` | number in `[0,1]` |
| `threshold` | `threshold` | UI comparison |
| `as_of_time` | `asOfTime` | UTC ISO string |
| `timeframe` | `timeframe` | displayed in chip |
| `model_version` | `modelVersion` | inspection/debug panel |
| `class_probabilities` | `classProbabilities` | probability stack |
| `policy` | `policy` | optional Sprint 3 policy snapshot attached by backend |

## Mapping decisions

- backend DTOs не тащатся в UI напрямую; сначала проходит тонкий mapping layer в `src/lib/api.ts`;
- UI-модели intentionally flatter, чем backend DTO, чтобы не смешивать API concerns и presentation concerns;
- mock data использует те же frontend domain models, поэтому shell одинаково работает с live API и fallback path.

## Policy validation run

| Backend DTO field | Frontend field | Notes |
| --- | --- | --- |
| `id` | `id` | persisted validation run id |
| `policy_status` | `policyStatus` | normalized policy state at snapshot time |
| `model_name` | `modelName` | model selected by production policy |
| `scenario_name` | `scenarioName` | feature scenario selected by research gate |
| `calibration_method` | `calibrationMethod` | calibration selected by policy gate |
| `threshold` | `threshold` | action threshold stored with snapshot |
| `dataset_version` | `datasetVersion` | dataset that produced the policy |
| `validation` | `validation` | mapped into `CandidateMetrics` |
| `test` | `test` | mapped into `CandidateMetrics` |
| `decision_state` | `decisionState` | candidate/shadow/promoted/blocked |
| `notes` | `notes` | operator note |
| `created_at` | `createdAt` | UTC ISO string |

Transition API:

- `PATCH /ml/policy/validation-runs/{id}` updates `decision_state` and `notes`.
- Allowed frontend actions map to `candidate`, `shadow_live`, `promoted`, `blocked`.

## Policy shadow summary

| Backend DTO field | Frontend field | Notes |
| --- | --- | --- |
| `validation_run_id` | `validationRunId` | validation run selected for shadow monitoring |
| `decision_state` | `decisionState` | current operator decision |
| `model_name` | `modelName` | persisted signal policy model |
| `calibration_method` | `calibrationMethod` | persisted signal policy calibration method |
| `threshold` | `threshold` | persisted policy threshold |
| `dataset_version` | `datasetVersion` | persisted signal policy dataset |
| `signals_total` | `signalsTotal` | signal rows matched by persisted policy metadata |
| `actionable_signals` | `actionableSignals` | actionable rows in matched signal set |
| `no_trade_signals` | `noTradeSignals` | no-trade rows in matched signal set |
| `up_signals` | `upSignals` | up actionable rows |
| `down_signals` | `downSignals` | down actionable rows |
| `observed_coverage` | `observedCoverage` | actionable / total |
