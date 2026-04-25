# Sprint 4 Completion

Sprint 4 закрывает forward-validation контур для production policy candidate из Sprint 3 и переводит shadow/live проверку в воспроизводимый runtime workflow.

## Готовность

- Backend:
  - added forward-validation summary endpoint `GET /ml/policy/outcomes`;
  - added manual materialization endpoint `POST /ml/policy/outcomes`;
  - added outcome audit endpoint `GET /ml/policy/outcomes/history`;
  - added tracked job endpoints `POST /jobs/outcomes/materialize`, `GET /jobs/runs`, `GET /jobs/scheduler`.
- Persistence:
  - introduced `signal_outcomes` table for persisted matured outcomes;
  - matured outcomes are upserted and reused for summaries and audit views.
- Promotion control:
  - policy promotion is blocked by forward-validation evidence;
  - blockers now include insufficient matured signals, pending horizons, overdue pending outcomes, stale forward-validation and low realized precision.
- Scheduler:
  - backend supports recurring outcome materialization via env-configured scheduler;
  - startup run is supported for immediate job bootstrap after service start.
- Frontend:
  - Policy Gate now shows Forward outcomes, Outcome audit, Recent jobs and Scheduler status;
  - operator can see matured/pending/overdue state, blocker reasons, recent job payloads and effective scheduler configuration.
- Verification path:
  - `make sprint4-check` covers outcome summary, history, jobs and scheduler endpoints on top of Sprint 3 smoke.

## Verification

```sh
cd backend
go test ./...
```

```sh
cd frontend
npm run build
```

Single command, when backend and frontend are already running:

```sh
make sprint4-check
```

Runtime forward-validation smoke:

```sh
curl 'http://127.0.0.1:8080/ml/policy/outcomes?limit=1000'
curl 'http://127.0.0.1:8080/ml/policy/outcomes/history?limit=5'
curl -X POST 'http://127.0.0.1:8080/jobs/outcomes/materialize?limit=1000'
curl 'http://127.0.0.1:8080/jobs/runs?limit=5'
curl 'http://127.0.0.1:8080/jobs/scheduler'
```

## Remaining Product Risks

- Forward-validation loop is operational, but realized outcomes are still limited by available future candles and data completeness.
- Overdue pending outcomes currently indicate likely missing candles or stale data, but there is no dedicated operator dashboard for data quality drill-down yet.
- Drift and calibration decay monitoring are not implemented yet; operator sees outcome-based blockers, not post-promotion quality decay views.
- Automated trading is still out of scope; policy remains in controlled validation workflow.

## Sprint 5 Starting Point

- Add instrument search and metadata lookup through T-Bank InstrumentsService.
- Allow operator to add discovered instruments into watchlist with richer instrument identity fields.
- Keep ML support conservative: watchlist can accept broader universe, but model-supported trading/validation should remain explicit.
