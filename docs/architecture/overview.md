# Architecture Overview v1

## Цель

Зафиксировать рабочую архитектуру системы и границы модулей.

## Контуры системы

- `ml-core`
  отвечает за ingest, feature engineering, target engineering, datasets, training, evaluation, ONNX export.
- `backend`
  отвечает за REST API, jobs, model loading, signal history, watchlist и notification rules.
- `service database`
  хранит системные сущности и историю событий.
- `parquet/data lake`
  хранит raw market data, cross-factors, features, datasets и model-related artifacts.
- `frontend`
  отвечает за dashboard, asset details, watchlist и управление правилами мониторинга.

## Основной поток данных

1. `ml-core` загружает market data и cross-factors.
2. `ml-core` строит features и datasets по согласованным contracts.
3. `ml-core` обучает модели и публикует model artifacts.
4. `backend` использует model manifest и later ONNX artifact для online inference.
5. `backend` публикует signal history и latest ML artifacts для operator GUI.
6. `frontend` запрашивает API backend и отображает signal state, history и artifact drill-down.

## Принятая product semantics сигнала

- модель рассматривается как `up / down / no_trade`;
- backend публикует `signal_probability`, `signal_direction` и `signal_state`;
- уведомления срабатывают только на actionable signals;
- frontend показывает и итоговый статус, и вероятностный breakdown на detail page.

## Текущая рабочая конфигурация

- интерактивный operator GUI поверх реального backend API;
- signal history по активу и latest signal tape;
- ML artifact overview и document browser;
- research stack: ablation, walk-forward, production gate, calibration audit;
- рабочий вертикальный срез `ml-core -> backend -> frontend`.
