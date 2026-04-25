package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"invest/backend/internal/domain"
)

type TinkoffAdapter struct {
	token   string
	baseURL string
	client  *http.Client
}

var ErrTinkoffUnavailable = errors.New("tinkoff instruments service is not configured")

func NewTinkoffAdapter(token string, target string) *TinkoffAdapter {
	baseURL := "https://invest-public-api.tbank.ru/rest/tinkoff.public.invest.api.contract.v1.InstrumentsService"
	if target == "sandbox" {
		baseURL = "https://sandbox-invest-public-api.tbank.ru/rest/tinkoff.public.invest.api.contract.v1.InstrumentsService"
	}

	return &TinkoffAdapter{
		token:   token,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			},
		},
	}
}

type getAssetsRequest struct {
	InstrumentType   string `json:"instrumentType,omitempty"`
	InstrumentStatus string `json:"instrumentStatus,omitempty"`
}

type getAssetsResponse struct {
	Assets []struct {
		Uid         string `json:"uid"`
		Type        string `json:"type"`
		Name        string `json:"name"`
		Instruments []struct {
			Uid       string `json:"uid"`
			Figi      string `json:"figi"`
			Ticker    string `json:"ticker"`
			ClassCode string `json:"classCode"`
			Isin      string `json:"isin"`
		} `json:"instruments"`
	} `json:"assets"`
}

func (a *TinkoffAdapter) FindInstrument(ctx context.Context, query string) ([]domain.TinkoffInstrument, error) {
	if a.token == "" {
		return nil, ErrTinkoffUnavailable
	}

	reqBody := getAssetsRequest{
		InstrumentType:   "INSTRUMENT_TYPE_UNSPECIFIED",
		InstrumentStatus: "INSTRUMENT_STATUS_BASE",
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	// In sandbox, FindInstrument is broken, so we fetch GetAssets and filter locally
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/GetAssets", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("tinkoff api error: %d %s", resp.StatusCode, string(body))
	}

	var payload getAssetsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	qLower := strings.ToLower(query)

	out := make([]domain.TinkoffInstrument, 0)
	for _, asset := range payload.Assets {
		if strings.Contains(strings.ToLower(asset.Name), qLower) {
			if len(asset.Instruments) > 0 {
				inst := asset.Instruments[0]
				out = append(out, domain.TinkoffInstrument{
					UID:               inst.Uid,
					Figi:              inst.Figi,
					Ticker:            inst.Ticker,
					ClassCode:         inst.ClassCode,
					Isin:              inst.Isin,
					Lot:               1,     // default
					Currency:          "rub", // default
					Name:              asset.Name,
					Exchange:          "",
					InstrumentType:    asset.Type,
					APITradeAvailable: true,
				})
			}
			continue
		}
		for _, inst := range asset.Instruments {
			if strings.Contains(strings.ToLower(inst.Ticker), qLower) || strings.Contains(strings.ToLower(inst.Uid), qLower) {
				out = append(out, domain.TinkoffInstrument{
					UID:               inst.Uid,
					Figi:              inst.Figi,
					Ticker:            inst.Ticker,
					ClassCode:         inst.ClassCode,
					Isin:              inst.Isin,
					Lot:               1,
					Currency:          "rub",
					Name:              asset.Name,
					Exchange:          "",
					InstrumentType:    asset.Type,
					APITradeAvailable: true,
				})
				break
			}
		}
	}

	return out, nil
}

type getInstrumentByRequest struct {
	IdType    string `json:"idType"`
	ClassCode string `json:"classCode,omitempty"`
	Id        string `json:"id"`
}

type getInstrumentByResponse struct {
	Instrument struct {
		Figi                  string `json:"figi"`
		Ticker                string `json:"ticker"`
		ClassCode             string `json:"classCode"`
		Isin                  string `json:"isin"`
		Lot                   int32  `json:"lot"`
		Currency              string `json:"currency"`
		Name                  string `json:"name"`
		Exchange              string `json:"exchange"`
		InstrumentType        string `json:"instrumentType"`
		Uid                   string `json:"uid"`
		ApiTradeAvailableFlag bool   `json:"apiTradeAvailableFlag"`
		First1MinCandleDate   string `json:"first1MinCandleDate"`
		First1DayCandleDate   string `json:"first1DayCandleDate"`
	} `json:"instrument"`
}

func (a *TinkoffAdapter) GetInstrumentByUID(ctx context.Context, uid string) (domain.TinkoffInstrument, error) {
	if a.token == "" {
		return domain.TinkoffInstrument{}, ErrTinkoffUnavailable
	}

	reqBody := getInstrumentByRequest{
		IdType: "INSTRUMENT_ID_TYPE_UID",
		Id:     uid,
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return domain.TinkoffInstrument{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/GetInstrumentBy", bytes.NewReader(b))
	if err != nil {
		return domain.TinkoffInstrument{}, err
	}

	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return domain.TinkoffInstrument{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return domain.TinkoffInstrument{}, fmt.Errorf("tinkoff api error: %d %s", resp.StatusCode, string(body))
	}

	var payload getInstrumentByResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return domain.TinkoffInstrument{}, err
	}

	inst := payload.Instrument
	if inst.Uid == "" {
		return domain.TinkoffInstrument{}, sql.ErrNoRows
	}
	res := domain.TinkoffInstrument{
		UID:               inst.Uid,
		Figi:              inst.Figi,
		Ticker:            inst.Ticker,
		ClassCode:         inst.ClassCode,
		Isin:              inst.Isin,
		Lot:               inst.Lot,
		Currency:          inst.Currency,
		Name:              inst.Name,
		Exchange:          inst.Exchange,
		InstrumentType:    inst.InstrumentType,
		APITradeAvailable: inst.ApiTradeAvailableFlag,
	}

	if inst.First1MinCandleDate != "" {
		if t, err := time.Parse(time.RFC3339, inst.First1MinCandleDate); err == nil {
			res.First1MinCandleDate = &t
		}
	}
	if inst.First1DayCandleDate != "" {
		if t, err := time.Parse(time.RFC3339, inst.First1DayCandleDate); err == nil {
			res.First1DayCandleDate = &t
		}
	}

	return res, nil
}
