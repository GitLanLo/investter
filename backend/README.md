# backend

Backend foundation для Sprint 1.

## Текущее состояние

- есть конфиг окружения;
- есть HTTP router;
- есть `health` и `ready` endpoints;
- есть PostgreSQL-aware storage layer;
- есть normal migration path через `backend/internal/storage/migrations` и `cmd/migrate`;
- есть foundation для `repository` и `service` слоёв;
- реализованы `GET /assets`, `GET /assets/{id}/candles`, `GET /assets/{id}/factors`, `GET /watchlist`, `POST /watchlist`;
- реализованы `POST /analysis/run`, `GET /signals/latest`;
- есть bootstrap model manifest registration из `artifacts/models/baseline_stub_v1/model_manifest.json`;
- подготовлен Dockerfile для локальной сборки;
- сервис подключён к PostgreSQL и пишет `signal_runs`.

## Команды

```sh
go run ./cmd/migrate
go run ./cmd/api
go test ./...
```

`cmd/api` также прогоняет миграции на старте, поэтому локальный runtime и compose не расходятся по схеме.

## Следующие задачи

- notification rules и jobs;
- реальная model loading/inference интеграция вместо deterministic stub runtime;
- ONNX integration.
