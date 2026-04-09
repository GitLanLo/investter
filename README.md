# invest

Рабочий Sprint 1 foundation для `invest`: backend, contracts, ML/data core и локальная инфраструктура.

## Что уже сделано

- подготовлены документы по ТЗ, roadmap, backlog и Sprint 1;
- создана базовая структура монорепозитория;
- добавлен `Go` backend с рабочими endpoint-ами `health`, `ready`, `assets`, `assets/{id}/candles`, `assets/{id}/factors`, `watchlist`, `analysis/run`, `signals/latest`;
- добавлен локальный `docker compose` для `PostgreSQL`;
- добавлены contracts/docs для архитектуры, API и ML;
- добавлен `ml-core` с feature build, labeling, dataset split, one-shot research pipeline, baseline training pack и Sprint 2 ablation runner;
- добавлен bootstrap model manifest для первого runtime path;
- добавлена интеграция загрузки исторических данных через Tinkoff Invest API REST proxy;
- добавлен frontend app shell/workspace с screen map, wireframes и API/domain mapping;
- backend переведён на normal migration files с compose-compatible bootstrap.

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
curl -X POST http://localhost:8080/analysis/run \
  -H 'Content-Type: application/json' \
  -d '{"asset_id":"SBER","as_of_time":"2026-04-09T09:55:00Z","model_version":"active","timeframe":"5m"}'
curl 'http://localhost:8080/signals/latest?limit=5'
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

## Документы Sprint 1

- `development_plan.md`
- `module_backlog.md`
- `sprint1_issue_cards.md`
- `github_issues_sprint1.md`
- `docs/architecture/overview.md`
- `docs/contracts/api_v1.md`
- `docs/contracts/data_model_contracts_v1.md`
- `docs/integrations/tinkoff_invest_api.md`
- `docs/ml/ml_objective_v1.md`
- `docs/ml/target_engineering_v1.md`
- `docs/ml/feature_space_v1.md`
- `docs/ml/research_protocol_v1.md`
