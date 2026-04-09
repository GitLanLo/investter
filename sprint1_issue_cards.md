# Sprint 1: issue/task cards по ролям

## Цель спринта

Собрать первый фундамент системы:

- зафиксировать продуктовые и технические контракты;
- определить новую ML-логику сигнала под проект;
- получить воспроизводимый путь `данные -> фичи -> датасет -> baseline research`;
- поднять каркас backend и service DB;
- подготовить frontend к работе по стабильному API контракту во втором спринте.

## Exit criteria спринта

- утверждены `MVP scope`, `API contract v1`, `data/model contracts v1`;
- есть воспроизводимый ingest и feature pipeline для MVP-инструментов;
- есть новая постановка ML-задачи, target logic, feature map и research protocol;
- обучается хотя бы один сильный baseline-контур на новой логике;
- поднят Go backend skeleton с конфигом, миграциями и service layer foundation;
- frontend имеет согласованные wireframes и контрактную базу под Sprint 2.

## Принципы разбиения

- карточки ниже готовы для переноса в GitHub Issues / Jira;
- `ML` и `backend` составляют критический путь Sprint 1;
- `frontend` в Sprint 1 в основном выполняет подготовительные и контрактные задачи;
- если карточка блокирует другие, это явно указано в `Depends on`.

## Роль ML

### `S1-ML-01` Зафиксировать product-oriented ML objective и смысл сигнала

- Role: `ML`
- Source backlog: `ARCH-01`, `ML-01`
- Priority: `P0`
- Estimate: `S`
- Depends on: нет
- Result:
- документ с формулировкой `что именно предсказывает модель`;
- решение по классовой постановке: `binary` или `up/down/no-trade`;
- фиксированный горизонт прогноза и трактовка `probability of signal`.
- Definition of Done:
- есть Markdown-документ с согласованной постановкой;
- backend и frontend понимают, какой score будут показывать;
- спорных трактовок сигнала не осталось.

### `S1-ML-02` Спроектировать target engineering v1

- Role: `ML`
- Source backlog: `ML-02`
- Priority: `P0`
- Estimate: `M`
- Depends on: `S1-ML-01`
- Result:
- выбор схемы разметки;
- описание anti-leak правил;
- описание neutral/no-trade зоны, если она нужна.
- Scope:
- сравнить `future return threshold`, `triple barrier`, `volatility-adjusted barrier`;
- выбрать рабочую схему для MVP;
- описать формулы, параметры и сдвиги по времени.
- Definition of Done:
- есть спецификация target logic;
- схема пригодна для кодирования в feature pipeline;
- определены параметры, которые будут храниться в конфиге.

### `S1-ML-03` Спроектировать feature space v1 под задачи проекта

- Role: `ML`
- Source backlog: `ML-03`, `FEAT-01`
- Priority: `P0`
- Estimate: `M`
- Depends on: `S1-ML-01`
- Result:
- feature map v1;
- список обязательных групп признаков;
- feature schema contract для data pipeline и inference.
- Scope:
- определить группы `price action`, `volume`, `volatility`, `cross-asset`, `session/calendar`, `regime`;
- разделить mandatory features и stretch features;
- зафиксировать имена колонок и ожидаемые типы.
- Definition of Done:
- утверждён список feature columns v1;
- этот список можно напрямую использовать в `FEAT-02`, `FEAT-05`, `REG-01`.

### `S1-ML-04` Реализовать labeling pipeline v1 без утечек

- Role: `ML`
- Source backlog: `FEAT-03`
- Priority: `P0`
- Estimate: `M`
- Depends on: `S1-ML-02`
- Result:
- код разметки target columns;
- тесты на anti-leak;
- сохранение label columns в feature store.
- Definition of Done:
- target columns строятся автоматически;
- тесты на отсутствие утечек проходят;
- на sample dataset есть валидный label distribution report.

### `S1-ML-05` Реализовать feature build pipeline v1

- Role: `ML`
- Source backlog: `FEAT-02`
- Priority: `P0`
- Estimate: `L`
- Depends on: `S1-ML-03`, `S1-BE-03`, `S1-ML-09`
- Result:
- воспроизводимая сборка feature store по MVP universe;
- merge с cross-assets;
- метаданные запуска и версия схемы.
- Scope:
- очистка и ресемплинг;
- расчёт утверждённых признаков;
- запись parquet в согласованный layout.
- Definition of Done:
- feature build запускается одной командой;
- повторный запуск на одинаковом входе даёт эквивалентный результат;
- output соответствует data contract.

### `S1-ML-06` Реализовать builder train/val/test datasets

- Role: `ML`
- Source backlog: `FEAT-05`, `ML-04`
- Priority: `P0`
- Estimate: `M`
- Depends on: `S1-ML-04`, `S1-ML-05`, `S1-ML-07`
- Result:
- canonical dataset split pipeline;
- manifest со split boundaries;
- purge gap и time-based split policy.
- Definition of Done:
- train/val/test сплиты строятся автоматически;
- у каждого датасета есть manifest и dataset version;
- split logic соответствует research protocol.

### `S1-ML-07` Зафиксировать research protocol и dataset policy

- Role: `ML`
- Source backlog: `ML-04`
- Priority: `P0`
- Estimate: `M`
- Depends on: `S1-ML-02`, `S1-ML-03`
- Result:
- единый протокол оценки моделей;
- правила walk-forward / rolling evaluation;
- policy по calibration и threshold search.
- Definition of Done:
- все модели в Sprint 1 и Sprint 2 сравниваются по одному протоколу;
- определён canonical evaluation dataset.

### `S1-ML-08` Реализовать advanced baseline pack v1

- Role: `ML`
- Source backlog: `ML-05`
- Priority: `P0`
- Estimate: `L`
- Depends on: `S1-ML-06`
- Result:
- минимум 2 сильных baseline-модели;
- сохранённые метрики, предсказания и feature importance;
- baseline report для выбора направления основной модели.
- Definition of Done:
- baseline training воспроизводим по конфигу;
- артефакты логируются и versioned;
- есть сравнительный отчёт по MVP данным.

### `S1-ML-09` Реализовать ingest abstraction и idempotent data path для ML

- Role: `ML`
- Source backlog: `DATA-01`, `DATA-02`
- Priority: `P0`
- Estimate: `L`
- Depends on: `S1-BE-03`
- Result:
- abstraction layer для market data providers;
- безопасная дозагрузка исторических данных;
- raw parquet layout по data contract.
- Definition of Done:
- повторный ingest не создаёт дублей;
- данные пополняются инкрементально;
- raw data пригодны для feature pipeline без ручной чистки.

## Роль Backend

### `S1-BE-01` Зафиксировать системную архитектуру v1

- Role: `Backend`
- Source backlog: `ARCH-02`
- Priority: `P0`
- Estimate: `M`
- Depends on: `S1-ML-01`
- Result:
- компонентная схема системы;
- границы `ml-core`, `backend`, `db`, `frontend`;
- схема локального dev/deploy окружения.
- Definition of Done:
- архитектура описана в Markdown;
- роли модулей и точки интеграции согласованы.

### `S1-BE-02` Подготовить API contract v1 и DTO schemas

- Role: `Backend`
- Source backlog: `ARCH-03`
- Priority: `P0`
- Estimate: `M`
- Depends on: `S1-BE-01`, `S1-ML-01`
- Result:
- OpenAPI draft или эквивалентная спецификация;
- request/response schemas для `assets`, `analysis`, `signals`, `watchlist`;
- единый формат ошибок.
- Definition of Done:
- контракт пригоден для кодогенерации или ручной имплементации;
- frontend может начинать клиентские модели по этой схеме.

### `S1-BE-03` Подготовить data/model contracts v1

- Role: `Backend`
- Source backlog: `ARCH-04`
- Priority: `P0`
- Estimate: `M`
- Depends on: `S1-BE-01`, `S1-ML-03`
- Result:
- parquet layout contract;
- model manifest contract;
- feature schema versioning policy.
- Definition of Done:
- backend и ml-core используют одинаковые соглашения по данным;
- контракт зафикси можетирован в документе  проверяться тестами.

### `S1-BE-04` Подготовить monorepo structure и env conventions

- Role: `Backend`
- Source backlog: `INFRA-01`, `INFRA-02`
- Priority: `P0`
- Estimate: `M`
- Depends on: `S1-BE-01`
- Result:
- корневая структура каталогов;
- `.env.example` и профили среды;
- единая схема запуска для dev.
- Definition of Done:
- репозиторий структурирован по модулям;
- новый разработчик может локально поднять окружение по README.

### `S1-BE-05` Подготовить локальный compose stack

- Role: `Backend`
- Source backlog: `INFRA-03`
- Priority: `P0`
- Estimate: `M`
- Depends on: `S1-BE-04`
- Result:
- локальный `docker compose` стек;
- `postgres` и shared volumes;
- проброс конфигов и переменных окружения.
- Definition of Done:
- стек стартует одной командой;
- backend container может подключиться к БД;
- артефакты и данные доступны через volume policy.

### `S1-BE-06` Создать каркас Go backend service

- Role: `Backend`
- Source backlog: `BE-01`
- Priority: `P0`
- Estimate: `M`
- Depends on: `S1-BE-02`, `S1-BE-04`
- Result:
- структура `cmd`, `internal`, `pkg`;
- конфиг, логирование, graceful shutdown;
- `health`/`readiness` endpoints;
- базовый router и middleware.
- Definition of Done:
- сервис компилируется и стартует локально;
- есть каркас для последующей имплементации API.

### `S1-BE-07` Спроектировать и завести service DB schema

- Role: `Backend`
- Source backlog: `BE-02`
- Priority: `P0`
- Estimate: `L`
- Depends on: `S1-BE-03`, `S1-BE-06`
- Result:
- миграции для `assets`, `watchlists`, `watchlist_items`, `notification_rules`, `signal_runs`, `signal_events`, `model_registry`, `job_runs`;
- индексы и базовые constraints.
- Definition of Done:
- миграции применяются автоматически;
- схема покрывает сущности MVP Sprint 1/2.

### `S1-BE-08` Реализовать repository/service layer foundation

- Role: `Backend`
- Source backlog: `BE-03`
- Priority: `P0`
- Estimate: `L`
- Depends on: `S1-BE-07`
- Result:
- repository interfaces;
- базовые реализации для `assets`, `signals`, `watchlist`, `models`;
- service layer без бизнес-логики в HTTP handlers.
- Definition of Done:
- есть unit-testable foundation для API Sprint 2;
- SQL не размазан по коду обработчиков.

## Роль Frontend

### Важное замечание по роли Frontend в Sprint 1

- в первом спринте frontend не должен блокировать ML/backend критический путь;
- задача frontend-команды здесь: зафиксировать UX и контрактную готовность под Sprint 2;
- реализация полноценного UI начинается во втором спринте.

### `S1-FE-01` Разобрать пользовательские сценарии и подготовить screen map

- Role: `Frontend`
- Source backlog: `ARCH-01`, `ARCH-03`
- Priority: `P0`
- Estimate: `S`
- Depends on: `S1-BE-02`
- Result:
- screen map для `dashboard`, `asset details`, `watchlist`, `notification settings`;
- список данных, нужных каждому экрану.
- Definition of Done:
- backend знает, какие поля реально нужны UI;
- нет скрытых UI-требований, всплывающих во втором спринте.

### `S1-FE-02` Подготовить wireframes и states для MVP интерфейсов

- Role: `Frontend`
- Source backlog: подготовка к `FE-02`, `FE-03`, `FE-04`
- Priority: `P0`
- Estimate: `M`
- Depends on: `S1-FE-01`, `S1-BE-02`
- Result:
- wireframes главного экрана и страницы инструмента;
- состояния `loading`, `empty`, `error`, `stale data`, `no signal`.
- Definition of Done:
- wireframes согласованы;
- Sprint 2 можно начинать без дополнительных UX-договорённостей.

### `S1-FE-03` Подготовить frontend domain models и API mapping

- Role: `Frontend`
- Source backlog: подготовка к `FE-01`
- Priority: `P0`
- Estimate: `S`
- Depends on: `S1-BE-02`
- Result:
- таблица соответствия frontend models <-> API DTO;
- список обязательных и optional полей;
- список вопросов к backend contract.
- Definition of Done:
- frontend готов к реализации typed API client в Sprint 2;
- все спорные поля подняты до старта реализации UI.

### `S1-FE-04` Подготовить app shell и техническое решение фронтенда

- Role: `Frontend`
- Source backlog: подготовка к `FE-01`
- Priority: `P1`
- Estimate: `M`
- Depends on: `S1-FE-03`, `S1-BE-05`
- Result:
- выбран стек frontend;
- создан пустой каркас приложения;
- настроены env handling, routing skeleton и базовая структура каталогов.
- Definition of Done:
- frontend project стартует локально;
- во втором спринте можно сразу переходить к экрану dashboard.

## Зависимости между ролями

### ML получает блокеры от Backend

- `S1-ML-05` зависит от `S1-BE-03`, потому что feature pipeline должен писать по зафиксированному data contract.
- `S1-ML-09` зависит от `S1-BE-03`, потому что raw ingest layout должен совпадать с общей схемой данных.

### Backend получает блокеры от ML

- `S1-BE-01` и `S1-BE-02` зависят от `S1-ML-01`, потому что API должен отражать реальную семантику сигнала.
- `S1-BE-03` зависит от `S1-ML-03`, потому что model/data contracts включают feature schema v1.

### Frontend получает блокеры от Backend

- почти все frontend cards стартуют после `S1-BE-02`, потому что без API contract wireframes и DTO mapping будут недостоверны.

## Рекомендуемый порядок в рамках спринта

### Неделя 1

- `S1-ML-01`
- `S1-ML-02`
- `S1-ML-03`
- `S1-BE-01`
- `S1-BE-02`
- `S1-BE-03`
- `S1-BE-04`
- `S1-FE-01`

### Неделя 2

- `S1-BE-05`
- `S1-BE-06`
- `S1-BE-07`
- `S1-BE-08`
- `S1-ML-09`
- `S1-ML-04`
- `S1-ML-05`
- `S1-ML-07`
- `S1-FE-02`
- `S1-FE-03`

### Неделя 3

- `S1-ML-06`
- `S1-ML-08`
- `S1-FE-04`
- bugfix / stabilization / sprint review materials

## Минимальный must-have для успешного закрытия Sprint 1

- `S1-ML-01`
- `S1-ML-02`
- `S1-ML-03`
- `S1-ML-04`
- `S1-ML-05`
- `S1-ML-06`
- `S1-ML-07`
- `S1-ML-08`
- `S1-BE-01`
- `S1-BE-02`
- `S1-BE-03`
- `S1-BE-04`
- `S1-BE-05`
- `S1-BE-06`
- `S1-BE-07`
- `S1-BE-08`

## Stretch для Sprint 1

- `S1-FE-04`
- дополнительные baseline-модели сверх минимального пакета;
- автоматические QA-отчёты по feature store, если хватает времени.
