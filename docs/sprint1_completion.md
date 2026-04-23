# Sprint 1 Completion Note

Обновлено: 2026-04-21.

## Closed Scope

- Tinkoff Invest API ingest подготовлен для asset/factor candles, universe sync, incremental state, overlap re-sync and raw QA reports.
- `mvp_universe_v1` загружен в raw layout для `SBER`, `GAZP`, `LKOH`, `MOEX`, `NVTK` и факторов `usdrub`, `brent`, `rtsi`.
- Feature store и canonical dataset `mvp_v1_20260409` пересобраны из реальных данных.
- ML baseline research запущен на полном multi-ticker dataset; tracked report: `docs/ml/baseline_report_sprint1.md`.
- Backend переведен на embedded SQL migrations с `cmd/migrate` и auto-migrate на старте API.
- Frontend shell/prep pack подготовлен: Vite/React app, typed API mapping, mock fallback, screen map and wireframes.
- Исправлена idempotency логика parquet writer/dataset materialization: multi-ticker split больше не теряет строки с одинаковым timestamp и stale split files не остаются после пересборки dataset version.

## Verification

- `make ml-test`: 25 passed.
- `make backend-test`: passed.
- `make frontend-build`: passed.

## Remaining Risks

- `rf_multiclass` проходит production gate только для shadow/live monitoring: test coverage низкий, поэтому автоторговые решения пока запрещены.
- `hgb_multiclass` дает лучший research signal, но требует calibration work из-за высокого ECE.
- `brent` и `rtsi` заведены через futures proxies; нужен explicit rollover policy.
- Для стабильных ML-выводов нужен более длинный исторический период и walk-forward validation.
