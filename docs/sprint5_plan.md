# Sprint 5 Plan

Status: completed.

Sprint 5 expands the operator workflow from a fixed local universe to a searchable instrument catalog backed by T-Bank InstrumentsService and a richer watchlist contract.

## Goal

- Search T-Bank instruments by ticker, uid or name.
- Load full metadata for a chosen instrument.
- Add discovered instruments to the local watchlist and asset catalog.
- Keep ML support conservative: newly discovered instruments are tracked first, but remain `watchlist-only` until model support is explicitly enabled.

## Scope

- Backend instrument endpoints:
  - `GET /instruments/search`
  - `GET /instruments/{uid}`
- Extended asset catalog migration with:
  - `figi`
  - `instrument_uid`
  - `class_code`
  - `instrument_type`
  - `lot`
  - `currency`
  - `api_trade_available`
  - `first_1min_candle_date`
  - `first_1day_candle_date`
  - `model_supported`
- Extended `POST /watchlist` to accept `instrument_uid`.
- Enriched `GET /watchlist` response with resolved asset metadata.
- Frontend instrument search modal and add-to-watchlist flow.

## Exit Criteria

- Operator can search instruments and inspect basic metadata.
- Operator can add a selected instrument into the local watchlist.
- Added instrument is persisted into `assets` with instrument metadata.
- Watchlist response is enriched enough to render tracked instrument state without extra lookups.
- `make sprint5-check` passes on a running local stack.
