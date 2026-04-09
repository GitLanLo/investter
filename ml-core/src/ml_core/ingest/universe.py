from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
import json


@dataclass(slots=True)
class UniverseAssetSpec:
    ticker: str
    timeframe: str = "5m"
    instrument_kind: str | None = "share"
    class_code: str | None = None
    enabled: bool = True


@dataclass(slots=True)
class UniverseFactorSpec:
    alias: str
    ticker: str
    timeframe: str = "5m"
    instrument_kind: str | None = None
    class_code: str | None = None
    enabled: bool = True


@dataclass(slots=True)
class UniverseConfig:
    name: str
    assets: list[UniverseAssetSpec]
    factors: list[UniverseFactorSpec]


def load_universe_config(path: Path) -> UniverseConfig:
    payload = json.loads(path.read_text(encoding="utf-8"))
    return UniverseConfig(
        name=str(payload["name"]),
        assets=[
            UniverseAssetSpec(
                ticker=str(item["ticker"]),
                timeframe=str(item.get("timeframe", "5m")),
                instrument_kind=item.get("instrument_kind", "share"),
                class_code=item.get("class_code"),
                enabled=bool(item.get("enabled", True)),
            )
            for item in payload.get("assets", [])
        ],
        factors=[
            UniverseFactorSpec(
                alias=str(item["alias"]),
                ticker=str(item["ticker"]),
                timeframe=str(item.get("timeframe", "5m")),
                instrument_kind=item.get("instrument_kind"),
                class_code=item.get("class_code"),
                enabled=bool(item.get("enabled", True)),
            )
            for item in payload.get("factors", [])
        ],
    )
