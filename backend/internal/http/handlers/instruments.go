package handlers

import (
	"net/http"

	"invest/backend/internal/app"
	"invest/backend/internal/service"
)

type InstrumentsSearchResponse struct {
	Items []InstrumentDTO `json:"items"`
}

func SearchInstruments(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int64)
		query := r.URL.Query().Get("query")
		if query == "" {
			WriteError(w, http.StatusBadRequest, "missing_query", "query parameter is required", nil)
			return
		}

		if container.Services.Instruments == nil {
			WriteError(w, http.StatusServiceUnavailable, "instruments_unavailable", "instrument service is not configured", nil)
			return
		}

		token := ""
		isSandbox := false
		if container.Services.TinkoffCredentials != nil {
			var err error
			token, isSandbox, err = container.Services.TinkoffCredentials.GetToken(r.Context(), userID)
			if err != nil || token == "" {
				WriteError(w, http.StatusUnauthorized, "tinkoff_token_missing", "Tinkoff API token is required for search", nil)
				return
			}
		}

		instruments := service.InstrumentServiceForSandbox(container.Services.Instruments, isSandbox)
		insts, err := instruments.FindInstrument(r.Context(), token, query)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "instrument_search_failed", err.Error(), nil)
			return
		}

		// Proactively cache found instruments to local assets table to avoid future GetInstrumentByUID calls
		if container.Services.Assets != nil {
			for _, inst := range insts {
				_, _ = upsertInstrumentAsset(r.Context(), container, inst)
			}
		}

		out := InstrumentsSearchResponse{Items: make([]InstrumentDTO, 0, len(insts))}
		for _, inst := range insts {
			out.Items = append(out.Items, ToInstrumentDTO(inst))
		}
		WriteJSON(w, http.StatusOK, out)
	}
}

func GetInstrument(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int64)
		uid := r.PathValue("uid")
		if uid == "" {
			WriteError(w, http.StatusBadRequest, "missing_uid", "uid is required", nil)
			return
		}

		// Try local cache first
		if container.Services.Assets != nil {
			if asset, err := container.Services.Assets.GetByID(r.Context(), uid); err == nil {
				// Map asset back to instrument DTO
				WriteJSON(w, http.StatusOK, InstrumentDTO{
					UID:               asset.InstrumentUID,
					Figi:              asset.Figi,
					Ticker:            asset.Ticker,
					ClassCode:         asset.ClassCode,
					Lot:               asset.Lot,
					Currency:          asset.Currency,
					Name:              asset.Name,
					Exchange:          asset.Exchange,
					InstrumentType:    asset.InstrumentType,
					APITradeAvailable: asset.APITradeAvailable,
				})
				return
			}
		}

		if container.Services.Instruments == nil {
			WriteError(w, http.StatusServiceUnavailable, "instruments_unavailable", "instrument service is not configured", nil)
			return
		}

		token := ""
		isSandbox := false
		if container.Services.TinkoffCredentials != nil {
			var err error
			token, isSandbox, err = container.Services.TinkoffCredentials.GetToken(r.Context(), userID)
			if err != nil || token == "" {
				WriteError(w, http.StatusUnauthorized, "tinkoff_token_missing", "Tinkoff API token is required", nil)
				return
			}
		}

		instruments := service.InstrumentServiceForSandbox(container.Services.Instruments, isSandbox)
		inst, err := instruments.GetInstrumentByUID(r.Context(), token, uid)
		if err != nil {
			WriteError(w, http.StatusNotFound, "instrument_not_found", err.Error(), nil)
			return
		}

		WriteJSON(w, http.StatusOK, ToInstrumentDTO(inst))
	}
}

func EnsureInstrumentAsset(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int64)
		uid := r.PathValue("uid")
		if uid == "" {
			WriteError(w, http.StatusBadRequest, "missing_uid", "uid is required", nil)
			return
		}

		// Try local cache first
		if container.Services.Assets != nil {
			if asset, err := container.Services.Assets.GetByID(r.Context(), uid); err == nil {
				WriteJSON(w, http.StatusOK, ToAssetDTO(asset))
				return
			}
		}

		if container.Services.Instruments == nil {
			WriteError(w, http.StatusServiceUnavailable, "instruments_unavailable", "instrument service is not configured", nil)
			return
		}

		token := ""
		isSandbox := false
		if container.Services.TinkoffCredentials != nil {
			var err error
			token, isSandbox, err = container.Services.TinkoffCredentials.GetToken(r.Context(), userID)
			if err != nil || token == "" {
				WriteError(w, http.StatusUnauthorized, "tinkoff_token_missing", "Tinkoff API token is required", nil)
				return
			}
		}

		instruments := service.InstrumentServiceForSandbox(container.Services.Instruments, isSandbox)
		inst, err := instruments.GetInstrumentByUID(r.Context(), token, uid)
		if err != nil {
			WriteError(w, http.StatusNotFound, "instrument_not_found", err.Error(), nil)
			return
		}
		asset, err := upsertInstrumentAsset(r.Context(), container, inst)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "asset_upsert_failed", err.Error(), nil)
			return
		}

		WriteJSON(w, http.StatusOK, ToAssetDTO(asset))
	}
}
