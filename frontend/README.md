# frontend

Frontend workspace для Sprint 1.

## Что есть сейчас

- `Vite + React + TypeScript` app shell;
- fallback на mock data, если backend недоступен;
- typed API/domain mapping в `src/lib/api.ts`;
- prep pack в `docs/`:
  - `screen-map.md`
  - `wireframes.md`
  - `api-domain-models.md`

## Локальный запуск

```sh
npm install
npm run dev -- --host 0.0.0.0 --port 5173
```

Для сборки:

```sh
npm run build
```

## Переменные окружения

- `VITE_API_BASE_URL` — базовый URL backend API, по умолчанию `http://localhost:8080`.

## Sprint 1 scope

- app shell стартует локально и не блокирует ML/backend;
- IA, wireframes и DTO mapping зафиксированы под Sprint 2;
- production-grade routing и полноценный typed client откладываются на следующий спринт.
