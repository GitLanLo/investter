# Sprint 3 Plan

Sprint 3 фокусируется на production policy и переходе от research cockpit к контролируемому runtime-кандидату.

## Goals

- Нормализовать production-candidate policy поверх Sprint 2 research/calibration artifacts.
- Показать policy quality в GUI без чтения raw JSON пользователем.
- Подготовить backend contract для следующего шага: live validation loop and production-candidate promotion.
- Сохранить воспроизводимую проверку через `make sprint3-check`.

## Initial Scope

- `GET /ml/policy/production` returns model, scenario, calibration method, threshold, dataset metadata and validation/test metrics.
- Frontend Research Board includes a Sprint 3 policy card.
- Smoke-check extends Sprint 2 verification with production policy endpoint.

## Next Slice

- Add live validation run records for candidate policy.
- Add policy decision states: `candidate`, `shadow_live`, `promoted`, `blocked`.
- Attach policy status to `/analysis/run` responses.
- Add GUI comparison between current live signal and selected production policy.

## Exit Criteria

- `make sprint3-check` passes against a running local stack.
- GUI shows policy identity and core reliability metrics.
- Backend returns a stable production policy contract independent of raw artifact document layout.
