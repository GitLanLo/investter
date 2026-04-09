# Feature Space v1

## Цель

Определить feature space, специально адаптированное под задачу проекта:

- directional signal detection;
- учёт внешних факторов;
- учёт режимов рынка;
- пригодность для sequence model и сильных tabular baselines.

## Принципы

- сначала включаем признаки, которые реально полезны для monitoring signal task;
- не переносим в проект все признаки из reference repo без проверки;
- mandatory features должны быть совместимы с training и production inference;
- порядок feature columns обязан фиксироваться в model manifest.

## Группы признаков v1

### 1. Price action

Обязательные:

- `ret_1`
- `ret_3`
- `ret_6`
- `ret_12`
- `ret_24`
- `log_ret_1`
- `close_to_prev_close`
- `hl_range_pct`
- `oc_range_pct`
- `close_to_day_open_pct`

### 2. Trend and momentum

Обязательные:

- `ema_12_dist`
- `ema_26_dist`
- `ema_60_dist`
- `sma_20_dist`
- `sma_60_dist`
- `rsi_7`
- `rsi_14`
- `momentum_12`
- `momentum_24`

### 3. Volatility and risk

Обязательные:

- `atr_14_pct`
- `realized_vol_12`
- `realized_vol_24`
- `bb_width_20`
- `range_zscore_24`

### 4. Volume and liquidity

Обязательные:

- `volume_rel_12`
- `volume_rel_24`
- `volume_zscore_24`
- `turnover_proxy`
- `volume_price_trend_component`

### 5. Session and calendar context

Обязательные:

- `minute_of_session_sin`
- `minute_of_session_cos`
- `day_of_week_sin`
- `day_of_week_cos`
- `is_opening_window`
- `is_closing_window`
- `is_evening_session`

### 6. Cross-asset context

Обязательные:

- `usdrub_ret_1`
- `usdrub_ret_6`
- `brent_ret_1`
- `brent_ret_6`
- `rtsi_ret_1`
- `rtsi_ret_6`
- `asset_vs_rtsi_rel_strength_12`
- `asset_vs_brent_rel_strength_12`

### 7. Regime features

Обязательные:

- `vol_regime_flag`
- `trend_regime_flag`
- `market_stress_proxy`
- `cross_asset_dispersion_proxy`

## Stretch features

Эти признаки не блокируют MVP, но могут быть добавлены после first stable baseline:

- rolling correlation to `RTSI`;
- rolling beta to market proxy;
- candle pattern embeddings;
- microstructure-like proxies на 5m данных;
- feature interactions для regime-aware tabular models.

## Feature schema contract

Для каждой строки feature store обязательно хранятся:

- `asof_time`
- `ticker`
- `timeframe`
- `feature_schema_version`
- все mandatory feature columns
- label columns из `target_engineering_v1`

## Требования к production compatibility

- набор и порядок feature columns фиксируются в model manifest;
- normalization parameters, если используются, сохраняются как отдельный artifact;
- inference pipeline не может добавлять или пропускать колонки вне manifest.

## Минимальный mandatory feature set для Sprint 1

Sprint 1 считается успешным, если реализованы:

- все группы `Price action`, `Trend and momentum`, `Volatility and risk`;
- минимум по 2-4 ключевых признака из `Volume`, `Cross-asset`, `Session`;
- хотя бы один рабочий `regime` proxy.

