# Sprint 1 GitHub Issues Drafts

## Как использовать

- каждый блок ниже можно переносить в отдельный GitHub Issue;
- `Title` использовать как заголовок issue;
- `Labels` можно заранее завести в репозитории и потом назначать при создании;
- `Body` уже оформлен в GitHub-flavored Markdown;
- `Depends on` не является нативным полем GitHub Issues, его стоит вставлять в тело issue или отражать через linked issues.

## Suggested labels

- `sprint:1`
- `role:ml`
- `role:backend`
- `role:frontend`
- `type:task`
- `priority:p0`
- `priority:p1`

---

## 1. S1-ML-01

**Title**

`[Sprint 1][ML] Зафиксировать product-oriented ML objective и смысл сигнала`

**Labels**

`sprint:1`, `role:ml`, `type:task`, `priority:p0`

**Depends on**

`none`

**Body**

```md
## Summary

Зафиксировать, что именно предсказывает ML-модель в продукте и как трактуется `probability of signal`.

## Scope

- определить, что означает `вероятность сигнала` для системы;
- решить, остаётся ли постановка бинарной или переходит в `up/down/no-trade`;
- зафиксировать горизонт(ы) прогноза;
- описать, как score используется в API, UI и notification rules.

## Deliverables

- Markdown-документ с product-oriented ML objective;
- согласованная трактовка score для команды ML, backend и frontend.

## Acceptance Criteria

- [ ] есть документ с формулировкой ML objective;
- [ ] определена классовая постановка задачи;
- [ ] определён горизонт прогноза;
- [ ] нет спорных трактовок смысла сигнала.
```

---

## 2. S1-ML-02

**Title**

`[Sprint 1][ML] Спроектировать target engineering v1`

**Labels**

`sprint:1`, `role:ml`, `type:task`, `priority:p0`

**Depends on**

`S1-ML-01`

**Body**

```md
## Summary

Спроектировать новую схему разметки target под задачи проекта.

## Scope

- сравнить варианты `future return threshold`, `triple barrier`, `volatility-adjusted barrier`;
- выбрать рабочую схему для MVP;
- описать anti-leak правила;
- определить neutral/no-trade зону, если она нужна;
- зафиксировать параметры и временные сдвиги.

## Deliverables

- спецификация target logic v1;
- набор параметров, выносимых в конфиг.

## Acceptance Criteria

- [ ] выбрана схема разметки;
- [ ] anti-leak требования описаны;
- [ ] параметры схемы зафиксированы;
- [ ] схема пригодна для реализации в pipeline.
```

---

## 3. S1-ML-03

**Title**

`[Sprint 1][ML] Спроектировать feature space v1 под задачи проекта`

**Labels**

`sprint:1`, `role:ml`, `type:task`, `priority:p0`

**Depends on**

`S1-ML-01`

**Body**

```md
## Summary

Определить feature space v1, адаптированное под задачу проекта, а не унаследованное из makeup как есть.

## Scope

- определить группы признаков `price action`, `volume`, `volatility`, `cross-asset`, `session/calendar`, `regime`;
- разделить mandatory и stretch features;
- зафиксировать naming convention, типы и ожидаемую схему колонок;
- подготовить feature map для downstream pipeline.

## Deliverables

- feature map v1;
- список обязательных feature columns;
- feature schema contract для training и inference.

## Acceptance Criteria

- [ ] утверждён список feature columns v1;
- [ ] схема колонок и типы описаны;
- [ ] feature map пригоден для `FEAT-02`, `FEAT-05`, `REG-01`.
```

---

## 4. S1-ML-04

**Title**

`[Sprint 1][ML] Реализовать labeling pipeline v1 без утечек`

**Labels**

`sprint:1`, `role:ml`, `type:task`, `priority:p0`

**Depends on**

`S1-ML-02`

**Body**

```md
## Summary

Реализовать генерацию target columns по новой схеме разметки и покрыть её anti-leak тестами.

## Scope

- реализовать target/label columns;
- добавить тесты на утечку будущего;
- встроить labeling в feature pipeline;
- проверить базовое распределение классов на sample dataset.

## Deliverables

- код labeling pipeline;
- anti-leak tests;
- sample label distribution report.

## Acceptance Criteria

- [ ] target columns строятся автоматически;
- [ ] anti-leak тесты проходят;
- [ ] labels сохраняются в feature store;
- [ ] на sample dataset распределение классов валидно и объяснимо.
```

---

## 5. S1-ML-05

**Title**

`[Sprint 1][ML] Реализовать feature build pipeline v1`

**Labels**

`sprint:1`, `role:ml`, `type:task`, `priority:p0`

**Depends on**

`S1-ML-03`, `S1-BE-03`, `S1-ML-09`

**Body**

```md
## Summary

Собрать воспроизводимый feature build pipeline по MVP universe.

## Scope

- очистка и ресемплинг данных;
- merge с cross-assets;
- расчёт утверждённых признаков;
- сохранение feature parquet в согласованном layout;
- запись metadata о запуске и версии схемы.

## Deliverables

- reproducible feature build command;
- feature parquet layout v1;
- run metadata.

## Acceptance Criteria

- [ ] feature build запускается одной командой;
- [ ] повторный запуск на одинаковом входе даёт эквивалентный результат;
- [ ] output соответствует data contract;
- [ ] cross-asset merge работает по MVP-факторам.
```

---

## 6. S1-ML-06

**Title**

`[Sprint 1][ML] Реализовать builder train/val/test datasets`

**Labels**

`sprint:1`, `role:ml`, `type:task`, `priority:p0`

**Depends on**

`S1-ML-04`, `S1-ML-05`, `S1-ML-07`

**Body**

```md
## Summary

Построить canonical train/val/test datasets по research protocol.

## Scope

- реализовать time-based split;
- реализовать purge gap policy;
- сохранять dataset manifest;
- фиксировать границы периодов и dataset version.

## Deliverables

- dataset builder v1;
- manifest для каждого набора;
- документированные split boundaries.

## Acceptance Criteria

- [ ] train/val/test сплиты строятся автоматически;
- [ ] у каждого датасета есть manifest;
- [ ] split logic соответствует research protocol;
- [ ] leakage policy соблюдена.
```

---

## 7. S1-ML-07

**Title**

`[Sprint 1][ML] Зафиксировать research protocol и dataset policy`

**Labels**

`sprint:1`, `role:ml`, `type:task`, `priority:p0`

**Depends on**

`S1-ML-02`, `S1-ML-03`

**Body**

```md
## Summary

Определить единый протокол оценки моделей и canonical policy по датасетам.

## Scope

- определить rolling/walk-forward evaluation;
- определить policy для calibration и threshold search;
- определить canonical evaluation dataset;
- описать, как сравниваются модели по периодам, инструментам и режимам рынка.

## Deliverables

- research protocol v1;
- dataset policy v1.

## Acceptance Criteria

- [ ] все модели сравниваются по одному протоколу;
- [ ] зафиксирован canonical evaluation dataset;
- [ ] protocol подходит для Sprint 1 и Sprint 2.
```

---

## 8. S1-ML-08

**Title**

`[Sprint 1][ML] Реализовать advanced baseline pack v1`

**Labels**

`sprint:1`, `role:ml`, `type:task`, `priority:p0`

**Depends on**

`S1-ML-06`

**Body**

```md
## Summary

Собрать воспроизводимый пакет сильных baseline-моделей на новой логике сигнала.

## Scope

- реализовать минимум 2 baseline-модели;
- добавить probability calibration при необходимости;
- сохранять метрики, предсказания и feature importance;
- подготовить baseline report.

## Deliverables

- reproducible baseline training configs;
- baseline artifacts;
- сравнительный baseline report.

## Acceptance Criteria

- [ ] baseline training воспроизводим по конфигу;
- [ ] есть минимум 2 baseline-модели;
- [ ] артефакты логируются и versioned;
- [ ] есть сравнительный отчёт по MVP данным.
```

---

## 9. S1-ML-09

**Title**

`[Sprint 1][ML] Реализовать ingest abstraction и idempotent data path`

**Labels**

`sprint:1`, `role:ml`, `type:task`, `priority:p0`

**Depends on**

`S1-BE-03`

**Body**

```md
## Summary

Построить abstraction layer для market data providers и безопасный idempotent ingest path.

## Scope

- выделить abstraction для market data source;
- реализовать инкрементальную дозагрузку истории;
- убрать дубликаты и проблемы повторных запусков;
- сохранять raw parquet по data contract.

## Deliverables

- ingest abstraction;
- idempotent historical loader;
- raw parquet layout v1.

## Acceptance Criteria

- [ ] повторный ingest не создаёт дублей;
- [ ] данные пополняются инкрементально;
- [ ] raw data пригодны для feature pipeline без ручной чистки;
- [ ] layout соответствует data contract.
```

---

## 10. S1-BE-01

**Title**

`[Sprint 1][Backend] Зафиксировать системную архитектуру v1`

**Labels**

`sprint:1`, `role:backend`, `type:task`, `priority:p0`

**Depends on**

`S1-ML-01`

**Body**

```md
## Summary

Подготовить архитектурную схему системы и зафиксировать границы модулей.

## Scope

- описать `ml-core`, `backend`, `db`, `frontend`;
- зафиксировать точки интеграции;
- описать dev/deploy setup для локальной разработки.

## Deliverables

- architecture document v1;
- component diagram;
- integration boundaries.

## Acceptance Criteria

- [ ] архитектура описана в Markdown;
- [ ] модули и их ответственность согласованы;
- [ ] точки интеграции понятны backend, ML и frontend.
```

---

## 11. S1-BE-02

**Title**

`[Sprint 1][Backend] Подготовить API contract v1 и DTO schemas`

**Labels**

`sprint:1`, `role:backend`, `type:task`, `priority:p0`

**Depends on**

`S1-BE-01`, `S1-ML-01`

**Body**

```md
## Summary

Подготовить API contract v1 для backend/frontend integration.

## Scope

- описать request/response schemas для `assets`, `analysis`, `signals`, `watchlist`;
- определить единый формат ошибок;
- зафиксировать временные форматы, обязательные поля и семантику score.

## Deliverables

- OpenAPI draft или эквивалентный API spec;
- DTO schema pack.

## Acceptance Criteria

- [ ] backend и frontend используют один контракт;
- [ ] обязательные поля и ошибки описаны;
- [ ] контракт пригоден для реализации API во втором спринте.
```

---

## 12. S1-BE-03

**Title**

`[Sprint 1][Backend] Подготовить data/model contracts v1`

**Labels**

`sprint:1`, `role:backend`, `type:task`, `priority:p0`

**Depends on**

`S1-BE-01`, `S1-ML-03`

**Body**

```md
## Summary

Зафиксировать общий contract по данным и model artifacts для backend и ML.

## Scope

- описать parquet layout для raw/features/datasets;
- описать model manifest contract;
- зафиксировать feature schema versioning policy.

## Deliverables

- data contract v1;
- model contract v1;
- versioning policy.

## Acceptance Criteria

- [ ] backend и ml-core используют одинаковые соглашения;
- [ ] contract описан и versioned;
- [ ] этот contract пригоден для `feature build`, `datasets`, `ONNX`, `backend loading`.
```

---

## 13. S1-BE-04

**Title**

`[Sprint 1][Backend] Подготовить monorepo structure и env conventions`

**Labels**

`sprint:1`, `role:backend`, `type:task`, `priority:p0`

**Depends on**

`S1-BE-01`

**Body**

```md
## Summary

Подготовить структуру репозитория и conventions по окружению.

## Scope

- создать базовую структуру каталогов;
- добавить `.env.example`;
- описать профили среды;
- подготовить единый developer entrypoint.

## Deliverables

- monorepo structure;
- env examples;
- developer bootstrap instructions.

## Acceptance Criteria

- [ ] репозиторий структурирован по модулям;
- [ ] локальный запуск документирован;
- [ ] обязательные env vars перечислены.
```

---

## 14. S1-BE-05

**Title**

`[Sprint 1][Backend] Подготовить локальный compose stack`

**Labels**

`sprint:1`, `role:backend`, `type:task`, `priority:p0`

**Depends on**

`S1-BE-04`

**Body**

```md
## Summary

Собрать локальный docker compose stack для Sprint 1.

## Scope

- добавить `postgres`;
- определить volumes для данных и артефактов;
- пробросить env/config;
- проверить локальный bootstrap.

## Deliverables

- `docker compose` stack;
- volume policy;
- startup instructions.

## Acceptance Criteria

- [ ] стек стартует одной командой;
- [ ] backend container видит БД;
- [ ] данные и артефакты доступны через volumes.
```

---

## 15. S1-BE-06

**Title**

`[Sprint 1][Backend] Создать каркас Go backend service`

**Labels**

`sprint:1`, `role:backend`, `type:task`, `priority:p0`

**Depends on**

`S1-BE-02`, `S1-BE-04`

**Body**

```md
## Summary

Создать runnable skeleton Go backend service.

## Scope

- структура `cmd`, `internal`, `pkg`;
- конфиг и логирование;
- graceful shutdown;
- `health` и `readiness` endpoints;
- базовый router и middleware.

## Deliverables

- Go service skeleton;
- base config layer;
- middleware foundation.

## Acceptance Criteria

- [ ] сервис компилируется и стартует локально;
- [ ] есть health/readiness endpoints;
- [ ] есть базовая структура для API-реализации Sprint 2.
```

---

## 16. S1-BE-07

**Title**

`[Sprint 1][Backend] Спроектировать и завести service DB schema`

**Labels**

`sprint:1`, `role:backend`, `type:task`, `priority:p0`

**Depends on**

`S1-BE-03`, `S1-BE-06`

**Body**

```md
## Summary

Создать service DB schema и миграции для MVP сущностей.

## Scope

- таблицы `assets`, `watchlists`, `watchlist_items`;
- таблицы `notification_rules`, `signal_runs`, `signal_events`;
- таблицы `model_registry`, `job_runs`;
- индексы и constraints.

## Deliverables

- migration set v1;
- DB schema v1.

## Acceptance Criteria

- [ ] миграции применяются автоматически;
- [ ] схема покрывает Sprint 1 и Sprint 2 MVP-сущности;
- [ ] constraints и индексы заданы для ключевых сущностей.
```

---

## 17. S1-BE-08

**Title**

`[Sprint 1][Backend] Реализовать repository/service layer foundation`

**Labels**

`sprint:1`, `role:backend`, `type:task`, `priority:p0`

**Depends on**

`S1-BE-07`

**Body**

```md
## Summary

Подготовить repository/service foundation для API Sprint 2.

## Scope

- repository interfaces;
- базовые реализации для `assets`, `signals`, `watchlist`, `models`;
- service layer с валидацией;
- убрать SQL из будущих HTTP handlers.

## Deliverables

- repository layer;
- service layer foundation;
- базовые unit tests.

## Acceptance Criteria

- [ ] foundation покрывает основные доменные сущности;
- [ ] SQL не размазан по handlers;
- [ ] слой пригоден для unit testing и расширения во втором спринте.
```

---

## 18. S1-FE-01

**Title**

`[Sprint 1][Frontend] Подготовить screen map для MVP`

**Labels**

`sprint:1`, `role:frontend`, `type:task`, `priority:p0`

**Depends on**

`S1-BE-02`

**Body**

```md
## Summary

Подготовить screen map для MVP интерфейсов и связать его с API contract.

## Scope

- определить экраны `dashboard`, `asset details`, `watchlist`, `notification settings`;
- описать, какие данные требуются каждому экрану;
- собрать список UI-требований к API.

## Deliverables

- screen map;
- data requirements per screen.

## Acceptance Criteria

- [ ] список экранов зафиксирован;
- [ ] для каждого экрана понятен required data set;
- [ ] backend получил список обязательных UI-полей.
```

---

## 19. S1-FE-02

**Title**

`[Sprint 1][Frontend] Подготовить wireframes и UI states для MVP`

**Labels**

`sprint:1`, `role:frontend`, `type:task`, `priority:p0`

**Depends on**

`S1-FE-01`, `S1-BE-02`

**Body**

```md
## Summary

Подготовить wireframes и обязательные UI states для MVP интерфейсов.

## Scope

- wireframes для главного экрана и страницы инструмента;
- состояния `loading`, `empty`, `error`, `stale data`, `no signal`;
- проверить, что UI не требует неописанных backend fields.

## Deliverables

- MVP wireframes;
- UI states sheet.

## Acceptance Criteria

- [ ] wireframes согласованы;
- [ ] UI states описаны;
- [ ] Sprint 2 можно начинать без дополнительных UX-договорённостей.
```

---

## 20. S1-FE-03

**Title**

`[Sprint 1][Frontend] Подготовить frontend domain models и API mapping`

**Labels**

`sprint:1`, `role:frontend`, `type:task`, `priority:p0`

**Depends on**

`S1-BE-02`

**Body**

```md
## Summary

Подготовить mapping между frontend domain models и backend DTO.

## Scope

- описать frontend-side модели;
- сопоставить их с API DTO;
- выделить required/optional fields;
- собрать open questions к backend contract.

## Deliverables

- DTO mapping table;
- open questions list.

## Acceptance Criteria

- [ ] frontend готов к typed API client во втором спринте;
- [ ] все спорные поля подняты до реализации UI;
- [ ] mapping покрывает основные MVP endpoints.
```

---

## 21. S1-FE-04

**Title**

`[Sprint 1][Frontend] Подготовить app shell и техническую основу фронтенда`

**Labels**

`sprint:1`, `role:frontend`, `type:task`, `priority:p1`

**Depends on**

`S1-FE-03`, `S1-BE-05`

**Body**

```md
## Summary

Подготовить базовый frontend app shell, не блокируя критический путь Sprint 1.

## Scope

- выбрать стек frontend;
- создать пустой каркас приложения;
- настроить env handling;
- настроить routing skeleton и структуру каталогов.

## Deliverables

- frontend app shell;
- project structure;
- local start instructions.

## Acceptance Criteria

- [ ] frontend project стартует локально;
- [ ] env handling работает;
- [ ] структура готова к реализации dashboard в Sprint 2.
```

