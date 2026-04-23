.PHONY: backend-run backend-build backend-test backend-migrate compose-up compose-up-all compose-down ml-install ml-test frontend-install frontend-build frontend-run sprint2-check sprint2-smoke sprint3-check sprint3-smoke

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

sprint2-check: backend-test frontend-build sprint2-smoke

sprint3-check: backend-test frontend-build sprint3-smoke

sprint2-smoke:
	curl -fsS http://127.0.0.1:8080/health >/dev/null
	curl -fsS http://127.0.0.1:8080/ready >/dev/null
	curl -fsS 'http://127.0.0.1:8080/ml/research/overview' >/dev/null
	curl -fsS 'http://127.0.0.1:8080/ml/research/documents' >/dev/null
	curl -fsS 'http://127.0.0.1:8080/assets/SBER/signals?limit=5' >/dev/null
	curl -fsSI http://127.0.0.1:5173/ >/dev/null

sprint3-smoke: sprint2-smoke
	curl -fsS 'http://127.0.0.1:8080/ml/policy/production' >/dev/null
	curl -fsS 'http://127.0.0.1:8080/ml/policy/validation-runs?limit=5' >/dev/null
	curl -fsS 'http://127.0.0.1:8080/ml/policy/shadow-summary?limit=1000' >/dev/null
