# invest

Рабочий Sprint 5 baseline для `invest`: backend, ML/data core, operator GUI, forward validation и instrument catalog.

## Что уже сделано

- подготовлены документы по ТЗ, roadmap, backlog и Sprint 1/Sprint 2/Sprint 3;
- создана базовая структура монорепозитория;
- добавлен `Go` backend с рабочими endpoint-ами `health`, `ready`, `assets`, `assets/{id}/candles`, `assets/{id}/factors`, `assets/{id}/signals`, `watchlist`, `analysis/run`, `signals/latest`, `ml/research/overview`, `ml/research/documents`, `ml/policy/production`, `ml/policy/validation-runs`;
- добавлен локальный `docker compose` для `PostgreSQL`;
- добавлены contracts/docs для архитектуры, API и ML;
- добавлен `ml-core` с feature build, labeling, dataset split, one-shot research pipeline, walk-forward research, ablation/calibration workflows и richer model pack;
- добавлен bootstrap model manifest для первого runtime path;
- добавлена интеграция загрузки исторических данных через Tinkoff Invest API REST proxy;
- добавлен frontend operator GUI с интерактивным candle workbench, range selection, factor sync, signal overlay, artifact browser и signal history;
- backend переведён на normal migration files с compose-compatible bootstrap.
- Sprint 1 закрыт на реальных Tinkoff данных; baseline report: `docs/ml/baseline_report_sprint1.md`.
- Sprint 3 закрыт; completion note: `docs/sprint3_completion.md`.
- Sprint 4 закрыт; completion note: `docs/sprint4_completion.md`.
- Sprint 5 закрыт; completion note: `docs/sprint5_completion.md`.
- Sprint 6 закрыт; completion note: `docs/sprint6_completion.md`.
- Sprint 7 активен; plan: `docs/sprint7_plan.md`.
- данные обновлены до `mvp_live_20260422`; активный Sprint 3 policy candidate: `logreg_multiclass` + `identity` calibration, threshold `0.30`, validation run `id=2` в состоянии `shadow_live`; refresh note: `docs/ml/data_refresh_20260422.md`.

## Структура

- `backend/` — Go backend skeleton.
- `frontend/` — frontend workspace.
- `ml-core/` — ML/data pipeline workspace.
- `infra/` — локальная инфраструктура и compose.
- `docs/` — архитектурные и контрактные документы.

## Быстрый старт

1. Скопировать `.env.example` в `.env`.
2. Поднять `PostgreSQL`:

```sh
docker compose -f infra/compose.yaml up -d postgres
```

3. Прогнать backend migrations:

```sh
make backend-migrate
```

4. Запустить backend локально:

```sh
make backend-run
```

5. Запустить frontend shell:

```sh
make frontend-install
make frontend-run
```

6. Проверить:

```sh
curl http://localhost:8080/health
curl http://localhost:8080/ready
curl 'http://localhost:8080/assets/SBER/candles?limit=5'
curl 'http://localhost:8080/assets/SBER/factors?from=2026-04-09T09:00:00Z&to=2026-04-09T10:00:00Z'
curl 'http://localhost:8080/assets/SBER/signals?limit=5'
curl -X POST http://localhost:8080/analysis/run \
  -H 'Content-Type: application/json' \
  -d '{"asset_id":"SBER","as_of_time":"2026-04-09T09:55:00Z","model_version":"active","timeframe":"5m"}'
curl 'http://localhost:8080/signals/latest?limit=5'
curl 'http://localhost:8080/ml/research/overview'
curl 'http://localhost:8080/ml/research/documents'
```

Для полного Sprint 2 smoke-check при уже запущенных backend/frontend:

```sh
make sprint2-check
```

## Загрузка данных из Tinkoff Invest API

```sh
export TINKOFF_INVEST_TOKEN=...
. .venv/bin/activate
invest-ml tinkoff-sync-universe \
  --config configs/mvp_universe_v1.json \
  --data-root data \
  --from 2026-04-08T07:00:00Z \
  --to 2026-04-09T07:00:00Z
```

## One-shot Research Pipeline

```sh
. .venv/bin/activate
invest-ml run-research-pipeline \
  --data-root data \
  --dataset-output-root data/datasets \
  --research-output-root artifacts/research/mvp_run \
  --dataset-version mvp_run \
  --ticker SBER \
  --ticker GAZP \
  --ticker LKOH \
  --ticker MOEX \
  --ticker NVTK \
  --factor usdrub \
  --factor brent \
  --factor rtsi \
  --run-ablation \
  --timeframe 5m
```

## Sprint 2 highlights

- Sprint 2 closure note: `docs/sprint2_completion.md`;
- chart workbench с zoom, brush-selection и detached chart window;
- range-synced factor strip и summary по выбранному диапазону;
- live signal overlay и asset-level signal history;
- artifact browser для latest dataset/research/calibration documents;
- Sprint 2 research stack с walk-forward, production gate и calibration audit.

## Sprint 3 highlights

- Sprint 3 closure note: `docs/sprint3_completion.md`;
- normalized production policy API and validation run ledger;
- operator transitions for policy validation runs: `candidate`, `shadow_live`, `promoted`, `blocked`;
- persisted policy snapshot in new `signal_runs` plus shadow summary endpoint;
- active policy candidate `logreg_multiclass` with `identity` calibration and threshold `0.30`;
- GUI Policy Gate, policy-vs-signal comparison and chart layout hardening;
- data refresh and calibration report: `docs/ml/data_refresh_20260422.md`.

## Sprint 4 status

- Sprint 4 closure note: `docs/sprint4_completion.md`;
- Sprint 4 plan: `docs/sprint4_plan.md`;
- forward-validation outcome summary endpoint: `GET /ml/policy/outcomes`;
- outcome audit endpoint: `GET /ml/policy/outcomes/history`;
- manual outcome materialization endpoint: `POST /ml/policy/outcomes`;
- tracked outcome materialization job: `POST /jobs/outcomes/materialize`;
- scheduler status endpoint: `GET /jobs/scheduler`;
- recurring backend scheduler via `OUTCOME_SCHEDULER_ENABLED`, `OUTCOME_SCHEDULER_INTERVAL`, `OUTCOME_SCHEDULER_LIMIT`, `OUTCOME_SCHEDULER_RUN_ON_START`;
- promotion blockers include stale validation and overdue pending outcomes;
- persisted matured outcomes in `signal_outcomes`;
- `make sprint4-check` extends Sprint 3 smoke with realised policy outcome metrics.

## Sprint 6 status

- Sprint 6 closure note: `docs/sprint6_completion.md`;
- Sprint 6 plan: `docs/sprint6_plan.md` (if applicable);
- Incremental market data refresh (Tinkoff API ingest in Go);
- Market schedule awareness (stale data checks skip closed markets);
- Partial failure support and robust job semantics;
- Frontend jobs panel displays failed job reasons and retry actions;
- `make sprint6-check` passes.

## Sprint 7 status

- Sprint 7 plan: `docs/sprint7_plan.md`;
- Expand universe to 20-50 liquid instruments;
- Execute research grid across timeframes (5m, 15m, 1h) and horizons (6, 12, 24);
- Determine optimal timeframe/horizon default for production.

## Документы

- `development_plan.md`
- `docs/development_sprint_plan.md`
- `module_backlog.md`
- `sprint1_issue_cards.md`
- `github_issues_sprint1.md`
- `docs/architecture/overview.md`
- `docs/sprint1_completion.md`
- `docs/sprint2_completion.md`
- `docs/sprint3_completion.md`
- `docs/sprint4_completion.md`
- `docs/sprint5_completion.md`
- `docs/sprint3_plan.md`
- `docs/sprint4_plan.md`
- `docs/sprint5_plan.md`
- `docs/contracts/api_v1.md`
- `docs/contracts/data_model_contracts_v1.md`
- `docs/integrations/tinkoff_invest_api.md`
- `docs/ml/baseline_report_sprint1.md`
- `docs/ml/data_refresh_20260422.md`
- `docs/ml/ml_objective_v1.md`
- `docs/ml/target_engineering_v1.md`
- `docs/ml/feature_space_v1.md`
- `docs/ml/research_protocol_v1.md`
