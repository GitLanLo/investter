# Data and Model Contracts v1

## Цель

Зафиксировать соглашения по данным и model artifacts между `ml-core` и `backend`.

## Базовые правила

- все timestamps хранятся в `UTC`;
- business timezone для продуктовой логики: `Europe/Moscow`;
- все parquet-схемы должны быть versioned;
- production inference может использовать только те признаки и preprocessing steps, которые описаны в model manifest.

## Raw candles layout

Рекомендуемый layout:

`data/raw/candles/ticker=<TICKER>/timeframe=<TF>/date=<YYYY-MM-DD>/*.parquet`

Обязательные колонки:

- `timestamp`
- `ticker`
- `timeframe`
- `open`
- `high`
- `low`
- `close`
- `volume`
- `source`
- `ingested_at`

## Cross-factors layout

Рекомендуемый layout:

`data/raw/factors/factor=<ALIAS>/timeframe=<TF>/date=<YYYY-MM-DD>/*.parquet`

Обязательные колонки:

- `timestamp`
- `factor_alias`
- `timeframe`
- `open`
- `high`
- `low`
- `close`
- `volume`
- `source`
- `ingested_at`

## Feature store layout

Рекомендуемый layout:

`data/features/schema=<SCHEMA_VERSION>/ticker=<TICKER>/timeframe=<TF>/horizon=<HORIZON_BARS>/date=<YYYY-MM-DD>/*.parquet`

Обязательные идентификаторы:

- `asof_time`
- `ticker`
- `timeframe`
- `horizon_bars`
- `feature_schema_version`

Обязательное содержимое:

- mandatory feature columns из `docs/ml/feature_space_v1.md`
- label columns из `docs/ml/target_engineering_v1.md`

## Dataset layout

Рекомендуемый layout:

`data/datasets/dataset_version=<VERSION>/split=<train|val|test>/*.parquet`

Для каждого dataset version дополнительно нужен manifest:

`data/datasets/dataset_version=<VERSION>/manifest.json`

Минимальные поля manifest:

- `dataset_version`
- `feature_schema_version`
- `target_schema_version`
- `timeframe`
- `horizon_bars`
- `tickers`
- `created_at`
- `train_range`
- `val_range`
- `test_range`
- `purge_gap_bars`
- `feature_columns`

## Sequence dataset layout

Рекомендуемый layout:

`data/sequences/dataset_version=<VERSION>/window=<WINDOW_BARS>/split=<train|val|test>/*.parquet`

Для каждой sequence сборки дополнительно нужен manifest:

`data/sequences/dataset_version=<VERSION>/window=<WINDOW_BARS>/manifest.json`

Минимальные поля manifest:

- `input_shape`
- `window_length`
- `feature_count`
- `class_mapping`
- `source_dataset_version`
- `timeframe`
- `horizon_bars`
- `split_counts`
- `checksums`

## Model manifest contract

Для каждой deployable модели нужен manifest:

`artifacts/models/<model_version>/model_manifest.json`

Минимальные поля:

- `model_version`
- `model_type` / `model_family`
- `task_type` / `task`
- `classes`
- `timeframe` / `input_timeframe`
- `horizon_bars` / `prediction_horizon_bars`
- `input_window_bars` (for neural sequence models)
- `input_tensor_shape` (for neural sequence models)
- `feature_schema_version`
- `feature_columns` / `feature_order`
- `normalization_artifact_path` / `normalization`
- `export_format`
- `model_artifact_path`
- `artifact_sha256`
- `metrics`
- `threshold` / `decision_threshold`
- `calibration`
- `created_at`
- `source_dataset_version`

## Inference input contract

Backend обязан использовать:

- ровно тот же порядок `feature_columns`, что указан в manifest;
- тот же preprocessing, который использовался при обучении;
- тот же `timeframe` и `horizon_bars`, что записаны в manifest.

Если contract нарушен, модель не считается валидной для production loading.

## Signal output contract

Результат инференса в системе должен уметь содержать:

- `signal_state`
- `signal_direction`
- `signal_probability`
- `class_probabilities.up`
- `class_probabilities.down`
- `class_probabilities.no_trade`
- `model_version`
- `as_of_time`
- `timeframe`
- `horizon_bars`
