# ML Objective v1

## Контекст

Текущий `investML` используется как reference baseline по данным, экспериментам и pipeline patterns. Финальная ML-логика проекта проектируется заново под задачу мониторинга и оценки вероятности сигнала в веб-системе.

## Принятое решение

Для MVP задача формулируется как `3-class classification`:

- `up_signal`
- `down_signal`
- `no_trade`

Причина выбора такой постановки:

- системе нужен не просто прогноз направления, а отделение `actionable signal` от рыночного шума;
- для UI и уведомлений нужен нейтральный режим, в котором система честно сообщает, что надёжного сигнала нет;
- такая схема лучше соответствует мониторинговому продукту, чем чисто бинарная классификация.

## Базовая временная конфигурация MVP

- базовый таймфрейм модели: `5m`
- горизонт прогноза MVP: `12` баров
- интерпретация горизонта: `60 минут` при `5m` данных

В последующих итерациях можно расширить модель до multi-horizon, но в MVP фиксируется один основной горизонт.

## Что предсказывает модель

Модель оценивает вероятность того, что после момента `t` инструмент в пределах горизонта `H` покажет значимое направленное движение вверх или вниз, достаточное для признания события сигналом.

Следовательно:

- `p_up` — вероятность значимого направленного движения вверх;
- `p_down` — вероятность значимого направленного движения вниз;
- `p_no_trade` — вероятность отсутствия качественного направленного сигнала.

## Продуктовая семантика `probability of signal`

Для backend, frontend и notification subsystem вводится следующая семантика:

- `signal_probability = max(p_up, p_down)`
- `signal_direction = up`, если `p_up > p_down` и выполнено условие actionable signal
- `signal_direction = down`, если `p_down > p_up` и выполнено условие actionable signal
- `signal_direction = neutral`, если лучшим классом является `no_trade` или directional confidence ниже порога

Отдельно система хранит полный набор классовых вероятностей:

- `class_probabilities.up`
- `class_probabilities.down`
- `class_probabilities.no_trade`

## Правило actionable signal для MVP

Событие считается actionable для UI и уведомлений, если одновременно выполнены условия:

1. `argmax` по классам принадлежит одному из направлений `up_signal` или `down_signal`
2. соответствующая directional probability выше порога `decision_threshold`

Стартовое значение `decision_threshold` для MVP:

- `0.65`

Это значение фиксируется как начальное product-default и затем может быть скорректировано по результатам validation.

## Как score используется в системе

### Backend

- возвращает полный набор классовых вероятностей;
- вычисляет `signal_probability`;
- вычисляет итоговый `signal_direction`;
- сохраняет `signal_state = actionable | neutral`.

### Frontend

- показывает `signal_probability`;
- показывает направление сигнала;
- показывает состояние `actionable` или `neutral`;
- может отображать breakdown по классам на detail page.

### Notifications

- уведомления срабатывают только по `actionable` сигналам;
- правила уведомлений сравнивают directional confidence с пользовательским порогом;
- `neutral` состояние уведомлений не создаёт.

## Что не делаем в MVP

- не строим торговую систему исполнения сделок;
- не используем ML-выход как гарантию результата;
- не делаем одновременно несколько разных product-level целей;
- не смешиваем offline research metric и user-facing signal semantics.

## Связанные документы

- `docs/ml/target_engineering_v1.md`
- `docs/ml/feature_space_v1.md`
- `docs/ml/research_protocol_v1.md`
- `docs/contracts/api_v1.md`
- `docs/contracts/data_model_contracts_v1.md`

## Результат Sprint 1

Этот документ считается завершённым, если:

- ML, backend и frontend одинаково понимают смысл сигнала;
- выбранная постановка пригодна для target engineering и online inference;
- API и UI могут использовать единый продуктовый contract без дополнительных трактовок.
