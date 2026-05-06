from __future__ import annotations

import json
from dataclasses import dataclass, field, asdict
from pathlib import Path
from typing import Any

@dataclass
class ModelManifest:
    model_version: str
    model_family: str
    task: str
    classes: list[str]
    input_timeframe: str
    prediction_horizon_bars: int
    feature_schema_version: str
    feature_order: list[str]
    export_format: str
    model_artifact_path: str
    artifact_sha256: str
    metrics: dict[str, float]
    threshold: float
    calibration: str
    created_at: str
    source_dataset_version: str

    # Neural / Sequence specific fields
    input_window_bars: int | None = None
    input_tensor_shape: list[int] | None = None
    normalization: str | None = None

    def to_json(self) -> str:
        return json.dumps(asdict(self), indent=2, ensure_ascii=False)

    @classmethod
    def from_json(cls, data: str) -> "ModelManifest":
        return cls(**json.loads(data))

    def save(self, path: Path) -> None:
        path.write_text(self.to_json(), encoding="utf-8")

    @classmethod
    def load(cls, path: Path) -> "ModelManifest":
        return cls.from_json(path.read_text(encoding="utf-8"))
