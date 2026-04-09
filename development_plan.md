# Plan for investML Development

## 1. Input materials

- Technical specification: `ТЗ_ВКР_веб_система_анализа_рынка_ценных_бумаг.docx`
- Existing makeup / research baseline: `https://github.com/GitLanLo/investML`
- Observed repository state: branch `fix-data`, commit `1d51323` (`baseline`)

## 2. Short analysis of the technical specification

The target system is not just an ML experiment. According to the specification, it must become a web system that:

- ingests market and cross-asset data by API;
- stores history and derived features in Parquet;
- trains and compares ML models, with Transformer as the main model;
- exports the chosen model to ONNX;
- runs online inference in a backend service;
- lets a user manage a watchlist, thresholds, and notifications;
- shows analytics, signal history, and model metadata in a web UI.

In practice, the system has 5 major contours:

1. Data ingestion and quality control.
2. Feature engineering and dataset preparation.
3. Offline ML research and model training.
4. Online inference and orchestration.
5. User-facing web application.

## 3. What already exists in the makeup repository

The `investML` repository already gives a strong research baseline:

- Python pipeline for ingest from Tinkoff Invest API.
- Feature generation, QC, labels, and parquet-based datasets.
- Cross-asset enrichment (`usdrub`, `brent`, etc.).
- Baseline models and transformer training.
- ONNX export function in `src/training/train_tst.py`.
- FastAPI skeleton in `src/serve/api.py`.
- Separate local backtest mini-UI in `backtest_ui/`.
- Tests for ingest, features, datasets, and training smoke scenarios.

This means the repository already covers a large part of:

- `FR-1` data acquisition;
- `FR-2` historical storage;
- part of `FR-3` signal probability calculation in offline form;
- part of the ML requirements section;
- part of the data architecture section.

## 4. Main gaps between the specification and the current baseline

The current `investML` repository is still a research toolbox, not the target web system from the specification.

Critical gaps:

- No Go backend, while the specification explicitly requires backend on Go.
- No production inference service around ONNX Runtime in Go.
- No real REST API matching the specification (`/assets`, `/watchlist`, `/signals/latest`, `/analysis/run`, etc.).
- No user data storage for watchlists, notification rules, event history, and model registry.
- No actual notification subsystem, only config hints for alerting.
- No full user-facing analytical frontend, only a local backtest utility.
- No scheduler/orchestrator for regular online data refresh and signal recalculation.
- No explicit model versioning and publication flow from training to production.
- No integration layer between Python training artifacts and a production backend.

Conclusion:

The best strategy is not to rewrite the current repository blindly. It should be treated as a reference ML/data baseline: useful for pipeline patterns and experiments, but not as the final ML logic of the product.

## 5. Recommended target architecture

Recommended architecture for the diploma prototype:

- `ml-core` in Python:
  data ingest, feature build, training, evaluation, ONNX export, batch utilities.
- `backend` in Go:
  REST API, scheduler, watchlist management, signal history, notification rules, inference orchestration.
- `data lake` in Parquet:
  candles, cross-factors, features, datasets, inference inputs/outputs snapshots.
- `service database`:
  watchlists, users, notification rules, signal events, model registry, job logs.
- `frontend`:
  analytical dashboard, asset page, signal history, settings.

For a diploma prototype, the service database can be:

- `PostgreSQL` if you want closer-to-production architecture;
- `SQLite` if you want to reduce infrastructure and focus on the demo.

Recommended choice for the first full prototype: `PostgreSQL`.

## 6. Development priorities

Priority order should be:

1. Stabilize data and ML pipeline.
2. Formalize interfaces between Python and Go.
3. Build production inference path.
4. Build domain backend and storage.
5. Build frontend around real backend data.
6. Add notifications and scheduled jobs.
7. Finish testing, demo scripts, and thesis artifacts.

This order is important because the frontend has low value until the signal pipeline and API contracts are stable.

## 7. Step-by-step development plan

### Phase 0. Project framing and contract definition

Goal:
fix the exact boundaries of the diploma prototype and freeze the first version of contracts.

Tasks:

- Approve the list of tracked instruments for MVP.
- Approve the list of cross-factors for MVP (`USD/RUB`, `Brent`, market index).
- Freeze the signal definition:
  horizon, threshold, label logic, target metric.
- Freeze the MVP API contract from the specification.
- Decide whether prototype auth is needed.
- Decide `PostgreSQL` vs `SQLite`.
- Split repository strategy:
  monorepo or separate `backend`, `frontend`, `ml-core`.

Deliverables:

- architecture diagram;
- entity model;
- API draft;
- final MVP scope.

### Phase 1. Rebuild the ML/data baseline for the project

Goal:
turn the current research repo into a reproducible ML core while redesigning the ML logic specifically for the project signal.

Tasks:

- Clean configuration structure and separate `dev`, `demo`, `prod-like` profiles.
- Reformulate the prediction target and signal semantics for the project.
- Redesign labeling logic, advanced feature space, and evaluation protocol.
- Add a reproducible pipeline command:
  ingest -> QC -> features -> splits -> training -> evaluation -> export.
- Add artifact naming conventions for datasets and trained models.
- Add model registry metadata file:
  model version, feature set version, training window, metrics, export path.
- Verify ONNX export parity against PyTorch inference.
- Extend tests for feature schema stability and export consistency.

Deliverables:

- stable training pipeline;
- project-specific ML logic instead of inherited generic classifier logic;
- stable ONNX artifact;
- metadata manifest for deployable model versions.

### Phase 2. Data layer and storage conventions

Goal:
formalize how offline and online contours exchange data.

Tasks:

- Define parquet layout for raw candles, cross-factors, features, and inference-ready windows.
- Define retention/update rules for historical data.
- Add dataset versioning by date/profile/model-ready tag.
- Add validation jobs for duplicates, holes, stale cross-data, and schema drift.
- Define service DB schema:
  `assets`, `watchlists`, `watchlist_items`, `notification_rules`, `signal_runs`, `signal_events`, `model_registry`, `job_runs`.

Deliverables:

- data contracts;
- DB schema migration plan;
- storage policy document.

### Phase 3. Production inference path

Goal:
make model inference callable from backend with predictable inputs and outputs.

Tasks:

- Finalize ONNX export interface:
  input tensor shape, feature ordering, normalization policy.
- Implement reference inference runner and validation fixtures.
- Define request/response contract for probability scoring.
- Add thresholding logic and abstain band policy.
- Decide inference mode:
  embedded ONNX Runtime in Go or Python sidecar for v1.

Recommended implementation:

- v1 for diploma: keep training in Python, run online inference in Go with ONNX Runtime.
- Fallback if timeline slips: Python inference microservice behind Go API facade.

Deliverables:

- deployable ONNX model;
- inference contract;
- golden test cases comparing Python and production inference.

### Phase 4. Go backend foundation

Goal:
build the actual backend required by the specification.

Tasks:

- Create Go service skeleton with modular packages:
  `api`, `assets`, `signals`, `watchlist`, `notifications`, `models`, `jobs`, `storage`.
- Add config management and environment profiles.
- Implement database access and migrations.
- Implement basic observability:
  structured logs, request ids, job logs, health/readiness endpoints.
- Integrate ONNX Runtime and signal scoring service.

Required MVP endpoints:

- `GET /assets`
- `GET /assets/{id}/candles`
- `GET /assets/{id}/factors`
- `POST /analysis/run`
- `GET /signals/latest`
- `GET /watchlist`
- `POST /watchlist`
- `POST /notifications/rules`

Deliverables:

- runnable Go backend;
- REST API matching MVP scope;
- DB-backed domain entities.

### Phase 5. Scheduled jobs and event history

Goal:
support continuous monitoring rather than one-off offline experiments.

Tasks:

- Implement jobs for regular data refresh.
- Implement jobs for feature refresh and signal recalculation.
- Save every scoring run with timestamp, model version, and probability.
- Add event history for triggered notifications.
- Add stale-data checks and failure states.

Deliverables:

- periodic data update;
- periodic signal generation;
- persisted signal history.

### Phase 6. Frontend analytical web system

Goal:
replace the local backtest helper with the actual user-facing web UI.

MVP screens:

- dashboard with tracked assets and latest signal probability;
- asset details page with chart, factors, and signal history;
- watchlist management;
- notification threshold settings;
- model/version status and last update time.

Tasks:

- Build frontend against real backend API.
- Show probability, thresholds, last refresh, and signal trend.
- Add charting for candles and cross-factors.
- Add UI states for loading, stale data, and errors.
- Prepare demo-first UX suitable for thesis defense.

Deliverables:

- analytical dashboard;
- asset details page;
- watchlist/settings views.

### Phase 7. Notification subsystem

Goal:
implement `FR-5` and make the system behave like a monitoring product.

Tasks:

- Implement notification rules per asset or per watchlist.
- Support threshold crossing logic and cooldowns.
- Add at least one real channel for MVP:
  Telegram is the most pragmatic option.
- Save notification history and delivery status.

Deliverables:

- working alert pipeline;
- notification event log;
- configurable rules.

### Phase 8. Testing and acceptance validation

Goal:
prove that the system meets the specification and is stable enough for demonstration.

Tasks:

- Unit tests for domain logic and API handlers.
- Integration tests for backend + DB + inference.
- Regression tests for data contracts and model contracts.
- End-to-end demo scenario:
  ingest data -> compute signal -> show in UI -> trigger notification.
- Acceptance checklist mapped to section 14 of the specification.

Deliverables:

- automated test suite;
- acceptance checklist;
- demo сценарий для защиты.

### Phase 9. Documentation and thesis packaging

Goal:
prepare the project not only as code, but as a defendable graduation result.

Tasks:

- Architecture document.
- API documentation.
- Data flow and model lifecycle documentation.
- Experiment summary:
  baselines vs Transformer, metrics, role of external factors.
- Deployment and demo instructions.

Deliverables:

- complete project docs;
- thesis-ready technical appendix;
- reproducible demo script.

## 8. Suggested MVP scope

To keep the work realistic, the MVP should include only:

- 5 to 10 MOEX instruments;
- 2 to 3 cross-factors;
- one final trained Transformer model plus 2 baseline models in research results;
- one notification channel;
- one analytical dashboard and one detail page;
- one manually triggered analysis mode plus one scheduled recalculation mode.

Anything beyond that is useful, but not necessary for a strong diploma prototype.

## 9. Suggested repository split

Most practical structure:

- `ml-core/`
  current Python pipeline, training, export, research notebooks.
- `backend/`
  Go REST API, jobs, ONNX inference, service DB access.
- `frontend/`
  web client.
- `infra/`
  compose, env examples, migration/bootstrap scripts.
- `docs/`
  architecture, API, demo сценарии.

If you prefer monorepo, keep these as top-level directories.

## 10. Risks and mitigations

Risk 1:
research code and product code get mixed, slowing both tracks.

Mitigation:
separate `ml-core` from production backend early.

Risk 2:
ONNX export works offline but differs in production inference.

Mitigation:
add golden parity tests before backend integration.

Risk 3:
frontend starts too early and blocks on unstable contracts.

Mitigation:
freeze API schemas before UI implementation.

Risk 4:
too many assets/factors increase data and debugging complexity.

Mitigation:
limit MVP universe and expand only after the first full vertical slice works.

Risk 5:
notification logic becomes noisy and hard to demonstrate.

Mitigation:
add threshold hysteresis, cooldown, and clear event history.

## 11. Final recommendation

The project should be developed as a vertical slice in this order:

1. Stabilize Python ML/data core.
2. Export and validate ONNX.
3. Build Go backend around model scoring and history persistence.
4. Connect a minimal analytical frontend.
5. Add watchlist, rules, and notifications.
6. Finish tests, documentation, and thesis presentation artifacts.

This path minimizes rework and aligns the existing `investML` baseline with the architecture demanded by the specification.
