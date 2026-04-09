# ml-core

ML/data workspace для Sprint 1.

## Цель

Построить новый ML-контур под задачи проекта, используя `investML` как reference baseline, а не как финальную ML-основу.

## Основные направления Sprint 1

- сформулировать ML objective;
- спроектировать target engineering;
- спроектировать feature space v1;
- реализовать ingest abstraction, labeling и feature build pipeline;
- подготовить canonical datasets и advanced baselines.

## Текущее состояние

- есть базовый Python package `ml_core`;
- есть abstraction для локального parquet provider;
- есть contract-aligned raw/features/datasets layouts;
- есть feature build pipeline v1;
- есть volatility-aware triple barrier labeling;
- есть dataset split builder с manifest;
- есть CLI для ingest, materialization pipeline и загрузки market data из Tinkoff Invest API.

## Локальная разработка

```sh
. ../.venv/bin/activate
pip install -e .
pytest
```

## CLI

```sh
invest-ml build-features --asset-parquet path/to/asset.parquet --output path/to/features.parquet
invest-ml apply-labels --input-parquet path/to/features.parquet --output path/to/labeled.parquet
invest-ml split-dataset --input-parquet path/to/labeled.parquet --output-root path/to/dataset_dir --dataset-version v1
invest-ml ingest-asset-parquet --input-parquet path/to/sber.parquet --data-root ../data --ticker SBER --timeframe 5m
invest-ml ingest-factor-parquet --input-parquet path/to/usdrub.parquet --data-root ../data --alias usdrub --timeframe 5m
invest-ml materialize-feature-store --data-root ../data --ticker SBER --timeframe 5m --factor usdrub --factor brent --factor rtsi
invest-ml materialize-dataset --data-root ../data --output-root ../data/datasets --dataset-version v1 --ticker SBER
invest-ml run-walk-forward-research --dataset-root ../data/datasets/dataset_version=v1 --output-root ../artifacts/research/walk_forward_v1
invest-ml run-research-pipeline --data-root ../data --dataset-output-root ../data/datasets --research-output-root ../artifacts/research/mvp_run --dataset-version mvp_v1 --ticker SBER --factor usdrub --factor brent --factor rtsi --timeframe 5m
invest-ml tinkoff-sync-asset --data-root ../data --ticker SBER --from 2026-04-01T07:00:00Z --to 2026-04-02T07:00:00Z --timeframe 5m --instrument-kind share --class-code TQBR
invest-ml tinkoff-sync-factor --data-root ../data --alias usdrub --ticker USD000UTSTOM --from 2026-04-01T07:00:00Z --to 2026-04-02T07:00:00Z --timeframe 5m --instrument-kind currency --class-code CETS
```

Текущий `mvp_universe_v1` использует:

- `usdrub` через `USD000UTSTOM`;
- `brent` через front futures proxy `BRM6`;
- `rtsi` через front futures proxy `RIM6`.

Для `brent/rtsi` позже понадобится rollover policy, потому что это срочные контракты, а не вечные тикеры.

Для Tinkoff commands нужны переменные окружения:

```sh
export TINKOFF_INVEST_TOKEN=...
export TINKOFF_INVEST_TARGET=prod
```
