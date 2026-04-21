# Sprint 2 Completion

Sprint 2 закрывает связку `ML research -> backend artifacts -> operator GUI`.

## Готовность

- Research stack: ablation workflow, richer model pack, production gate and calibration audit are available under `ml-core`.
- Backend: research overview/documents, asset signal history, market data and analysis runtime endpoints are wired into the API.
- Frontend: operator GUI includes asset workbench, candle chart, timeframe aggregation, mouse range selection, zoom/pan controls, detached chart window, factor sync, signal history and artifact browser.
- Reliability: frontend guards against runtime blank screens with an error boundary and keeps chart-derived state stable after async data load.
- Infrastructure: compose bootstrap still supports local Postgres and backend/frontend local run.

## Verification

```sh
cd backend
go test ./...
```

```sh
cd frontend
npm run build
```

Manual smoke:

```sh
curl http://127.0.0.1:8080/health
curl http://127.0.0.1:8080/ready
curl 'http://127.0.0.1:8080/ml/research/overview'
curl 'http://127.0.0.1:8080/ml/research/documents'
curl 'http://127.0.0.1:8080/assets/SBER/signals?limit=5'
```

Single command, when backend and frontend are already running:

```sh
make sprint2-check
```

## Remaining Product Risks

- Production-candidate policy still needs forward live validation on a larger market window before real trading usage.
- Tinkoff live sync depends on token validity and API availability; keep incremental QA reports in the review loop.
- The GUI is ready as an operator cockpit, not yet as a retail-grade product surface.
