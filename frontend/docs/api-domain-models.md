# Frontend Domain Models and API Mapping

## Asset list

| Backend DTO field | Frontend field | Notes |
| --- | --- | --- |
| `id` | `id` | canonical asset id |
| `ticker` | `ticker` | shown in compact cards |
| `name` | `name` | human-readable title |
| `exchange` | `venue` | renamed for UI semantics |
| `timeframe` | `timeframe` | directly rendered |
| `is_active` | `active` | boolean status badge |

## Signal card

| Backend DTO field | Frontend field | Notes |
| --- | --- | --- |
| `id` | `id` | list key |
| `asset_id` | `assetId` | joins with asset cards |
| `signal_direction` | `direction` | `up/down/flat` UI badge |
| `signal_state` | `state` | actionable/watch/no-trade semantics |
| `signal_probability` | `probability` | number in `[0,1]` |
| `threshold` | `threshold` | UI comparison |
| `as_of_time` | `asOfTime` | UTC ISO string |
| `timeframe` | `timeframe` | displayed in chip |
| `model_version` | `modelVersion` | inspection/debug panel |
| `class_probabilities` | `classProbabilities` | probability stack |

## Mapping decisions

- backend DTOs не тащатся в UI напрямую; сначала проходит тонкий mapping layer в `src/lib/api.ts`;
- UI-модели intentionally flatter, чем backend DTO, чтобы не смешивать API concerns и presentation concerns;
- mock data использует те же frontend domain models, поэтому shell одинаково работает с live API и fallback path.
