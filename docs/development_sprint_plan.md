# Development Sprint Plan

Status: active planning document.

This plan starts from the current repository state: Sprint 1-3 are closed, Sprint 4 is started on forward validation.

## Product Direction

The target product flow is:

1. Operator searches any supported T-Bank Invest instrument.
2. Operator adds it to the local watchlist.
3. The system refreshes market data for watchlist instruments.
4. The ML pipeline produces signals only for instrument classes covered by the trained policy.
5. Shadow/live outcomes are measured before policy promotion.
6. The UI shows model quality, data freshness, signal history and operator blockers.

Instrument search and watchlist expansion must be separated from ML support. Any instrument may be tracked, but ML signals require `model_supported=true` after data and validation checks.

## ML Defaults

Initial training baseline:

- timeframe: `5m`;
- prediction horizon: `12` bars, approximately 60 minutes;
- research comparison: `5m`, `15m`, `1h`;
- horizon comparison: `6`, `12`, `24` bars;
- neural input window candidates: `48` and `96` bars.

Data volume targets:

- tabular baseline minimum: `50k-200k` labeled rows;
- simple neural model minimum: `200k-500k` windows;
- robust multi-instrument target: `1M+` windows;
- preferred dataset: 1-2 years of `5m` data for 20-50 liquid instruments.

## Sprint 4. Forward Validation

Goal:
turn the Sprint 3 `shadow_live` policy candidate into a measurable forward-validation loop.

Already started:

- `GET /ml/policy/outcomes`;
- `POST /ml/policy/outcomes`;
- computed matured/pending/hit/miss summary;
- persisted matured outcomes in `signal_outcomes`;
- promotion blockers in outcome summary;
- stale-validation and overdue-pending blockers in outcome summary;
- tracked materialization jobs in `job_runs`;
- recurring backend scheduler for outcome materialization via env-configured interval;
- `GET /ml/policy/outcomes/history`;
- scheduler status endpoint/card in operator UI;
- GUI Forward outcomes card;
- GUI outcome audit card;
- `make sprint4-check`.

Remaining work:

- Extend blockers with stale-data age and critical gap checks.
- Add tests for:
  - up hit/miss;
  - down hit/miss;
  - no-trade maturity;
  - pending horizon;
  - missing candles.
- Add Sprint 4 completion note.

Exit criteria:

- `make sprint4-check` passes.
- Outcomes are persisted and reproducible.
- UI shows realized precision, pending count and promotion blockers.
- Candidate cannot be promoted without enough forward evidence.

## Sprint 5. Instrument Catalog And Flexible Watchlist

Goal:
allow the operator to search T-Bank instruments and add any suitable instrument to the local watchlist.

Completed:

- Add T-Bank InstrumentsService adapter:
  - `FindInstrument`;
  - `GetInstrumentByUID`.
- Add API endpoints:
  - `GET /instruments/search?query=...`;
  - `GET /instruments/{uid}`;
  - extend `POST /watchlist` to accept `instrument_uid`;
  - enrich `GET /watchlist` items with asset metadata.
- Extend `assets` schema:
  - `figi`;
  - `instrument_uid`;
  - `class_code`;
  - `instrument_type`;
  - `exchange`;
  - `lot`;
  - `currency`;
  - `api_trade_available`;
  - `first_1min_candle_date`;
  - `first_1day_candle_date`;
  - `model_supported`.
- Add instrument search panel.
- Add filters by instrument type.
- Add instrument details preview.
- Add button to add to watchlist.
- Show support status:
  - trackable;
  - data-loadable;
  - model-supported;
  - watchlist-only.

Exit criteria:

- Operator can search and add an instrument without editing config files.
- Watchlist supports instruments beyond the hardcoded MVP set.
- Unsupported instruments do not accidentally receive ML signals.
- `make sprint5-check` passes.

## Sprint 6. Scheduled Data Refresh And Watchlist Live Loop (Completed)

Goal:
make watchlist instruments refresh and signal generation repeatable instead of manual.

Backend and data work:

- Implement `job_runs` repository/service around existing DB table.
- Add jobs:
  - incremental candle refresh;
  - factor refresh;
  - signal generation for watchlist;
  - outcome materialization.
- Add idempotent retry policy.
- Add stale-data and missing-candle checks.
- Use T-Bank trading schedules to avoid false failures outside exchange sessions.
- Add manual endpoints:
  - `POST /jobs/data-refresh`;
  - `POST /jobs/signals/run`;
  - `POST /jobs/outcomes/materialize`;
  - `GET /jobs/runs`.

Frontend work:

- Add Jobs/Freshness panel.
- Show last refresh time per instrument.
- Show failed job reason and retry action.

Exit criteria:

- A local run can refresh data, produce signals and mature outcomes from one UI/API flow.
- Job history is visible.
- Re-running jobs is safe.

## Sprint 7. Dataset Expansion And Timeframe/Horizon Research (Closed)

Goal:
build enough data and evidence to choose the right timeframe/horizon before neural modeling.

Data work:

- Build a liquid instrument universe of 20-50 instruments.
- Backfill 1-2 years where API availability permits.
- Track per-instrument data coverage and gaps.
- Add explicit futures rollover note or block futures from ML training until rollover is implemented.

ML work:

- Build comparable datasets for:
  - `5m`, `15m`, `1h`;
  - horizons `6`, `12`, `24` bars.
- Run tabular baselines on each combination.
- Compare:
  - actionable F1;
  - precision;
  - signal coverage;
  - calibration error;
  - per-instrument stability.
- Choose the production research default for neural training.

Exit criteria:

- There is a documented timeframe/horizon decision.
- The decision is based on comparable metrics, not intuition.
- Dataset size is sufficient for the selected model family.

Closure:

- Decision document: `docs/ml/sprint7_timeframe_horizon_decision.md`.
- Selected Sprint 8 default: `1h`, horizon `24` bars.
- Data-backed universe: 25 ML-enabled TQBR instruments.
- Matrix artifact: `artifacts/research/sprint7/timeframe_horizon_matrix.json`.

## Sprint 8. Neural Model And Deployable Contract (Planned)

Goal:
train a first neural sequence model and make it deployable.

Detailed implementation plan:

- `docs/sprint8_plan.md`.

Starting point:

- timeframe: `1h`;
- prediction horizon: `24` bars;
- ML universe: 25 data-backed TQBR instruments;
- baseline comparator: `hgb_multiclass` from `sprint7_1h_h24_20260506`.

ML work:

- Implement sequence dataset builder with windows `48` and `96`.
- Train first neural baseline:
  - compact temporal CNN, GRU/LSTM, or transformer-lite;
  - multiclass `up/down/no_trade`;
  - class imbalance handling;
  - calibration audit.
- Compare neural model against tabular production baseline.
- Add model manifest fields for neural artifacts.

Registry/export work:

- Define deployable model contract:
  - feature ordering;
  - normalization;
  - input tensor shape;
  - output probabilities.
- Add export path, preferably ONNX if feasible.
- Add parity/golden tests for reference inference.

Exit criteria:

- Neural model is trained on scaled dataset.
- It beats or clearly fails against the baseline with documented evidence.
- A deployable artifact and manifest exist.

## Sprint 9. Promotion, Notifications And Monitoring

Goal:
turn validated models/signals into an operator-ready production loop.

Promotion work:

- Add model/policy lifecycle states:
  - `candidate`;
  - `shadow_live`;
  - `approved`;
  - `active`;
  - `blocked`;
  - `archived`.
- Add promotion log and rollback.
- Connect promotion to Sprint 4 blockers.

Notification work:

- Implement notification rules:
  - asset/instrument;
  - direction;
  - probability threshold;
  - cooldown;
  - policy state requirement.
- Persist signal events.
- Add mock notification channel first; real Telegram/email can be a later integration.

Monitoring work:

- Add production metrics:
  - realized precision decay;
  - coverage decay;
  - calibration decay;
  - data freshness;
  - factor staleness.
- Add UI monitoring view.

Exit criteria:

- A policy can be promoted, monitored and rolled back.
- Signal events are recorded.
- Operator can define notification rules.

## Sprint 10. Final Demo And Diploma Pack

Goal:
prepare the project for stable demonstration and thesis defense.

Engineering work:

- One-command local demo flow.
- Clean README and setup instructions.
- Smoke checks for backend, frontend, ML and compose.
- Seed/demo dataset instructions.

Documentation work:

- Final architecture document.
- API contract finalization.
- ML methodology:
  - target engineering;
  - feature space;
  - timeframe/horizon choice;
  - model comparison;
  - calibration;
  - limitations.
- Risk and compliance note:
  - not automated trading;
  - not investment advice;
  - data provider limitations.

Demo materials:

- Screenshots.
- Demo script.
- Final metrics table.
- Known limitations and future work.

Exit criteria:

- Project can be demonstrated end-to-end.
- Metrics and decisions are reproducible.
- The repository is understandable without oral context.

## Critical Path

1. Finish Sprint 4 persistence and blockers.
2. Add instrument search and flexible watchlist.
3. Add scheduled refresh/jobs for watchlist instruments.
4. Expand dataset and decide timeframe/horizon.
5. Train neural model only after the dataset decision.
6. Add promotion/notifications/monitoring.
7. Package for demo and defense.
