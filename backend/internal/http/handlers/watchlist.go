package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"invest/backend/internal/app"
	"invest/backend/internal/domain"
	"invest/backend/internal/service"
)

func GetWatchlist(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int64)

		watchlist, err := container.Services.Watchlist.GetUserWatchlist(r.Context(), userID)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "watchlist_init_failed", err.Error(), nil)
			return
		}
		items, err := container.Services.Watchlist.ListItems(r.Context(), userID)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "watchlist_list_failed", err.Error(), nil)
			return
		}

		out := WatchlistResponse{
			WatchlistID: watchlist.ID,
			Name:        watchlist.Name,
			Items:       make([]WatchlistItemDTO, 0, len(items)),
		}
		for _, item := range items {
			out.Items = append(out.Items, ToWatchlistItemDTO(r.Context(), container, item))
		}

		WriteJSON(w, http.StatusOK, out)
	}
}

func UpsertWatchlistItem(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int64)

		var req WatchlistUpsertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request payload", nil)
			return
		}
		assetID := req.AssetID
		if assetID == "" && req.InstrumentUID == "" {
			WriteError(w, http.StatusBadRequest, "validation_error", "asset_id or instrument_uid is required", nil)
			return
		}

		if req.InstrumentUID != "" && container.Services.Instruments != nil {
			token := ""
			isSandbox := false
			if container.Services.TinkoffCredentials != nil {
				var err error
				token, isSandbox, err = container.Services.TinkoffCredentials.GetToken(r.Context(), userID)
				if err != nil || token == "" {
					WriteError(w, http.StatusUnauthorized, "tinkoff_token_missing", "Tinkoff API token is required for instrument lookup", nil)
					return
				}
			}
			instruments := service.InstrumentServiceForSandbox(container.Services.Instruments, isSandbox)
			inst, err := instruments.GetInstrumentByUID(r.Context(), token, req.InstrumentUID)
			if err != nil {
				WriteError(w, http.StatusNotFound, "instrument_not_found", err.Error(), nil)
				return
			}
			asset, err := upsertInstrumentAsset(r.Context(), container, inst)
			if err != nil {
				WriteError(w, http.StatusInternalServerError, "asset_upsert_failed", err.Error(), nil)
				return
			}
			assetID = asset.ID
		}

		if assetID == "" {
			WriteError(w, http.StatusBadRequest, "validation_error", "asset_id or instrument_uid is required", nil)
			return
		}

		err := container.Services.Watchlist.AddItem(r.Context(), userID, assetID, req.Position)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "watchlist_add_failed", err.Error(), nil)
			return
		}

		writeCurrentWatchlist(w, r, container, userID, http.StatusCreated)
	}
}

func RemoveWatchlistItem(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int64)
		assetID := r.PathValue("asset_id")
		if assetID == "" {
			WriteError(w, http.StatusBadRequest, "validation_error", "asset_id is required", nil)
			return
		}

		err := container.Services.Watchlist.RemoveItem(r.Context(), userID, assetID)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "watchlist_remove_failed", err.Error(), nil)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func GetWatchlistFreshness(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if container.Services.WatchlistRefresh == nil {
			WriteError(w, http.StatusServiceUnavailable, "freshness_unavailable", "watchlist refresh service is not configured", nil)
			return
		}
		userID := r.Context().Value("user_id").(int64)
		summary, err := container.Services.WatchlistRefresh.GetFreshnessForUser(r.Context(), userID)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "freshness_failed", err.Error(), nil)
			return
		}
		WriteJSON(w, http.StatusOK, summary)
	}
}

func writeCurrentWatchlist(w http.ResponseWriter, r *http.Request, container app.Container, userID int64, status int) {
	watchlist, err := container.Services.Watchlist.GetUserWatchlist(r.Context(), userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "watchlist_init_failed", err.Error(), nil)
		return
	}
	items, err := container.Services.Watchlist.ListItems(r.Context(), userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "watchlist_list_failed", err.Error(), nil)
		return
	}
	out := WatchlistResponse{
		WatchlistID: watchlist.ID,
		Name:        watchlist.Name,
		Items:       make([]WatchlistItemDTO, 0, len(items)),
	}
	for _, item := range items {
		out.Items = append(out.Items, ToWatchlistItemDTO(r.Context(), container, item))
	}
	WriteJSON(w, status, out)
}

func upsertInstrumentAsset(ctx context.Context, container app.Container, inst domain.TinkoffInstrument) (domain.Asset, error) {
	if container.Services.Assets == nil {
		return domain.Asset{}, service.ErrAssetNotFound
	}

	domainAsset := findExistingInstrumentAsset(ctx, container, inst)
	if domainAsset.ID == "" {
		domainAsset.ID = inst.Ticker
		if domainAsset.ID == "" {
			domainAsset.ID = inst.UID
		}
		domainAsset.Timeframe = "5m"
		domainAsset.IsActive = true
	}
	if domainAsset.Timeframe == "" {
		domainAsset.Timeframe = "5m"
	}

	domainAsset.Ticker = inst.Ticker
	domainAsset.Name = inst.Name
	domainAsset.Exchange = inst.Exchange
	domainAsset.Figi = inst.Figi
	domainAsset.InstrumentUID = inst.UID
	domainAsset.ClassCode = inst.ClassCode
	domainAsset.InstrumentType = inst.InstrumentType
	domainAsset.Lot = inst.Lot
	domainAsset.Currency = inst.Currency
	domainAsset.APITradeAvailable = inst.APITradeAvailable
	if inst.First1MinCandleDate != nil {
		domainAsset.First1MinCandleDate = inst.First1MinCandleDate
	}
	if inst.First1DayCandleDate != nil {
		domainAsset.First1DayCandleDate = inst.First1DayCandleDate
	}

	if err := container.Services.Assets.Upsert(ctx, domainAsset); err != nil {
		return domain.Asset{}, err
	}
	return domainAsset, nil
}

func findExistingInstrumentAsset(ctx context.Context, container app.Container, inst domain.TinkoffInstrument) domain.Asset {
	if container.Services.Assets == nil {
		return domain.Asset{}
	}
	items, err := container.Services.Assets.List(ctx)
	if err != nil {
		return domain.Asset{}
	}
	for _, asset := range items {
		if inst.UID != "" && asset.InstrumentUID == inst.UID {
			return asset
		}
	}
	for _, asset := range items {
		if !strings.EqualFold(asset.Ticker, inst.Ticker) {
			continue
		}
		if asset.ClassCode != "" && inst.ClassCode != "" && !strings.EqualFold(asset.ClassCode, inst.ClassCode) {
			continue
		}
		return asset
	}
	return domain.Asset{}
}
