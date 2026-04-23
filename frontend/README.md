# frontend

Frontend workspace для Sprint 2 operator GUI.

## Что есть сейчас

- `Vite + React + TypeScript` app shell;
- fallback на mock data, если backend недоступен;
- typed API/domain mapping в `src/lib/api.ts`;
- interactive candle workbench:
  - zoom / pan
  - brush-selection по диапазону
  - timeframe switch
  - detached chart window;
- synced factor strip и signal overlay;
- artifact browser для latest dataset/research/calibration documents;
- policy validation ledger для сохранения production policy snapshots перед promotion;
- asset signal history panel;
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

## Sprint 2 scope

- operator GUI работает поверх реального backend API;
- chart/workbench уже связан с ML artifact layer и signal history;
- при недоступности backend UI остаётся функциональным через mock fallback;
- полноценные auth, notifications workflow и multi-run artifact navigation остаются следующим слоем.
