.PHONY: backend-run backend-build backend-test backend-migrate compose-up compose-up-all compose-down ml-install ml-test frontend-install frontend-build frontend-run

backend-run:
	cd backend && go run ./cmd/api

backend-build:
	cd backend && go build ./...

backend-test:
	cd backend && go test ./...

backend-migrate:
	cd backend && go run ./cmd/migrate

compose-up:
	docker compose -f infra/compose.yaml up -d postgres

compose-up-all:
	docker compose -f infra/compose.yaml up -d postgres backend frontend

compose-down:
	docker compose -f infra/compose.yaml down

ml-install:
	. .venv/bin/activate && cd ml-core && pip install -e '.[dev]'

ml-test:
	. .venv/bin/activate && cd ml-core && pytest

frontend-install:
	cd frontend && npm install

frontend-build:
	cd frontend && npm run build

frontend-run:
	cd frontend && npm run dev -- --host 0.0.0.0 --port 5173
