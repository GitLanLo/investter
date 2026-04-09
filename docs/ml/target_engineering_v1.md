# Target Engineering v1

## Цель

Определить схему разметки, которая:

- отражает именно `actionable directional signal`;
- устойчива к рыночному шуму;
- не допускает утечки будущего;
- пригодна для online inference semantics, принятой в `ML Objective v1`.

## Выбранная схема

Для MVP принимается `volatility-aware triple barrier` с тремя исходами:

- `up_signal`
- `down_signal`
- `no_trade`

## Базовые параметры

- base timeframe: `5m`
- horizon bars: `12`
- horizon duration: `60 minutes`
- price reference at time `t`: `close_t`
- volatility reference: `ATR(14)` или эквивалентный устойчивый volatility proxy, рассчитанный только по данным `<= t`

## Формирование барьеров

Для наблюдения в момент `t` рассчитывается динамический порог движения:

- `move_pct_t = max(min_move_pct, atr_mult * atr14_t / close_t)`

Стартовые значения MVP:

- `min_move_pct = 0.003` (`0.3%`)
- `atr_mult = 1.0`

Далее строятся барьеры:

- upper barrier: `close_t * (1 + move_pct_t)`
- lower barrier: `close_t * (1 - move_pct_t)`
- time barrier: `t + H`

## Правило присвоения класса

### `up_signal`

Если в интервале `(t, t + H]` верхний барьер достигнут раньше нижнего.

### `down_signal`

Если в интервале `(t, t + H]` нижний барьер достигнут раньше верхнего.

### `no_trade`

Если:

- ни один барьер не достигнут до `t + H`;
- оба барьера достигнуты неоднозначно в одном и том же баре;
- наблюдение попадает в ambiguous case, который повышает label noise.

Для MVP ambiguity policy фиксируется как:

- `ambiguous -> no_trade`

Это решение снижает шум разметки и делает product-level signal более консервативным.

## Анти-утечка

Обязательные правила:

- все feature values строятся как состояние на момент `t`;
- volatility proxy и все derived inputs используют только историю `<= t`;
- label смотрит только в окно `(t, t + H]`;
- split между `train`, `val`, `test` должен включать purge gap не меньше горизонта;
- любые rolling statistics не должны использовать будущие значения.

## Служебные колонки разметки

В feature store и/или dataset manifest должны появляться:

- `label_class`
- `label_up`
- `label_down`
- `label_no_trade`
- `horizon_bars`
- `move_pct`
- `barrier_up_price`
- `barrier_down_price`
- `label_asof_time`
- `label_horizon_end_time`

## Почему схема подходит проекту

- учитывает рыночный шум через dynamic threshold;
- лучше переносится между режимами рынка, чем фиксированный абсолютный порог;
- поддерживает нейтральный режим для monitoring product;
- естественно маппится на `up/down/neutral` UI semantics.

## Что подлежит дальнейшей настройке

- `min_move_pct`
- `atr_mult`
- `horizon_bars`
- ambiguity handling policy, если empirical validation покажет пользу иной схемы

