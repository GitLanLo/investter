# Tinkoff Invest API Integration

## Цель

Подготовить проект к загрузке исторических рыночных данных из актуального Tinkoff Invest API в contract-aligned raw layout проекта.

## Принятые решения

- интеграция построена через актуальную REST proxy нового Tinkoff Invest API;
- legacy `openAPI` ветка не используется;
- данные складываются в существующий layout:
  - `data/raw/candles/...`
  - `data/raw/factors/...`
- downstream pipeline остаётся тем же:
  - `materialize-feature-store`
  - `materialize-dataset`

## Переменные окружения

- `TINKOFF_INVEST_TOKEN`
- `TINKOFF_INVEST_TARGET`
  - `prod`
  - `sandbox`
- `TINKOFF_INVEST_TIMEOUT_SECONDS`
- `TINKOFF_INVEST_USER_AGENT`

## CLI команды

### Asset candles

```sh
invest-ml tinkoff-sync-asset \
  --data-root ../data \
  --ticker SBER \
  --from 2026-04-01T07:00:00Z \
  --to 2026-04-02T07:00:00Z \
  --timeframe 5m \
  --instrument-kind share \
  --class-code TQBR
```

### Factor candles

```sh
invest-ml tinkoff-sync-factor \
  --data-root ../data \
  --alias usdrub \
  --ticker USD000UTSTOM \
  --from 2026-04-01T07:00:00Z \
  --to 2026-04-02T07:00:00Z \
  --timeframe 5m \
  --instrument-kind currency \
  --class-code CETS
```

### Universe sync

```sh
invest-ml tinkoff-sync-universe \
  --config configs/mvp_universe_v1.json \
  --data-root data \
  --from 2026-03-30T07:00:00Z \
  --to 2026-04-09T07:00:00Z
```

## Что делает интеграция

1. Ищет инструмент через `InstrumentsService/FindInstrument`.
2. Резолвит точный инструмент по `ticker` и при необходимости `class_code`.
3. Учитывает `first1minCandleDate` или `first1dayCandleDate`.
4. Чанкует запросы свечей по ограничениям API для выбранного таймфрейма.
5. Загружает свечи через `MarketDataService/GetCandles`.
6. Валидирует, что incremental state относится к тому же `uid/figi/class_code`, и не даёт смешать другой инструмент в тот же raw path.
7. Сохраняет watermark последней записанной свечи даже если очередной incremental-run не принёс новых данных.
8. Складывает результат в raw parquet layout проекта.

## Ограничения

- для `5m` максимальный диапазон одного запроса API ограничен одним днём, поэтому интеграция автоматически режет диапазон на чанки;
- если по тикеру найдено несколько инструментов, нужно передать `--class-code`;
- если нужно сознательно перемапить тот же `ticker` на другой инструмент, сначала нужно очистить соответствующий `ingest_state` и raw parquet partition;
- для factor aliases необходимо явно задавать соответствующий биржевой тикер Tinkoff.

## MVP factor notes

- `usdrub` использует спот `USD000UTSTOM`;
- `brent` и `rtsi` в `mvp_universe_v1` заведены через front futures proxies (`BRM6`, `RIM6`);
- для этих срочных факторов в следующем шаге нужен explicit rollover policy, иначе alias останется корректным, а тикер со временем устареет.
