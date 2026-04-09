# Research Protocol v1

## Цель

Определить единый протокол сравнения моделей, чтобы:

- результаты разных запусков были сопоставимы;
- выбор production candidate был воспроизводимым;
- исследование было пригодно для ВКР.

## Базовый режим оценки

Для MVP принимается `time-based walk-forward evaluation`.

Текущее runtime-покрытие:

- frozen `train/val/test` report для финального holdout сравнения;
- отдельный walk-forward report по `train+val` истории для устойчивости модели по rolling folds.

Обязательные правила:

- train/validation/test разбиваются только по времени;
- между сплитами добавляется purge gap не меньше `horizon_bars`;
- threshold tuning и calibration выполняются только на validation;
- final test остаётся замороженным для сравнения финальных кандидатов.

## Уровни оценки

### Global pooled

Модель оценивается на объединённом наборе инструментов.

### Per-ticker

Модель оценивается по каждому инструменту отдельно.

### Per-regime

Модель оценивается по рыночным режимам:

- low vol / high vol
- trend / mean-reversion-like regime
- opening / regular / evening session

## Основные метрики

### Classification quality

- `macro_f1`
- `precision_macro`
- `recall_macro`
- `balanced_accuracy`
- `roc_auc_ovr`, если применимо

### Signal usefulness

- `precision@actionable_signal`
- `recall@actionable_signal`
- `signal_coverage`
- `directional_hit_rate`

### Probability quality

- `brier_score`
- calibration curve / reliability diagnostics

## Правила выбора победившей модели

Побеждает не просто модель с лучшим одним числом, а модель, которая:

- имеет лучший или сопоставимый `macro_f1`;
- не деградирует критично по `precision@actionable_signal`;
- стабильно работает по разным тикерам и режимам;
- пригодна к production export и online inference.

## Обязательные артефакты каждого запуска

- training config
- dataset version
- feature schema version
- random seed
- validation metrics
- test metrics
- predictions dump
- calibration artifact, если применяется

## Ablation policy

После получения стабильного baseline и advanced candidate проводится минимум одна ablation series:

- без cross-asset features
- без regime features
- только core price/volume features

Это нужно, чтобы понять вклад продвинутых признаков в итоговый signal quality.
