# Sprint 5 Completion

Sprint 5 closes the first operator-facing instrument catalog slice: search through T-Bank InstrumentsService, persist selected metadata locally, and add discovered instruments into the watchlist without automatically enabling ML support.

## Готовность

- Backend:
  - added `GET /instruments/search`;
  - added `GET /instruments/{uid}`;
  - extended `POST /watchlist` to accept `instrument_uid`;
  - enriched `GET /watchlist` items with resolved asset metadata.
- Persistence:
  - added `005_instrument_catalog.sql`;
  - `assets` now stores instrument identity and metadata:
    - `figi`
    - `instrument_uid`
    - `class_code`
    - `instrument_type`
    - `lot`
    - `currency`
    - `api_trade_available`
    - first candle availability dates
    - `model_supported`
- Frontend:
  - added instrument search modal;
  - operator can search, preview metadata and add an instrument into the watchlist;
  - UI labels newly added instruments conservatively as `watchlist-only`.
- Verification:
  - backend router tests cover instrument search/details and add-to-watchlist flow;
  - `make sprint5-check` extends Sprint 4 smoke with watchlist and instrument-catalog checks.

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
make sprint5-check
```

Runtime catalog/watchlist smoke:

```sh
curl 'http://127.0.0.1:8080/watchlist'
curl -X POST 'http://127.0.0.1:8080/watchlist' \
  -H 'Content-Type: application/json' \
  -d '{"asset_id":"SBER","position":1}'
curl 'http://127.0.0.1:8080/instruments/search?query=sber'
curl 'http://127.0.0.1:8080/instruments/<uid>'
```

## Remaining Product Risks

- Instrument search still depends on T-Bank token availability and upstream API health.
- New instruments are persisted as `watchlist-only`; there is still no automatic path to dataset refresh, data QA or ML enablement.
- Watchlist management still lacks edit/remove/reorder UX beyond simple add/update.

## Sprint 6 Starting Point

- Add scheduled refresh/jobs for watchlist instruments.
- Track data freshness and job runs per watchlist instrument.
- Move from manual tracking setup to repeatable live update flow.
