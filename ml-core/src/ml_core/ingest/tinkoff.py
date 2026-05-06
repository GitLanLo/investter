from __future__ import annotations

from dataclasses import dataclass, field
import json
import os
from pathlib import Path
import ssl
import time
from typing import Any, Mapping
from urllib import error, request

import pandas as pd

from ml_core.ingest.providers import MarketDataProvider
from ml_core.ingest.universe import load_universe_config
from ml_core.qa.raw import build_raw_qa_report, write_raw_qa_report
from ml_core.storage.layouts import ingest_asset_frame, ingest_factor_frame


PROD_REST_URL = "https://invest-public-api.tinkoff.ru/rest/"
SANDBOX_REST_URL = "https://sandbox-invest-public-api.tinkoff.ru/rest/"

TIMEFRAME_TO_INTERVAL = {
    "1m": "CANDLE_INTERVAL_1_MIN",
    "2m": "CANDLE_INTERVAL_2_MIN",
    "3m": "CANDLE_INTERVAL_3_MIN",
    "5m": "CANDLE_INTERVAL_5_MIN",
    "10m": "CANDLE_INTERVAL_10_MIN",
    "15m": "CANDLE_INTERVAL_15_MIN",
    "30m": "CANDLE_INTERVAL_30_MIN",
    "1h": "CANDLE_INTERVAL_HOUR",
    "2h": "CANDLE_INTERVAL_2_HOUR",
    "4h": "CANDLE_INTERVAL_4_HOUR",
    "1d": "CANDLE_INTERVAL_DAY",
    "1w": "CANDLE_INTERVAL_WEEK",
    "1mo": "CANDLE_INTERVAL_MONTH",
}

TIMEFRAME_MAX_SPAN = {
    "1m": pd.Timedelta(days=1),
    "2m": pd.Timedelta(days=1),
    "3m": pd.Timedelta(days=1),
    "5m": pd.Timedelta(days=1),
    "10m": pd.Timedelta(days=1),
    "15m": pd.Timedelta(days=1),
    "30m": pd.Timedelta(days=2),
    "1h": pd.Timedelta(weeks=1),
    "2h": pd.Timedelta(days=31),
    "4h": pd.Timedelta(days=31),
    "1d": pd.Timedelta(days=365),
    "1w": pd.Timedelta(days=365 * 2),
    "1mo": pd.Timedelta(days=365 * 10),
}

TIMEFRAME_BAR_DELTA = {
    "1m": pd.Timedelta(minutes=1),
    "2m": pd.Timedelta(minutes=2),
    "3m": pd.Timedelta(minutes=3),
    "5m": pd.Timedelta(minutes=5),
    "10m": pd.Timedelta(minutes=10),
    "15m": pd.Timedelta(minutes=15),
    "30m": pd.Timedelta(minutes=30),
    "1h": pd.Timedelta(hours=1),
    "2h": pd.Timedelta(hours=2),
    "4h": pd.Timedelta(hours=4),
    "1d": pd.Timedelta(days=1),
    "1w": pd.Timedelta(days=7),
    "1mo": pd.Timedelta(days=30),
}

INSTRADAY_TIMEFRAMES = {"1m", "2m", "3m", "5m", "10m", "15m", "30m", "1h", "2h", "4h"}
INSTRUMENT_KIND_MAP = {
    "bond": "INSTRUMENT_TYPE_BOND",
    "share": "INSTRUMENT_TYPE_SHARE",
    "currency": "INSTRUMENT_TYPE_CURRENCY",
    "etf": "INSTRUMENT_TYPE_ETF",
    "futures": "INSTRUMENT_TYPE_FUTURES",
    "future": "INSTRUMENT_TYPE_FUTURES",
    "option": "INSTRUMENT_TYPE_OPTION",
    "sp": "INSTRUMENT_TYPE_SP",
    "clearing_certificate": "INSTRUMENT_TYPE_CLEARING_CERTIFICATE",
}

class TinkoffAPIError(RuntimeError):
    """Raised when Tinkoff REST API returns an error."""


class TinkoffConfigError(RuntimeError):
    """Raised when Tinkoff integration is misconfigured."""


class TinkoffInstrumentNotFoundError(TinkoffAPIError):
    """Raised when instrument lookup returns no exact match."""


class TinkoffAmbiguousInstrumentError(TinkoffAPIError):
    """Raised when instrument lookup returns several exact matches."""


@dataclass(slots=True)
class TinkoffConfig:
    token: str
    base_url: str = PROD_REST_URL
    timeout_seconds: float = 30.0
    user_agent: str = "invest-ml/0.1"
    ca_cert_file: str | None = None


@dataclass(slots=True)
class TinkoffInstrumentSpec:
    ticker: str
    instrument_kind: str | None = "INSTRUMENT_TYPE_SHARE"
    class_code: str | None = None
    api_trade_available_flag: bool = True


@dataclass(slots=True)
class TinkoffInstrument:
    ticker: str
    name: str
    uid: str
    figi: str
    instrument_kind: str
    class_code: str | None
    api_trade_available_flag: bool
    first_1min_candle_date: pd.Timestamp | None
    first_1day_candle_date: pd.Timestamp | None


@dataclass(slots=True)
class TinkoffRESTClient:
    config: TinkoffConfig

    def find_instruments(self, spec: TinkoffInstrumentSpec) -> list[TinkoffInstrument]:
        payload: dict[str, Any] = {
            "query": spec.ticker,
            "apiTradeAvailableFlag": spec.api_trade_available_flag,
        }
        if spec.instrument_kind:
            payload["instrumentKind"] = spec.instrument_kind

        response = self._post_json(
            "/tinkoff.public.invest.api.contract.v1.InstrumentsService/FindInstrument",
            payload,
        )
        instruments = response.get("instruments", [])
        return [self._parse_instrument(item) for item in instruments]

    def resolve_instrument(self, spec: TinkoffInstrumentSpec) -> TinkoffInstrument:
        items = self.find_instruments(spec)
        exact = [item for item in items if item.ticker.upper() == spec.ticker.upper()]
        if spec.class_code:
            exact = [item for item in exact if item.class_code == spec.class_code]

        if not exact:
            raise TinkoffInstrumentNotFoundError(
                f"instrument not found for ticker={spec.ticker} class_code={spec.class_code or '-'}"
            )
        if len(exact) > 1:
            variants = ", ".join(
                f"{item.ticker}:{item.class_code or '-'}:{item.instrument_kind}"
                for item in exact
            )
            raise TinkoffAmbiguousInstrumentError(
                f"ambiguous instrument for ticker={spec.ticker}; variants={variants}"
            )
        return exact[0]

    def get_historical_candles(
        self,
        *,
        instrument_id: str,
        timeframe: str,
        start: pd.Timestamp,
        end: pd.Timestamp,
        include_incomplete: bool = False,
    ) -> pd.DataFrame:
        normalized_timeframe = _normalize_timeframe(timeframe)
        interval = TIMEFRAME_TO_INTERVAL[normalized_timeframe]
        max_span = TIMEFRAME_MAX_SPAN[normalized_timeframe]

        start_ts = _to_utc_timestamp(start)
        end_ts = _to_utc_timestamp(end)
        if start_ts >= end_ts:
            raise ValueError("start must be earlier than end")

        frames: list[pd.DataFrame] = []
        cursor = start_ts
        while cursor < end_ts:
            chunk_end = min(cursor + max_span, end_ts)
            response = self._post_json(
                "/tinkoff.public.invest.api.contract.v1.MarketDataService/GetCandles",
                {
                    "instrumentId": instrument_id,
                    "from": cursor.isoformat(),
                    "to": chunk_end.isoformat(),
                    "interval": interval,
                },
            )
            chunk = self._parse_candles_response(response, include_incomplete=include_incomplete)
            if not chunk.empty:
                frames.append(chunk)
            cursor = chunk_end

        if not frames:
            return pd.DataFrame(columns=["timestamp", "open", "high", "low", "close", "volume"])

        out = pd.concat(frames, ignore_index=True)
        out["timestamp"] = pd.to_datetime(out["timestamp"], utc=True)
        out = out.sort_values("timestamp").drop_duplicates(subset=["timestamp"]).reset_index(drop=True)
        return out

    def _parse_instrument(self, payload: Mapping[str, Any]) -> TinkoffInstrument:
        return TinkoffInstrument(
            ticker=str(payload.get("ticker", "")),
            name=str(payload.get("name", "")),
            uid=str(payload.get("uid", "")),
            figi=str(payload.get("figi", "")),
            instrument_kind=str(payload.get("instrumentKind", "")),
            class_code=_optional_str(payload.get("classCode")),
            api_trade_available_flag=bool(payload.get("apiTradeAvailableFlag", False)),
            first_1min_candle_date=_optional_timestamp(payload.get("first1minCandleDate")),
            first_1day_candle_date=_optional_timestamp(payload.get("first1dayCandleDate")),
        )

    def _parse_candles_response(
        self,
        payload: Mapping[str, Any],
        *,
        include_incomplete: bool,
    ) -> pd.DataFrame:
        candles = payload.get("candles", [])
        rows = []
        for candle in candles:
            if not include_incomplete and not candle.get("isComplete", False):
                continue
            rows.append(
                {
                    "timestamp": candle["time"],
                    "open": quotation_to_float(candle.get("open")),
                    "high": quotation_to_float(candle.get("high")),
                    "low": quotation_to_float(candle.get("low")),
                    "close": quotation_to_float(candle.get("close")),
                    "volume": int(candle.get("volume", 0)),
                }
            )
        return pd.DataFrame(rows)

    def _post_json(self, path: str, payload: Mapping[str, Any]) -> dict[str, Any]:
        body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        req = request.Request(
            url=self.config.base_url.rstrip("/") + path,
            data=body,
            method="POST",
            headers={
                "Authorization": f"Bearer {self.config.token}",
                "Content-Type": "application/json",
                "Accept": "application/json",
                "User-Agent": self.config.user_agent,
            },
        )

        ctx = None
        if self.config.ca_cert_file:
            ctx = ssl.create_default_context()
            ctx.load_verify_locations(cafile=self.config.ca_cert_file)

        retries = 3
        backoff = 2.0
        last_exc: TinkoffAPIError | None = None

        for attempt in range(retries):
            try:
                with request.urlopen(req, timeout=self.config.timeout_seconds, context=ctx) as response:
                    return json.loads(response.read().decode("utf-8"))
            except error.HTTPError as exc:
                detail = exc.read().decode("utf-8", errors="replace")
                last_exc = TinkoffAPIError(f"Tinkoff API HTTP {exc.code}: {detail}")
                if exc.code not in (429, 500, 502, 503, 504):
                    raise last_exc from exc
            except error.URLError as exc:
                last_exc = TinkoffAPIError(f"Tinkoff API request failed: {exc.reason}")

            if attempt < retries - 1:
                time.sleep(backoff * (2 ** attempt))

        raise last_exc or TinkoffAPIError("Tinkoff API request failed")


@dataclass(slots=True)
class TinkoffRESTProvider(MarketDataProvider):
    client: TinkoffRESTClient
    asset_specs: Mapping[str, TinkoffInstrumentSpec] = field(default_factory=dict)
    factor_specs: Mapping[str, TinkoffInstrumentSpec] = field(default_factory=dict)

    def fetch_asset_history(
        self,
        ticker: str,
        timeframe: str,
        *,
        start: pd.Timestamp | None = None,
        end: pd.Timestamp | None = None,
    ) -> pd.DataFrame:
        spec = self.asset_specs.get(ticker) or TinkoffInstrumentSpec(
            ticker=ticker,
            instrument_kind="INSTRUMENT_TYPE_SHARE",
        )
        instrument = self.client.resolve_instrument(spec)
        return self.client.get_historical_candles(
            instrument_id=instrument.uid or instrument.figi,
            timeframe=timeframe,
            start=_coalesce_time(start, instrument, timeframe, "start"),
            end=_coalesce_time(end, instrument, timeframe, "end"),
        )

    def fetch_factor_history(
        self,
        alias: str,
        timeframe: str,
        *,
        start: pd.Timestamp | None = None,
        end: pd.Timestamp | None = None,
    ) -> pd.DataFrame:
        spec = self.factor_specs.get(alias)
        if spec is None:
            raise TinkoffConfigError(
                f"factor spec for alias={alias} is not configured; pass explicit mapping"
            )
        instrument = self.client.resolve_instrument(spec)
        return self.client.get_historical_candles(
            instrument_id=instrument.uid or instrument.figi,
            timeframe=timeframe,
            start=_coalesce_time(start, instrument, timeframe, "start"),
            end=_coalesce_time(end, instrument, timeframe, "end"),
        )


def load_tinkoff_config_from_env() -> TinkoffConfig:
    token = os.getenv("TINKOFF_INVEST_TOKEN", "").strip()
    if not token:
        raise TinkoffConfigError("TINKOFF_INVEST_TOKEN is required")

    target = os.getenv("TINKOFF_INVEST_TARGET", "prod").strip().lower()
    if target in {"prod", "production"}:
        base_url = PROD_REST_URL
    elif target == "sandbox":
        base_url = SANDBOX_REST_URL
    else:
        raise TinkoffConfigError("TINKOFF_INVEST_TARGET must be one of: prod, production, sandbox")

    timeout_raw = os.getenv("TINKOFF_INVEST_TIMEOUT_SECONDS", "30").strip()
    try:
        timeout_seconds = float(timeout_raw)
    except ValueError as exc:
        raise TinkoffConfigError("TINKOFF_INVEST_TIMEOUT_SECONDS must be numeric") from exc

    user_agent = os.getenv("TINKOFF_INVEST_USER_AGENT", "invest-ml/0.1").strip() or "invest-ml/0.1"
    ca_cert_file = os.getenv("TINKOFF_CA_CERT_FILE", "").strip() or None

    return TinkoffConfig(
        token=token,
        base_url=base_url,
        timeout_seconds=timeout_seconds,
        user_agent=user_agent,
        ca_cert_file=ca_cert_file,
    )


def sync_tinkoff_asset_history(
    *,
    data_root: Path,
    ticker: str,
    timeframe: str,
    start: pd.Timestamp,
    end: pd.Timestamp,
    instrument_kind: str | None = "share",
    class_code: str | None = None,
    incremental: bool = True,
    overlap_bars: int = 3,
    client: TinkoffRESTClient | None = None,
) -> dict[str, Any]:
    rest_client = client or TinkoffRESTClient(load_tinkoff_config_from_env())
    spec = TinkoffInstrumentSpec(
        ticker=ticker,
        instrument_kind=normalize_instrument_kind(instrument_kind),
        class_code=class_code,
    )
    instrument = rest_client.resolve_instrument(spec)
    normalized_timeframe = _normalize_timeframe(timeframe)
    state_path = _sync_state_path(data_root, dataset_kind="asset", dataset_name=ticker, timeframe=normalized_timeframe)
    state = _load_sync_state(state_path)
    _validate_sync_state_compatibility(
        state,
        state_path=state_path,
        dataset_kind="asset",
        dataset_name=ticker,
        timeframe=normalized_timeframe,
        instrument=instrument,
    )
    effective_start = _resolve_effective_start(start, instrument, timeframe)
    if incremental:
        effective_start = _apply_incremental_start(
            effective_start,
            state=state,
            timeframe=normalized_timeframe,
            overlap_bars=overlap_bars,
        )
    candles = rest_client.get_historical_candles(
        instrument_id=instrument.uid or instrument.figi,
        timeframe=timeframe,
        start=effective_start,
        end=_to_utc_timestamp(end),
    )
    written = ingest_asset_frame(
        candles,
        data_root=data_root,
        ticker=ticker,
        timeframe=normalized_timeframe,
        source="tinkoff_invest_api_rest",
    )
    qa_report = build_raw_qa_report(
        candles,
        timeframe=normalized_timeframe,
        dataset_name=ticker,
        source="tinkoff_invest_api_rest",
    )
    qa_report_path = write_raw_qa_report(
        qa_report,
        data_root=data_root,
        dataset_kind="asset",
        dataset_name=ticker,
        timeframe=normalized_timeframe,
        source="tinkoff_invest_api_rest",
    )
    persisted_max_timestamp = _persist_sync_state(
        state_path,
        existing_state=state,
        dataset_name=ticker,
        dataset_kind="asset",
        timeframe=normalized_timeframe,
        start=effective_start,
        end=_to_utc_timestamp(end),
        rows=len(candles),
        instrument=instrument,
        overlap_bars=overlap_bars,
        candles=candles,
    )
    return {
        "ticker": ticker,
        "rows": int(len(candles)),
        "timeframe": normalized_timeframe,
        "instrument_uid": instrument.uid,
        "figi": instrument.figi,
        "effective_start": effective_start.isoformat(),
        "effective_end": _to_utc_timestamp(end).isoformat(),
        "incremental": incremental,
        "state_path": str(state_path),
        "qa_report_path": str(qa_report_path),
        "persisted_max_timestamp": persisted_max_timestamp,
        "written_files": [str(path) for path in written],
        "range": _frame_range(candles),
        "qa": qa_report,
    }


def sync_tinkoff_factor_history(
    *,
    data_root: Path,
    alias: str,
    ticker: str,
    timeframe: str,
    start: pd.Timestamp,
    end: pd.Timestamp,
    instrument_kind: str | None,
    class_code: str | None = None,
    incremental: bool = True,
    overlap_bars: int = 3,
    client: TinkoffRESTClient | None = None,
) -> dict[str, Any]:
    rest_client = client or TinkoffRESTClient(load_tinkoff_config_from_env())
    spec = TinkoffInstrumentSpec(
        ticker=ticker,
        instrument_kind=normalize_instrument_kind(instrument_kind),
        class_code=class_code,
    )
    instrument = rest_client.resolve_instrument(spec)
    normalized_timeframe = _normalize_timeframe(timeframe)
    state_path = _sync_state_path(data_root, dataset_kind="factor", dataset_name=alias, timeframe=normalized_timeframe)
    state = _load_sync_state(state_path)
    _validate_sync_state_compatibility(
        state,
        state_path=state_path,
        dataset_kind="factor",
        dataset_name=alias,
        timeframe=normalized_timeframe,
        instrument=instrument,
    )
    effective_start = _resolve_effective_start(start, instrument, timeframe)
    if incremental:
        effective_start = _apply_incremental_start(
            effective_start,
            state=state,
            timeframe=normalized_timeframe,
            overlap_bars=overlap_bars,
        )
    candles = rest_client.get_historical_candles(
        instrument_id=instrument.uid or instrument.figi,
        timeframe=timeframe,
        start=effective_start,
        end=_to_utc_timestamp(end),
    )
    written = ingest_factor_frame(
        candles,
        data_root=data_root,
        alias=alias,
        timeframe=normalized_timeframe,
        source="tinkoff_invest_api_rest",
    )
    qa_report = build_raw_qa_report(
        candles,
        timeframe=normalized_timeframe,
        dataset_name=alias,
        source="tinkoff_invest_api_rest",
    )
    qa_report_path = write_raw_qa_report(
        qa_report,
        data_root=data_root,
        dataset_kind="factor",
        dataset_name=alias,
        timeframe=normalized_timeframe,
        source="tinkoff_invest_api_rest",
    )
    persisted_max_timestamp = _persist_sync_state(
        state_path,
        existing_state=state,
        dataset_name=alias,
        dataset_kind="factor",
        timeframe=normalized_timeframe,
        start=effective_start,
        end=_to_utc_timestamp(end),
        rows=len(candles),
        instrument=instrument,
        overlap_bars=overlap_bars,
        candles=candles,
    )
    return {
        "alias": alias,
        "ticker": ticker,
        "rows": int(len(candles)),
        "timeframe": normalized_timeframe,
        "instrument_uid": instrument.uid,
        "figi": instrument.figi,
        "effective_start": effective_start.isoformat(),
        "effective_end": _to_utc_timestamp(end).isoformat(),
        "incremental": incremental,
        "state_path": str(state_path),
        "qa_report_path": str(qa_report_path),
        "persisted_max_timestamp": persisted_max_timestamp,
        "written_files": [str(path) for path in written],
        "range": _frame_range(candles),
        "qa": qa_report,
    }


def sync_tinkoff_universe(
    *,
    config_path: Path,
    data_root: Path,
    start: pd.Timestamp,
    end: pd.Timestamp,
    incremental: bool = True,
    overlap_bars: int = 3,
    client: TinkoffRESTClient | None = None,
    timeframe_override: str | None = None,
) -> dict[str, Any]:
    config = load_universe_config(config_path)
    rest_client = client or TinkoffRESTClient(load_tinkoff_config_from_env())

    assets: list[dict[str, Any]] = []
    for item in config.assets:
        if not item.enabled:
            continue
        try:
            res = sync_tinkoff_asset_history(
                data_root=data_root,
                ticker=item.ticker,
                timeframe=timeframe_override or item.timeframe,
                start=start,
                end=end,
                instrument_kind=item.instrument_kind,
                class_code=item.class_code,
                incremental=incremental,
                overlap_bars=overlap_bars,
                client=rest_client,
            )
            assets.append(res)
        except Exception as exc:
            assets.append({
                "ticker": item.ticker,
                "timeframe": timeframe_override or item.timeframe,
                "error": str(exc),
                "rows": 0,
            })

    factors: list[dict[str, Any]] = []
    for item in config.factors:
        if not item.enabled:
            continue
        try:
            res = sync_tinkoff_factor_history(
                data_root=data_root,
                alias=item.alias,
                ticker=item.ticker,
                timeframe=timeframe_override or item.timeframe,
                start=start,
                end=end,
                instrument_kind=item.instrument_kind,
                class_code=item.class_code,
                incremental=incremental,
                overlap_bars=overlap_bars,
                client=rest_client,
            )
            factors.append(res)
        except Exception as exc:
            factors.append({
                "alias": item.alias,
                "ticker": item.ticker,
                "timeframe": timeframe_override or item.timeframe,
                "error": str(exc),
                "rows": 0,
            })

    summary = {
        "universe_name": config.name,
        "config_path": str(config_path),
        "start": _to_utc_timestamp(start).isoformat(),
        "end": _to_utc_timestamp(end).isoformat(),
        "incremental": incremental,
        "overlap_bars": overlap_bars,
        "assets": assets,
        "factors": factors,
        "totals": {
            "asset_rows": int(sum(item["rows"] for item in assets)),
            "factor_rows": int(sum(item["rows"] for item in factors)),
            "asset_count": len(assets),
            "factor_count": len(factors),
        },
    }
    summary_path = _universe_summary_path(data_root, config.name)
    summary_path.parent.mkdir(parents=True, exist_ok=True)
    summary_path.write_text(json.dumps(summary, indent=2, ensure_ascii=False, default=str), encoding="utf-8")
    summary["summary_path"] = str(summary_path)
    return summary


def quotation_to_float(value: Mapping[str, Any] | None) -> float:
    if not value:
        return 0.0
    units = int(value.get("units", 0) or 0)
    nano = int(value.get("nano", 0) or 0)
    return units + nano / 1_000_000_000


def normalize_instrument_kind(value: str | None) -> str | None:
    if value is None:
        return None
    raw = value.strip()
    if not raw:
        return None
    if raw.startswith("INSTRUMENT_TYPE_"):
        return raw
    mapped = INSTRUMENT_KIND_MAP.get(raw.lower())
    if mapped:
        return mapped
    raise TinkoffConfigError(f"unsupported instrument kind: {value}")


def _coalesce_time(
    value: pd.Timestamp | None,
    instrument: TinkoffInstrument,
    timeframe: str,
    kind: str,
) -> pd.Timestamp:
    if kind == "end":
        if value is None:
            return pd.Timestamp.now(tz="UTC")
        return _to_utc_timestamp(value)
    return _resolve_effective_start(value, instrument, timeframe)


def _resolve_effective_start(
    start: pd.Timestamp | None,
    instrument: TinkoffInstrument,
    timeframe: str,
) -> pd.Timestamp:
    requested = _to_utc_timestamp(start) if start is not None else None
    if _normalize_timeframe(timeframe) in INSTRADAY_TIMEFRAMES:
        earliest = instrument.first_1min_candle_date
    else:
        earliest = instrument.first_1day_candle_date

    if earliest is None and requested is None:
        raise TinkoffConfigError("start is required when instrument does not expose first candle date")
    if earliest is None:
        return requested  # type: ignore[return-value]
    if requested is None:
        return earliest
    return max(requested, earliest)


def _frame_range(df: pd.DataFrame) -> dict[str, Any]:
    if df.empty:
        return {"min": None, "max": None, "rows": 0}
    return {
        "min": pd.to_datetime(df["timestamp"], utc=True).min().isoformat(),
        "max": pd.to_datetime(df["timestamp"], utc=True).max().isoformat(),
        "rows": int(len(df)),
    }


def _normalize_timeframe(value: str) -> str:
    normalized = value.strip().lower()
    if normalized not in TIMEFRAME_TO_INTERVAL:
        raise TinkoffConfigError(f"unsupported timeframe: {value}")
    return normalized


def _optional_timestamp(value: Any) -> pd.Timestamp | None:
    if value in (None, ""):
        return None
    return _to_utc_timestamp(pd.Timestamp(value))


def _optional_str(value: Any) -> str | None:
    if value in (None, ""):
        return None
    return str(value)


def _apply_incremental_start(
    requested_start: pd.Timestamp,
    *,
    state: Mapping[str, Any] | None,
    timeframe: str,
    overlap_bars: int,
) -> pd.Timestamp:
    if not state or not state.get("last_written_max_timestamp"):
        return requested_start

    watermark = _to_utc_timestamp(pd.Timestamp(state["last_written_max_timestamp"]))
    overlap = TIMEFRAME_BAR_DELTA[timeframe] * overlap_bars
    return max(requested_start, watermark - overlap)


def _sync_state_path(
    data_root: Path,
    *,
    dataset_kind: str,
    dataset_name: str,
    timeframe: str,
) -> Path:
    return (
        data_root
        / "_meta"
        / "ingest_state"
        / "source=tinkoff_invest_api_rest"
        / f"kind={dataset_kind}"
        / f"name={dataset_name}"
        / f"timeframe={timeframe}.json"
    )


def _load_sync_state(path: Path) -> dict[str, Any] | None:
    if not path.exists():
        return None
    return json.loads(path.read_text(encoding="utf-8"))


def _validate_sync_state_compatibility(
    state: Mapping[str, Any] | None,
    *,
    state_path: Path,
    dataset_kind: str,
    dataset_name: str,
    timeframe: str,
    instrument: TinkoffInstrument,
) -> None:
    if not state:
        return

    mismatches: list[str] = []
    for field_name, current_value in (
        ("instrument_uid", instrument.uid),
        ("figi", instrument.figi),
        ("instrument_kind", instrument.instrument_kind),
        ("class_code", instrument.class_code),
    ):
        persisted_value = state.get(field_name)
        if persisted_value in (None, "") or current_value in (None, ""):
            continue
        if str(persisted_value) != str(current_value):
            mismatches.append(f"{field_name}: state={persisted_value} current={current_value}")

    if mismatches:
        details = "; ".join(mismatches)
        raise TinkoffConfigError(
            "existing ingest state belongs to another Tinkoff instrument for "
            f"{dataset_kind}={dataset_name} timeframe={timeframe}: {details}. "
            f"Reset {state_path} and the corresponding raw partition if remapping is intentional."
        )


def _resolve_persisted_watermark(
    existing_state: Mapping[str, Any] | None,
    candles: pd.DataFrame,
) -> str | None:
    candidates: list[pd.Timestamp] = []

    if existing_state and existing_state.get("last_written_max_timestamp"):
        candidates.append(_to_utc_timestamp(pd.Timestamp(existing_state["last_written_max_timestamp"])))
    if not candles.empty:
        candidates.append(pd.to_datetime(candles["timestamp"], utc=True).max())

    if not candidates:
        return None
    return max(candidates).isoformat()


def _persist_sync_state(
    path: Path,
    *,
    existing_state: Mapping[str, Any] | None,
    dataset_name: str,
    dataset_kind: str,
    timeframe: str,
    start: pd.Timestamp,
    end: pd.Timestamp,
    rows: int,
    instrument: TinkoffInstrument,
    overlap_bars: int,
    candles: pd.DataFrame,
) -> str | None:
    path.parent.mkdir(parents=True, exist_ok=True)
    last_written_max_timestamp = _resolve_persisted_watermark(existing_state, candles)

    payload = {
        "dataset_kind": dataset_kind,
        "dataset_name": dataset_name,
        "timeframe": timeframe,
        "source": "tinkoff_invest_api_rest",
        "instrument_uid": instrument.uid,
        "figi": instrument.figi,
        "instrument_kind": instrument.instrument_kind,
        "class_code": instrument.class_code,
        "requested_start": _to_utc_timestamp(start).isoformat(),
        "requested_end": _to_utc_timestamp(end).isoformat(),
        "last_written_max_timestamp": last_written_max_timestamp,
        "rows_fetched_last_run": int(rows),
        "overlap_bars": overlap_bars,
        "updated_at": pd.Timestamp.now(tz="UTC").isoformat(),
    }
    path.write_text(json.dumps(payload, indent=2, ensure_ascii=False), encoding="utf-8")
    return last_written_max_timestamp


def _universe_summary_path(data_root: Path, universe_name: str) -> Path:
    timestamp = pd.Timestamp.now(tz="UTC").strftime("%Y%m%dT%H%M%SZ")
    return data_root / "_meta" / "ingest_runs" / "tinkoff" / universe_name / f"{timestamp}.json"


def _to_utc_timestamp(value: pd.Timestamp) -> pd.Timestamp:
    ts = pd.Timestamp(value)
    if ts.tzinfo is None:
        return ts.tz_localize("UTC")
    return ts.tz_convert("UTC")
