# API Contract v1

## Scope Sprint 1

В Sprint 1 этот документ фиксирует contract v1 для предметных endpoint-ов и уже принятые решения по семантике сигнала.

## Общие правила

- timestamps: `RFC3339 UTC`
- все domain ids в MVP могут совпадать с `ticker`
- ошибки возвращаются в едином формате
- signal semantics синхронизированы с `docs/ml/ml_objective_v1.md`

## Error response format

```json
{
  "error": {
    "code": "validation_error",
    "message": "invalid request payload",
    "details": {}
  }
}
```

## Endpoint status

### `GET /health`

- назначение: liveness check backend service
- status: implemented
- response:

```json
{
  "status": "ok",
  "service": "backend",
  "env": "dev",
  "timestamp": "2026-04-09T00:00:00Z"
}
```

### `GET /ready`

- назначение: readiness check backend service
- status: implemented
- response:

```json
{
  "status": "ready",
  "service": "backend",
  "env": "dev",
  "timestamp": "2026-04-09T00:00:00Z"
}
```

### `GET /assets`

- назначение: список инструментов для мониторинга
- status: implemented
- response example:

```json
{
  "items": [
    {
      "id": "SBER",
      "ticker": "SBER",
      "name": "Sberbank",
      "exchange": "MOEX",
      "timeframe": "5m",
      "is_active": true
    }
  ]
}
```

### `GET /assets/{id}/candles`

- назначение: исторические свечи по инструменту
- status: implemented
- query params:
  - `from`
  - `to`
  - `limit`
- response example:

```json
{
  "asset_id": "SBER",
  "timeframe": "5m",
  "items": [
    {
      "timestamp": "2026-04-09T09:55:00Z",
      "open": 301.1,
      "high": 301.7,
      "low": 300.9,
      "close": 301.5,
      "volume": 123456
    }
  ]
}
```

### `GET /assets/{id}/factors`

- назначение: связанные внешние факторы
- status: implemented
- query params:
  - `from`
  - `to`
- response example:

```json
{
  "asset_id": "SBER",
  "items": [
    {
      "factor": "usdrub",
      "timestamp": "2026-04-09T09:55:00Z",
      "close": 92.15
    },
    {
      "factor": "brent",
      "timestamp": "2026-04-09T09:55:00Z",
      "close": 81.42
    }
  ]
}
```

### `POST /analysis/run`

- назначение: запуск расчёта вероятности сигнала
- status: implemented with deterministic Sprint 1 stub runtime via active model manifest
- request example:

```json
{
  "asset_id": "SBER",
  "as_of_time": "2026-04-09T09:55:00Z",
  "model_version": "active",
  "timeframe": "5m"
}
```

- response example:

```json
{
  "asset_id": "SBER",
  "as_of_time": "2026-04-09T09:55:00Z",
  "signal_state": "actionable",
  "signal_direction": "down",
  "signal_probability": 0.799141,
  "class_probabilities": {
    "up": 0.112816,
    "down": 0.799141,
    "no_trade": 0.088042
  },
  "model_version": "baseline_stub_v1",
  "threshold": 0.65,
  "timeframe": "5m",
  "horizon_bars": 12
}
```

### `GET /signals/latest`

- назначение: получение последних сигналов
- status: implemented
- response example:

```json
{
  "items": [
    {
      "asset_id": "SBER",
      "as_of_time": "2026-04-09T09:55:00Z",
      "signal_state": "actionable",
      "signal_direction": "down",
      "signal_probability": 0.799141,
      "class_probabilities": {
        "up": 0.112816,
        "down": 0.799141,
        "no_trade": 0.088042
      },
      "model_version": "baseline_stub_v1",
      "threshold": 0.65,
      "timeframe": "5m",
      "horizon_bars": 12
    }
  ]
}
```

### `GET /watchlist`

- назначение: список наблюдаемых инструментов
- status: implemented for default watchlist
- response example:

```json
{
  "watchlist_id": 1,
  "name": "default",
  "items": [
    {
      "asset_id": "SBER",
      "position": 1
    }
  ]
}
```

### `POST /watchlist`

- назначение: добавление инструмента в список наблюдения
- status: implemented for default watchlist
- request example:

```json
{
  "asset_id": "SBER"
}
```

### `POST /notifications/rules`

- назначение: создание или изменение правила уведомлений
- request example:

```json
{
  "asset_id": "SBER",
  "enabled": true,
  "direction": "up",
  "threshold": 0.72,
  "cooldown_minutes": 60
}
```

## Принято по signal semantics

- модель работает как `up / down / no_trade`
- `signal_probability = max(p_up, p_down)`
- actionable signal возникает только если directional class победил и directional probability выше порога
- для UI и notifications backend возвращает и итоговое состояние сигнала, и полный probability breakdown
- в Sprint 1 backend использует bootstrap manifest `artifacts/models/baseline_stub_v1/model_manifest.json` и deterministic stub runtime как временный inference adapter
