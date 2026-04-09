from __future__ import annotations

from pathlib import Path
import json

import pandas as pd

from ml_core.ingest.providers import LocalParquetProvider
from ml_core.ingest.tinkoff import (
    PROD_REST_URL,
    TinkoffConfig,
    TinkoffConfigError,
    TinkoffInstrumentSpec,
    TinkoffRESTClient,
    normalize_instrument_kind,
    sync_tinkoff_asset_history,
    sync_tinkoff_universe,
)


class FakeTinkoffRESTClient(TinkoffRESTClient):
    def __init__(self, *, instrument: dict | None = None) -> None:
        super().__init__(TinkoffConfig(token="test", base_url=PROD_REST_URL))
        self.calls: list[tuple[str, dict]] = []
        self.instrument = instrument or {
            "ticker": "SBER",
            "name": "Sberbank",
            "uid": "uid-sber",
            "figi": "figi-sber",
            "instrumentKind": "INSTRUMENT_TYPE_SHARE",
            "classCode": "TQBR",
            "apiTradeAvailableFlag": True,
            "first1minCandleDate": "2026-04-01T07:00:00Z",
            "first1dayCandleDate": "2010-01-01T00:00:00Z",
        }

    def _post_json(self, path: str, payload: dict) -> dict:  # type: ignore[override]
        self.calls.append((path, payload))

        if path.endswith("/FindInstrument"):
            return {"instruments": [self.instrument]}

        if path.endswith("/GetCandles"):
            if payload["from"] == "2026-04-01T07:00:00+00:00":
                return {
                    "candles": [
                        {
                            "time": "2026-04-01T07:00:00Z",
                            "open": {"units": "300", "nano": 0},
                            "high": {"units": "301", "nano": 0},
                            "low": {"units": "299", "nano": 0},
                            "close": {"units": "300", "nano": 500000000},
                            "volume": "1000",
                            "isComplete": True,
                        },
                        {
                            "time": "2026-04-02T06:55:00Z",
                            "open": {"units": "301", "nano": 0},
                            "high": {"units": "302", "nano": 0},
                            "low": {"units": "300", "nano": 0},
                            "close": {"units": "301", "nano": 500000000},
                            "volume": "1100",
                            "isComplete": True,
                        },
                    ]
                }
            return {
                "candles": [
                    {
                        "time": "2026-04-02T06:55:00Z",
                        "open": {"units": "301", "nano": 0},
                        "high": {"units": "302", "nano": 0},
                        "low": {"units": "300", "nano": 0},
                        "close": {"units": "301", "nano": 500000000},
                        "volume": "1100",
                        "isComplete": True,
                    },
                    {
                        "time": "2026-04-02T07:00:00Z",
                        "open": {"units": "302", "nano": 0},
                        "high": {"units": "303", "nano": 0},
                        "low": {"units": "301", "nano": 0},
                        "close": {"units": "302", "nano": 500000000},
                        "volume": "1200",
                        "isComplete": True,
                    },
                ]
            }

        raise AssertionError(f"unexpected path {path}")


class SequencedCandleTinkoffRESTClient(FakeTinkoffRESTClient):
    def __init__(self, candle_batches: list[list[dict]], *, instrument: dict | None = None) -> None:
        super().__init__(instrument=instrument)
        self.candle_batches = candle_batches

    def _post_json(self, path: str, payload: dict) -> dict:  # type: ignore[override]
        self.calls.append((path, payload))

        if path.endswith("/FindInstrument"):
            return {"instruments": [self.instrument]}
        if path.endswith("/GetCandles"):
            if not self.candle_batches:
                raise AssertionError("unexpected GetCandles call")
            return {"candles": self.candle_batches.pop(0)}

        raise AssertionError(f"unexpected path {path}")


def test_tinkoff_sync_asset_history_writes_raw_layout_and_chunks_requests(tmp_path: Path) -> None:
    data_root = tmp_path / "data"
    client = FakeTinkoffRESTClient()

    summary = sync_tinkoff_asset_history(
        data_root=data_root,
        ticker="SBER",
        timeframe="5m",
        start=pd.Timestamp("2026-03-20T07:00:00Z"),
        end=pd.Timestamp("2026-04-03T07:00:00Z"),
        instrument_kind="share",
        class_code="TQBR",
        client=client,
    )

    assert summary["ticker"] == "SBER"
    assert summary["rows"] == 3
    assert summary["instrument_uid"] == "uid-sber"
    assert len([call for call in client.calls if call[0].endswith("/GetCandles")]) == 2
    assert Path(summary["state_path"]).exists()
    assert Path(summary["qa_report_path"]).exists()

    provider = LocalParquetProvider(root=data_root)
    loaded = provider.fetch_asset_history("SBER", "5m")
    assert len(loaded) == 3
    assert loaded["source"].eq("tinkoff_invest_api_rest").all()

    second_summary = sync_tinkoff_asset_history(
        data_root=data_root,
        ticker="SBER",
        timeframe="5m",
        start=pd.Timestamp("2026-03-20T07:00:00Z"),
        end=pd.Timestamp("2026-04-03T07:00:00Z"),
        instrument_kind="share",
        class_code="TQBR",
        overlap_bars=1,
        client=client,
    )
    loaded_after_second_sync = provider.fetch_asset_history("SBER", "5m")
    assert len(loaded_after_second_sync) == 3
    assert loaded_after_second_sync["timestamp"].is_unique
    state = json.loads(Path(second_summary["state_path"]).read_text(encoding="utf-8"))
    assert state["last_written_max_timestamp"] == "2026-04-02T07:00:00+00:00"
    assert state["instrument_kind"] == "INSTRUMENT_TYPE_SHARE"
    assert state["class_code"] == "TQBR"


def test_incremental_sync_preserves_watermark_when_next_run_returns_no_new_candles(tmp_path: Path) -> None:
    data_root = tmp_path / "data"
    client = SequencedCandleTinkoffRESTClient(
        candle_batches=[
            [
                {
                    "time": "2026-04-01T07:00:00Z",
                    "open": {"units": "300", "nano": 0},
                    "high": {"units": "301", "nano": 0},
                    "low": {"units": "299", "nano": 0},
                    "close": {"units": "300", "nano": 500000000},
                    "volume": "1000",
                    "isComplete": True,
                },
            ],
            [
                {
                    "time": "2026-04-02T07:00:00Z",
                    "open": {"units": "302", "nano": 0},
                    "high": {"units": "303", "nano": 0},
                    "low": {"units": "301", "nano": 0},
                    "close": {"units": "302", "nano": 500000000},
                    "volume": "1200",
                    "isComplete": True,
                },
            ],
            [],
        ]
    )

    first_summary = sync_tinkoff_asset_history(
        data_root=data_root,
        ticker="SBER",
        timeframe="5m",
        start=pd.Timestamp("2026-03-20T07:00:00Z"),
        end=pd.Timestamp("2026-04-02T07:05:00Z"),
        instrument_kind="share",
        class_code="TQBR",
        overlap_bars=0,
        client=client,
    )
    second_summary = sync_tinkoff_asset_history(
        data_root=data_root,
        ticker="SBER",
        timeframe="5m",
        start=pd.Timestamp("2026-03-20T07:00:00Z"),
        end=pd.Timestamp("2026-04-02T07:05:00Z"),
        instrument_kind="share",
        class_code="TQBR",
        overlap_bars=0,
        client=client,
    )

    first_state = json.loads(Path(first_summary["state_path"]).read_text(encoding="utf-8"))
    second_state = json.loads(Path(second_summary["state_path"]).read_text(encoding="utf-8"))
    assert first_state["last_written_max_timestamp"] == "2026-04-02T07:00:00+00:00"
    assert second_state["last_written_max_timestamp"] == "2026-04-02T07:00:00+00:00"


def test_tinkoff_sync_rejects_existing_state_for_other_instrument(tmp_path: Path) -> None:
    data_root = tmp_path / "data"
    first_client = FakeTinkoffRESTClient()
    sync_tinkoff_asset_history(
        data_root=data_root,
        ticker="SBER",
        timeframe="5m",
        start=pd.Timestamp("2026-03-20T07:00:00Z"),
        end=pd.Timestamp("2026-04-03T07:00:00Z"),
        instrument_kind="share",
        class_code="TQBR",
        client=first_client,
    )

    conflicting_client = FakeTinkoffRESTClient(
        instrument={
            "ticker": "SBER",
            "name": "Sberbank Alt",
            "uid": "uid-sber-alt",
            "figi": "figi-sber-alt",
            "instrumentKind": "INSTRUMENT_TYPE_SHARE",
            "classCode": "SPBX",
            "apiTradeAvailableFlag": True,
            "first1minCandleDate": "2026-04-01T07:00:00Z",
            "first1dayCandleDate": "2010-01-01T00:00:00Z",
        }
    )

    try:
        sync_tinkoff_asset_history(
            data_root=data_root,
            ticker="SBER",
            timeframe="5m",
            start=pd.Timestamp("2026-03-20T07:00:00Z"),
            end=pd.Timestamp("2026-04-03T07:00:00Z"),
            instrument_kind="share",
            class_code="SPBX",
            client=conflicting_client,
        )
    except TinkoffConfigError as exc:
        assert "belongs to another Tinkoff instrument" in str(exc)
    else:
        raise AssertionError("expected TinkoffConfigError")


def test_normalize_instrument_kind_supports_short_names() -> None:
    assert normalize_instrument_kind("share") == "INSTRUMENT_TYPE_SHARE"
    assert normalize_instrument_kind("currency") == "INSTRUMENT_TYPE_CURRENCY"
    assert normalize_instrument_kind("INSTRUMENT_TYPE_ETF") == "INSTRUMENT_TYPE_ETF"


def test_normalize_instrument_kind_rejects_unknown_values() -> None:
    try:
        normalize_instrument_kind("crypto")
    except TinkoffConfigError:
        pass
    else:
        raise AssertionError("expected TinkoffConfigError")


def test_fake_client_resolves_exact_instrument() -> None:
    client = FakeTinkoffRESTClient()
    instrument = client.resolve_instrument(TinkoffInstrumentSpec(ticker="SBER", class_code="TQBR"))
    assert instrument.uid == "uid-sber"
    assert instrument.class_code == "TQBR"


def test_tinkoff_sync_universe_reads_config_and_writes_summary(tmp_path: Path) -> None:
    data_root = tmp_path / "data"
    client = FakeTinkoffRESTClient()
    config_path = tmp_path / "universe.json"
    config_path.write_text(
        json.dumps(
            {
                "name": "test_universe",
                "assets": [
                    {
                        "ticker": "SBER",
                        "timeframe": "5m",
                        "instrument_kind": "share",
                        "class_code": "TQBR",
                    }
                ],
                "factors": [],
            }
        ),
        encoding="utf-8",
    )

    summary = sync_tinkoff_universe(
        config_path=config_path,
        data_root=data_root,
        start=pd.Timestamp("2026-03-20T07:00:00Z"),
        end=pd.Timestamp("2026-04-03T07:00:00Z"),
        client=client,
    )

    assert summary["universe_name"] == "test_universe"
    assert summary["totals"]["asset_count"] == 1
    assert summary["totals"]["asset_rows"] == 3
    assert Path(summary["summary_path"]).exists()
