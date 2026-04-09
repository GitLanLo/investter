# MVP Wireframes v1

## Dashboard

```text
+--------------------------------------------------------------+
| Header: brand | nav | env badge                              |
+--------------------------------------------------------------+
| Hero: universe summary | backend/feed status | latest signal |
+--------------------------------------------------------------+
| Spotlight asset card    | Probability stack                  |
+--------------------------------------------------------------+
| Universe tiles / watchlist                                  |
+--------------------------------------------------------------+
| Latest signals rail                                          |
+--------------------------------------------------------------+
```

## Asset Details

```text
+--------------------------------------------------------------+
| Instrument header: ticker, name, timeframe, signal state     |
+--------------------------------------------------------------+
| Probability hero | model version | threshold                 |
+--------------------------------------------------------------+
| Signal timeline / recent runs                                |
+--------------------------------------------------------------+
| Feature/label metadata panel                                 |
+--------------------------------------------------------------+
```

## UI States

- `loading`: skeleton or muted placeholder cards;
- `api ready`: actual backend data;
- `api offline`: fallback mock cards with explicit source badge;
- `empty signals`: zero-state block with explanation and refresh CTA;
- `degraded`: backend доступен, но сигналов нет или часть DTO невалидна.
