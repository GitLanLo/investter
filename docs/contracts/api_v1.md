# API Contract v1

## Scope Sprint 2

В Sprint 2 этот документ фиксирует рабочий contract v1 для backend API и уже принятые решения по семантике сигнала и ML artifact access.

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

### `GET /assets/{id}/signals`

- назначение: история signal runs по выбранному инструменту
- status: implemented
- query params:
  - `limit`
- response example:

```json
{
  "items": [
    {
      "asset_id": "SBER",
      "as_of_time": "2026-04-11T14:04:00Z",
      "signal_state": "actionable",
      "signal_direction": "up",
      "signal_probability": 0.7375,
      "class_probabilities": {
        "up": 0.7375,
        "down": 0.1302,
        "no_trade": 0.1323
      },
      "model_version": "baseline_stub_v1",
      "threshold": 0.65,
      "timeframe": "5m",
      "horizon_bars": 12,
      "policy": {
        "policy_status": "calibration_review",
        "model_name": "logreg_multiclass",
        "scenario_name": "",
        "calibration_method": "",
        "threshold": 0.55,
        "dataset_version": "mvp_live_20260422"
      }
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

### `GET /ml/research/overview`

- назначение: latest dataset/research/calibration summary для operator GUI
- status: implemented
- response fields:
  - `dataset_manifest`
  - `research_summary`
  - `calibration_summary`
  - `source_paths`
  - `warnings`

### `GET /ml/research/documents`

- назначение: latest безопасный набор artifact documents для GUI drill-down
- status: implemented
- response example:

```json
{
  "generated_at": "2026-04-15T00:00:00Z",
  "items": [
    {
      "key": "research_report",
      "title": "Research report",
      "path": "../artifacts/research/mvp_live_wf_20260409/ablation_sprint2_richer_modelpack/report.md",
      "content_type": "markdown",
      "content": "# Research report\n\n..."
    }
  ]
}
```

### `GET /ml/policy/production`

- назначение: normalized production-candidate policy для Sprint 3 runtime/GUI слоя
- status: implemented
- response example:

```json
{
  "generated_at": "2026-04-15T00:00:00Z",
  "policy_status": "production_candidate",
  "model_name": "logreg_multiclass",
  "scenario_name": "",
  "calibration_method": "identity",
  "threshold": 0.3,
  "timeframe": "5m",
  "horizon_bars": 12,
  "dataset_version": "mvp_live_20260422",
  "feature_schema": "feature_v1",
  "train_rows": 59145,
  "validation_rows": 12771,
  "test_rows": 12641,
  "validation": {
    "actionable_f1": 0.343,
    "precision": 0.3082,
    "coverage": 0.4939,
    "actionable_ece": 0.1647
  },
  "test": {
    "actionable_f1": 0.3095,
    "precision": 0.2885,
    "coverage": 0.4291,
    "actionable_ece": 0.1763
  }
}
```

### `GET /ml/policy/validation-runs`

- назначение: список сохранённых проверок production policy перед shadow/live promotion
- status: implemented
- query params:
  - `limit`
- response example:

```json
{
  "items": [
    {
      "id": 1,
      "policy_status": "production_candidate",
      "model_name": "logreg_multiclass",
      "scenario_name": "",
      "calibration_method": "identity",
      "threshold": 0.3,
      "dataset_version": "mvp_live_20260422",
      "validation": {
        "actionable_f1": 0.343,
        "precision": 0.3082,
        "coverage": 0.4939,
        "actionable_ece": 0.1647
      },
      "test": {
        "actionable_f1": 0.3095,
        "precision": 0.2885,
        "coverage": 0.4291,
        "actionable_ece": 0.1763
      },
      "decision_state": "candidate",
      "notes": "shadow candidate",
      "created_at": "2026-04-21T15:00:00Z"
    }
  ]
}
```

### `POST /ml/policy/validation-runs`

- назначение: сохранить текущий normalized production policy snapshot как validation run
- status: implemented
- request example:

```json
{
  "notes": "shadow candidate"
}
```

- response: один объект из `GET /ml/policy/validation-runs`.

### `PATCH /ml/policy/validation-runs/{id}`

- назначение: операторский переход validation run между decision states перед shadow/live promotion
- status: implemented
- allowed `decision_state`: `candidate`, `shadow_live`, `promoted`, `blocked`
- request example:

```json
{
  "decision_state": "shadow_live",
  "notes": "approved for shadow validation"
}
```

- response: один объект из `GET /ml/policy/validation-runs`.

### `GET /ml/policy/shadow-summary`

- назначение: aggregate snapshot по сигналам, сохранённым с persisted policy metadata в `signal_runs`
- status: implemented
- query params:
  - `limit`
- response example:

```json
{
  "validation_run_id": 2,
  "decision_state": "shadow_live",
  "model_name": "logreg_multiclass",
  "calibration_method": "identity",
  "threshold": 0.3,
  "dataset_version": "mvp_live_20260422",
  "signals_total": 24,
  "actionable_signals": 9,
  "no_trade_signals": 15,
  "up_signals": 5,
  "down_signals": 4,
  "observed_coverage": 0.375,
  "first_signal_at": "2026-04-22T10:00:00Z",
  "last_signal_at": "2026-04-22T13:38:00Z"
}
```

### `GET /ml/policy/outcomes`

- назначение: Sprint 4 forward-validation summary по matured shadow/live signals
- status: implemented over persisted signals, stored outcomes and available candles
- query params:
  - `limit`
- semantics:
  - `matured_signals` имеют достаточно будущих свечей для `horizon_bars`;
  - `pending_signals` ещё ждут завершения горизонта;
  - `realized_precision` считается по actionable `up/down` сигналам как directional hit-rate;
  - `average_action_return_pct` учитывает направление сигнала: для `down` прибыльным считается отрицательный raw return.
- response example:

```json
{
  "validation_run_id": 2,
  "decision_state": "shadow_live",
  "model_name": "logreg_multiclass",
  "calibration_method": "identity",
  "threshold": 0.3,
  "dataset_version": "mvp_live_20260422",
  "signals_total": 24,
  "actionable_signals": 9,
  "matured_signals": 21,
  "pending_signals": 3,
  "overdue_pending_signals": 1,
  "hit_signals": 5,
  "miss_signals": 4,
  "realized_precision": 0.5556,
  "average_return_pct": 0.0012,
  "average_action_return_pct": 0.0021,
  "last_signal_at": "2026-04-24T16:55:00Z",
  "can_promote": false,
  "promotion_blockers": [
    {
      "code": "pending_signal_horizons",
      "message": "Some shadow/live signals have not completed their forecast horizon yet."
    }
  ],
  "first_matured_at": "2026-04-22T11:00:00Z",
  "last_matured_at": "2026-04-23T13:09:00Z"
}
```

### `POST /ml/policy/outcomes`

- назначение: вручную materialize все matured outcomes для активного `shadow_live`/`promoted` validation run
- status: implemented
- query params:
  - `limit`
- response: тот же payload, что и `GET /ml/policy/outcomes`, уже после upsert в `signal_outcomes`
- notes:
  - endpoint идемпотентен;
  - повторный запуск обновляет persisted outcome rows и возвращает актуальный summary.

### `GET /ml/policy/outcomes/history`

- назначение: получить последние matured signal outcomes для active `shadow_live`/`promoted` policy snapshot
- status: implemented
- query params:
  - `limit`
- response example:

```json
{
  "items": [
    {
      "signal_run_id": 101,
      "asset_id": "SBER",
      "as_of_time": "2026-04-24T10:00:00Z",
      "signal_state": "actionable",
      "signal_direction": "up",
      "signal_probability": 0.71,
      "timeframe": "5m",
      "horizon_bars": 12,
      "matured_at": "2026-04-24T11:00:00Z",
      "entry_price": 301.5,
      "exit_price": 303.0,
      "raw_return_pct": 0.00497,
      "action_return_pct": 0.00497,
      "is_hit": true
    }
  ]
}
```

### `POST /jobs/outcomes/materialize`

- назначение: запустить materialization matured outcomes как tracked backend job
- status: implemented
- query params:
  - `limit`
- response example:

```json
{
  "id": 7,
  "job_type": "outcomes_materialize",
  "status": "succeeded",
  "started_at": "2026-04-24T17:00:00Z",
  "finished_at": "2026-04-24T17:00:02Z",
  "payload": {
    "limit": 1000,
    "validation_run_id": 2,
    "matured_signals": 21,
    "pending_signals": 3,
    "overdue_pending_signals": 1,
    "can_promote": false,
    "promotion_blocker_count": 1
  }
}
```

### `GET /jobs/runs`

- назначение: получить последние job runs backend services
- status: implemented
- query params:
  - `limit`
- response example:

```json
{
  "items": [
    {
      "id": 7,
      "job_type": "outcomes_materialize",
      "status": "succeeded",
      "started_at": "2026-04-24T17:00:00Z",
      "finished_at": "2026-04-24T17:00:02Z",
      "payload": {
        "limit": 1000,
        "validation_run_id": 2,
        "matured_signals": 21,
        "pending_signals": 3,
        "overdue_pending_signals": 1,
        "can_promote": false
      }
    }
  ]
}
```

### `GET /jobs/scheduler`

- назначение: получить effective status/config recurring scheduler для outcome materialization
- status: implemented
- response example:

```json
{
  "enabled": true,
  "interval": "15m0s",
  "limit": 1000,
  "run_on_start": true
}
```

### `GET /watchlist`

- назначение: список наблюдаемых инструментов
- status: implemented for default watchlist with enriched asset metadata
- response example:

```json
{
  "watchlist_id": 1,
  "name": "default",
  "items": [
    {
      "asset_id": "SBER",
      "position": 1,
      "asset": {
        "id": "SBER",
        "ticker": "SBER",
        "name": "Sberbank",
        "exchange": "MOEX",
        "timeframe": "5m",
        "is_active": true,
        "figi": "BBG004730N88",
        "instrument_uid": "uid-sber",
        "class_code": "TQBR",
        "instrument_type": "share",
        "lot": 10,
        "currency": "rub",
        "api_trade_available": true,
        "model_supported": false
      }
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

Alternative request for discovered catalog instruments:

```json
{
  "instrument_uid": "uid-sber",
  "position": 2
}
```

### `GET /instruments/search`

- назначение: поиск инструментов через T-Bank InstrumentsService
- status: implemented
- query params:
  - `query`
- response example:

```json
{
  "items": [
    {
      "uid": "uid-sber",
      "figi": "BBG004730N88",
      "ticker": "SBER",
      "class_code": "TQBR",
      "isin": "RU0009029540",
      "lot": 10,
      "currency": "rub",
      "name": "Sberbank",
      "exchange": "MOEX",
      "instrument_type": "share",
      "api_trade_available": true,
      "first_1min_candle_date": "2024-01-10T07:00:00Z",
      "first_1day_candle_date": "2020-01-10T07:00:00Z"
    }
  ]
}
```

### `GET /instruments/{uid}`

- назначение: получить детальную metadata по выбранному instrument uid
- status: implemented
- response: same DTO as `GET /instruments/search`

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
- backend пока использует bootstrap manifest `artifacts/models/baseline_stub_v1/model_manifest.json` и deterministic stub runtime как временный inference adapter;
- Sprint 2 UI уже работает поверх signal history и ML artifact documents, даже если production inference path ещё stub-based.
