from __future__ import annotations

from pathlib import Path
import json
from typing import Any


def generate_coverage_report(data_root: Path) -> dict[str, Any]:
    qa_root = data_root / "_meta" / "qa" / "raw"
    if not qa_root.exists():
        return {"items": [], "total_reports": 0}

    items = []
    for report_path in qa_root.rglob("report.json"):
        try:
            report = json.loads(report_path.read_text(encoding="utf-8"))
            items.append(report)
        except Exception:
            pass

    items.sort(key=lambda x: (x.get("source", ""), x.get("timeframe", ""), x.get("dataset_name", "")))

    return {
        "items": items,
        "total_reports": len(items),
    }


def write_coverage_report(report: dict[str, Any], output_path: Path) -> None:
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(report, indent=2, ensure_ascii=False, default=str), encoding="utf-8")
