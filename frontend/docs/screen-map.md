# Frontend Screen Map v1

## Основные экраны

1. `Dashboard`
   Назначение: обзор состояния universe, последние сигналы, быстрый переход к деталям инструмента.
   Источники: `GET /assets`, `GET /signals/latest`.

2. `Asset Details`
   Назначение: карточка одного инструмента, breakdown вероятностей, история последних прогонов модели.
   Источники: текущий `signals/latest`, `GET /assets/{id}/signals`, policy snapshot внутри signal DTO.

3. `Watchlist`
   Назначение: список активов, порядок и статусы слежения.
   Источники: `GET /watchlist/default`.

4. `Notification Rules`
   Назначение: конфигурация trigger rules и cooldown.
   Источники: Sprint 2 API для notification rules.

5. `Policy Gate`
   Назначение: фиксация текущего production policy snapshot и операторский перевод candidate в shadow/live/promotion states.
   Источники: `GET /ml/policy/validation-runs`, `POST /ml/policy/validation-runs`, `PATCH /ml/policy/validation-runs/{id}`, `GET /ml/policy/shadow-summary`.

## Навигационные переходы

- `Dashboard -> Asset Details`
- `Dashboard -> Watchlist`
- `Watchlist -> Asset Details`
- `Asset Details -> Notification Rules`
- `Research Board -> Policy Gate`

## Sprint 1 scope

- экранная карта фиксирует IA и contract points;
- реальная business-логика редактирования правил переносится в Sprint 2;
- frontend shell должен уметь жить на mock data и на живом backend contract.
