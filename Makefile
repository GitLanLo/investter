.PHONY: backend-run backend-build backend-test backend-migrate compose-up compose-up-all compose-down ml-install ml-test frontend-install frontend-build frontend-run sprint2-check sprint2-smoke sprint3-check sprint3-smoke sprint4-check sprint4-smoke sprint5-check sprint5-smoke sprint6-check sprint6-smoke sprint7-check sprint7-smoke sprint7-research-run sprint8-check sprint8-smoke sprint9-check sprint9-smoke

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

sprint4-check: backend-test frontend-build sprint4-smoke

sprint5-check: backend-test frontend-build sprint5-smoke

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

sprint4-smoke: sprint3-smoke
	curl -fsS -X POST 'http://127.0.0.1:8080/jobs/outcomes/materialize?limit=1000' >/dev/null
	curl -fsS 'http://127.0.0.1:8080/jobs/runs?limit=5' >/dev/null
	curl -fsS 'http://127.0.0.1:8080/jobs/scheduler' >/dev/null
	curl -fsS -X POST 'http://127.0.0.1:8080/ml/policy/outcomes?limit=1000' >/dev/null
	curl -fsS 'http://127.0.0.1:8080/ml/policy/outcomes?limit=1000' >/dev/null
	curl -fsS 'http://127.0.0.1:8080/ml/policy/outcomes/history?limit=5' >/dev/null

sprint5-smoke: sprint4-smoke
	curl -fsS 'http://127.0.0.1:8080/watchlist' >/dev/null
	curl -fsS -X POST 'http://127.0.0.1:8080/watchlist' -H 'Content-Type: application/json' -d '{"asset_id":"SBER","position":1}' >/dev/null
	if [ -n "$$TINKOFF_INVEST_TOKEN" ]; then curl -fsS 'http://127.0.0.1:8080/instruments/search?query=sber' >/dev/null; fi

sprint6-check: backend-test frontend-build sprint6-smoke

sprint6-smoke: sprint5-smoke
	curl -fsS 'http://127.0.0.1:8080/watchlist/freshness' >/dev/null
	curl -fsS 'http://127.0.0.1:8080/jobs/schedulers' >/dev/null
	curl -fsS -X POST 'http://127.0.0.1:8080/jobs/data-refresh' >/dev/null
	curl -fsS -X POST 'http://127.0.0.1:8080/jobs/signals/run' >/dev/null
	curl -fsS 'http://127.0.0.1:8080/jobs/runs?limit=10' >/dev/null

sprint7-check: ml-test backend-test frontend-build sprint7-smoke

sprint7-smoke: sprint6-smoke
	curl -fsS 'http://127.0.0.1:8080/ml/research/overview' >/dev/null
	curl -fsS 'http://127.0.0.1:8080/ml/research/documents' >/dev/null
	curl -fsS 'http://127.0.0.1:8080/ml/research/overview' | jq -e '.grid_matrix.status == "completed" and (.grid_matrix.results | length) == 9 and ([.grid_matrix.results[] | select(.status != "completed")] | length) == 0' >/dev/null
	curl -fsS 'http://127.0.0.1:8080/ml/research/documents' | jq -e 'any(.items[]; .key == "grid_matrix")' >/dev/null
	test -f artifacts/research/sprint7/timeframe_horizon_matrix.json

sprint7-research-run:
	. .venv/bin/activate && cd ml-core && invest-ml raw-coverage-report --data-root ../data --output ../artifacts/research/sprint7/data_coverage.json
	. .venv/bin/activate && cd ml-core && invest-ml run-research-grid --data-root ../data --dataset-output-root ../data/datasets --research-output-root ../artifacts/research/sprint7 --grid-name timeframe_horizon_matrix $$(jq -r '.assets[] | select(.ml_enabled != false) | "--ticker " + .ticker' ../configs/sprint7_universe_v1.json) --factor usdrub --factor brent --factor rtsi --timeframe 5m --timeframe 15m --timeframe 1h --horizon 6 --horizon 12 --horizon 24

sprint8-check: ml-test backend-test frontend-build sprint8-smoke

sprint8-smoke: sprint7-smoke
	@echo "Checking Sprint 8 Neural Sequence model artifacts..."
	test -f data/sequences/dataset_version=sprint8_1h_h24_20260506/window=48/manifest.json
	test -f data/sequences/dataset_version=sprint8_1h_h24_20260506/window=96/manifest.json
	jq -e '.feature_count == 48 and .splits.test.windows > 0 and ([.feature_order[] | select(. == "horizon_bars" or . == "move_pct" or . == "barrier_up_price" or . == "barrier_down_price")] | length) == 0' data/sequences/dataset_version=sprint8_1h_h24_20260506/window=48/manifest.json >/dev/null
	jq -e '.feature_count == 48 and .splits.test.windows > 0 and ([.feature_order[] | select(. == "horizon_bars" or . == "move_pct" or . == "barrier_up_price" or . == "barrier_down_price")] | length) == 0' data/sequences/dataset_version=sprint8_1h_h24_20260506/window=96/manifest.json >/dev/null
	test -f artifacts/models/sprint8_gru_1h_h24_w96/model.pt
	test -f artifacts/models/sprint8_gru_1h_h24_w96/model_manifest.json
	test -f artifacts/models/sprint8_gru_1h_h24_w96/scaler.joblib
	test -f artifacts/models/sprint8_gru_1h_h24_w96/predictions_test.parquet
	jq -e '.model_family == "gru" and .input_window_bars == 96 and .input_tensor_shape == [96, 48] and .export_format == "torchscript" and (.metrics.f1_macro | type) == "number"' artifacts/models/sprint8_gru_1h_h24_w96/model_manifest.json >/dev/null
	curl -fsS 'http://127.0.0.1:8080/ml/models/active' >/dev/null
	curl -fsS 'http://127.0.0.1:8080/ml/models/sprint8_gru_1h_h24_w96' | jq -e '.model_family == "gru" and .input_window_bars == 96 and .runtime_status == "metadata_only"' >/dev/null

sprint9-check: backend-test ml-test frontend-build sprint9-smoke

sprint9-smoke: sprint8-smoke
	@echo "Checking Sprint 9 Operational API..."
	curl -fsS 'http://127.0.0.1:8080/ml/monitoring/summary' | jq -e '.status == "ok" or .status == "warn" or .status == "error"' >/dev/null
	curl -fsS 'http://127.0.0.1:8080/ml/monitoring/summary' | jq -e '.severity | has("info") and has("warning") and has("critical")' >/dev/null
	curl -fsS 'http://127.0.0.1:8080/ml/monitoring/summary' | jq -e '.items | type == "array"' >/dev/null
	status_code=$$(curl -sS -o /tmp/sprint9-activation.json -w '%{http_code}' -X POST 'http://127.0.0.1:8080/ml/models/sprint8_gru_1h_h24_w96/activate'); test "$$status_code" = "403"; jq -e '.error.code == "model_runtime_blocked"' /tmp/sprint9-activation.json >/dev/null
	curl -fsS 'http://127.0.0.1:8080/ml/events?limit=1' | jq -e '.items | length >= 0' >/dev/null
	curl -fsS 'http://127.0.0.1:8080/ml/notifications/rules' | jq -e '.items | length >= 0' >/dev/null
	tmp_rule=$$(mktemp); curl -fsS -X POST 'http://127.0.0.1:8080/ml/notifications/rules' -H 'Content-Type: application/json' -d '{"event_type":"decision_threshold_triggered","severity":"warning","direction":"up","model_version":"baseline_stub_v1","threshold":0.95,"is_enabled":true,"cooldown_minutes":5}' > "$$tmp_rule"; rule_id=$$(jq -r '.id' "$$tmp_rule"); test "$$rule_id" != "" -a "$$rule_id" != "null"; jq -e '.direction == "up" and .model_version == "baseline_stub_v1"' "$$tmp_rule" >/dev/null; curl -fsS -X PATCH "http://127.0.0.1:8080/ml/notifications/rules/$$rule_id" -H 'Content-Type: application/json' -d '{"event_type":"decision_threshold_triggered","severity":"critical","direction":"down","model_version":"baseline_stub_v1","threshold":0.90,"is_enabled":true,"cooldown_minutes":10}' | jq -e '.severity == "critical" and .direction == "down"' >/dev/null; curl -fsS -X DELETE "http://127.0.0.1:8080/ml/notifications/rules/$$rule_id" >/dev/null; rm -f "$$tmp_rule"
	curl -fsS 'http://127.0.0.1:8080/signals/events?asset_id=SBER&limit=10' | jq -e '.items | type == "array"' >/dev/null
	@echo "Sprint 9 smoke tests passed."
