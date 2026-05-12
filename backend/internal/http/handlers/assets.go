package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"invest/backend/internal/app"
	"invest/backend/internal/service"
)

type CandlesResponse struct {
	AssetID   string      `json:"asset_id"`
	Timeframe string      `json:"timeframe"`
	Items     []CandleDTO `json:"items"`
}

type FactorsResponse struct {
	AssetID string      `json:"asset_id"`
	Items   []FactorDTO `json:"items"`
}

type AssetSummaryResponse struct {
	Asset        AssetDTO   `json:"asset"`
	LastPrice    float64    `json:"last_price"`
	PriceChange  float64    `json:"price_change"`
	DataFresh    bool       `json:"data_fresh"`
	LastCandleAt string     `json:"last_candle_at,omitempty"`
	LastSignal   *SignalDTO `json:"last_signal,omitempty"`
}

type refreshAssetRequest struct {
	Timeframe string `json:"timeframe"`
}

func ListAssets(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := container.Services.Assets.List(r.Context())
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "assets_list_failed", err.Error(), nil)
			return
		}

		out := AssetsResponse{Items: make([]AssetDTO, 0, len(items))}
		for _, item := range items {
			out.Items = append(out.Items, ToAssetDTO(item))
		}
		WriteJSON(w, http.StatusOK, out)
	}
}

func GetAssetCandles(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		assetID := r.PathValue("id")
		if assetID == "" {
			WriteError(w, http.StatusBadRequest, "missing_asset_id", "asset id is required", nil)
			return
		}

		from := time.Time{}
		if rawFrom := r.URL.Query().Get("from"); rawFrom != "" {
			if t, err := time.Parse(time.RFC3339, rawFrom); err == nil {
				from = t
			}
		}
		to := time.Time{}
		if rawTo := r.URL.Query().Get("to"); rawTo != "" {
			if t, err := time.Parse(time.RFC3339, rawTo); err == nil {
				to = t
			}
		}
		limit := 1000
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			if l, err := strconv.Atoi(rawLimit); err == nil {
				limit = l
			}
		}

		asset, items, err := container.Services.MarketData.ListCandlesTimeframe(
			r.Context(),
			assetID,
			r.URL.Query().Get("timeframe"),
			from,
			to,
			limit,
		)
		if err != nil {
			if err == service.ErrAssetNotFound {
				WriteError(w, http.StatusNotFound, "asset_not_found", err.Error(), nil)
				return
			}
			WriteError(w, http.StatusInternalServerError, "candles_fetch_failed", err.Error(), nil)
			return
		}

		out := CandlesResponse{
			AssetID:   asset.ID,
			Timeframe: asset.Timeframe,
			Items:     make([]CandleDTO, 0, len(items)),
		}
		for _, item := range items {
			out.Items = append(out.Items, ToCandleDTO(item))
		}
		WriteJSON(w, http.StatusOK, out)
	}
}

func GetAssetFactors(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		assetID := r.PathValue("id")
		if assetID == "" {
			WriteError(w, http.StatusBadRequest, "missing_asset_id", "asset id is required", nil)
			return
		}

		from := time.Now().Add(-24 * time.Hour)
		if rawFrom := r.URL.Query().Get("from"); rawFrom != "" {
			if t, err := time.Parse(time.RFC3339, rawFrom); err == nil {
				from = t
			}
		}
		to := time.Now()
		if rawTo := r.URL.Query().Get("to"); rawTo != "" {
			if t, err := time.Parse(time.RFC3339, rawTo); err == nil {
				to = t
			}
		}

		asset, items, err := container.Services.MarketData.ListFactors(
			r.Context(),
			assetID,
			from,
			to,
		)
		if err != nil {
			if err == service.ErrAssetNotFound {
				WriteError(w, http.StatusNotFound, "asset_not_found", err.Error(), nil)
				return
			}
			WriteError(w, http.StatusInternalServerError, "factors_fetch_failed", err.Error(), nil)
			return
		}

		out := FactorsResponse{
			AssetID: asset.ID,
			Items:   make([]FactorDTO, 0, len(items)),
		}
		for _, item := range items {
			out.Items = append(out.Items, ToFactorDTO(item))
		}
		WriteJSON(w, http.StatusOK, out)
	}
}

func RefreshAsset(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int64)
		assetID := r.PathValue("id")
		if assetID == "" {
			WriteError(w, http.StatusBadRequest, "missing_asset_id", "asset id is required", nil)
			return
		}

		token, isSandbox, err := container.Services.TinkoffCredentials.GetToken(r.Context(), userID)
		if err != nil || token == "" {
			WriteError(w, http.StatusUnauthorized, "tinkoff_token_missing", "Tinkoff API token is required for refresh", nil)
			return
		}

		var req refreshAssetRequest
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&req)
		}

		err = container.Services.MarketData.RefreshTimeframe(r.Context(), userID, assetID, token, isSandbox, req.Timeframe)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "refresh_failed", err.Error(), nil)
			return
		}

		WriteJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}
}

func GetAssetIndicators(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		assetID := r.PathValue("id")
		if assetID == "" {
			WriteError(w, http.StatusBadRequest, "missing_asset_id", "asset id is required", nil)
			return
		}

		from := time.Now().Add(-7 * 24 * time.Hour)
		if rawFrom := r.URL.Query().Get("from"); rawFrom != "" {
			if t, err := time.Parse(time.RFC3339, rawFrom); err == nil {
				from = t
			}
		}
		to := time.Now()
		if rawTo := r.URL.Query().Get("to"); rawTo != "" {
			if t, err := time.Parse(time.RFC3339, rawTo); err == nil {
				to = t
			}
		}

		indicators, err := container.Services.MarketData.GetIndicatorsTimeframe(r.Context(), assetID, r.URL.Query().Get("timeframe"), from, to)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "indicators_failed", err.Error(), nil)
			return
		}

		WriteJSON(w, http.StatusOK, indicators)
	}
}

func GetAssetSummary(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		assetID := r.PathValue("id")
		if assetID == "" {
			WriteError(w, http.StatusBadRequest, "missing_asset_id", "asset id is required", nil)
			return
		}

		summary, err := container.Services.MarketData.GetSummaryTimeframe(r.Context(), assetID, r.URL.Query().Get("timeframe"))
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "summary_failed", err.Error(), nil)
			return
		}

		out := AssetSummaryResponse{
			Asset:       ToAssetDTO(summary.Asset),
			LastPrice:   summary.LastPrice,
			PriceChange: summary.PriceChange,
			DataFresh:   summary.DataFresh,
		}
		if summary.LastCandleAt != nil {
			out.LastCandleAt = summary.LastCandleAt.UTC().Format(time.RFC3339)
		}
		if summary.LastSignal != nil {
			dto := ToSignalDTO(*summary.LastSignal)
			out.LastSignal = &dto
		}

		WriteJSON(w, http.StatusOK, out)
	}
}
