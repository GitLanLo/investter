# Sprint 3 Completion

Sprint 3 закрывает переход от research artifacts к контролируемому production policy candidate в runtime/API/GUI.

## Готовность

- Data refresh: Tinkoff Invest API данные обновлены до `2026-04-22T07:35:00Z`; dataset version `mvp_live_20260422`.
- ML policy: baseline research выбрал production-gated model `logreg_multiclass`; calibration audit выполнен для `logreg_multiclass`.
- Active candidate: `production_candidate`, calibration method `identity`, threshold `0.30`.
- Backend: добавлены normalized policy endpoint `GET /ml/policy/production`, validation ledger `GET/POST /ml/policy/validation-runs`, operator transition endpoint `PATCH /ml/policy/validation-runs/{id}` и shadow summary endpoint `GET /ml/policy/shadow-summary`.
- Runtime API: signal responses включают persisted policy snapshot для сравнения текущего сигнала с активным policy.
- Frontend: добавлены Policy Gate, validation run ledger, shadow monitor, operator transition actions, policy comparison strip, горизонтальный scroll графика, закрепленные оси и стабильные chart inspector/metrification blocks без layout jumps.
- Signal persistence: новые `signal_runs` сохраняют policy snapshot columns для корректной исторической атрибуции shadow/live метрик.
- Persistence: текущий policy snapshot сохранен как validation run `id=2`, decision state `shadow_live`.

## Verification

```sh
cd backend
go test ./...
```

```sh
cd frontend
npm run build
```

```sh
. .venv/bin/activate
cd ml-core
pytest
```

Single command, when backend and frontend are already running:

```sh
make sprint3-check
```

Runtime policy smoke:

```sh
curl 'http://127.0.0.1:8080/ml/policy/production'
curl 'http://127.0.0.1:8080/ml/policy/validation-runs?limit=5'
curl 'http://127.0.0.1:8080/ml/policy/shadow-summary?limit=1000'
curl 'http://127.0.0.1:8080/signals/latest?limit=1'
```

## Remaining Product Risks

- The candidate is ready for shadow/live validation, not automated trading execution.
- Promotion requires accumulated forward validation outcomes; shadow monitor tracks persisted policy signal coverage, but outcome metrics are not yet computed from realized future candles.
- Factors `brent` and `rtsi` still need explicit futures rollover policy.
- Tinkoff sync remains dependent on token validity, provider availability and exchange session gaps.

## Sprint 4 Starting Point

- Implement shadow/live validation loop over fresh candles and persisted signal outcomes.
- Add realized outcome metrics for shadow/live signals.
- Add production monitoring views for drift, precision and calibration decay.
