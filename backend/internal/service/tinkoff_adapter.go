package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"invest/backend/internal/domain"
)

type TinkoffAdapter struct {
	target             string
	caCertFile         string
	instrumentsBaseURL string
	marketDataBaseURL  string
	client             *http.Client
}

var ErrTinkoffUnavailable = errors.New("tinkoff api token is required")

func NewTinkoffAdapter(target string, caCertFiles ...string) *TinkoffAdapter {
	instURL := "https://invest-public-api.tbank.ru/rest/tinkoff.public.invest.api.contract.v1.InstrumentsService"
	mdURL := "https://invest-public-api.tbank.ru/rest/tinkoff.public.invest.api.contract.v1.MarketDataService"
	if target == "sandbox" {
		instURL = "https://sandbox-invest-public-api.tbank.ru/rest/tinkoff.public.invest.api.contract.v1.InstrumentsService"
		mdURL = "https://sandbox-invest-public-api.tbank.ru/rest/tinkoff.public.invest.api.contract.v1.MarketDataService"
	}

	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	caCertFile := ""
	if len(caCertFiles) > 0 && caCertFiles[0] != "" {
		caCertFile = caCertFiles[0]
		if roots, err := loadCertPool(caCertFiles[0]); err == nil {
			tlsConfig.RootCAs = roots
		}
	}

	return &TinkoffAdapter{
		target:             target,
		caCertFile:         caCertFile,
		instrumentsBaseURL: instURL,
		marketDataBaseURL:  mdURL,
		client: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig:     tlsConfig,
				MaxIdleConns:        100,
				IdleConnTimeout:     120 * time.Second,
				MaxIdleConnsPerHost: 20,
			},
		},
	}
}

func (a *TinkoffAdapter) WithSandboxTarget(isSandbox bool) InstrumentService {
	target := "prod"
	if isSandbox {
		target = "sandbox"
	}
	if a != nil && a.target == target {
		return a
	}
	caCertFile := ""
	if a != nil {
		caCertFile = a.caCertFile
	}
	return NewTinkoffAdapter(target, caCertFile)
}

func (a *TinkoffAdapter) FindInstrument(ctx context.Context, token string, query string) ([]domain.TinkoffInstrument, error) {
	if token == "" {
		return nil, ErrTinkoffUnavailable
	}

	reqBody := getInstrumentsRequest{
		InstrumentStatus: "INSTRUMENT_STATUS_BASE",
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	endpoints := []string{"/Shares", "/Etfs", "/Currencies", "/Futures"}
	qLower := strings.ToLower(query)

	type result struct {
		instruments []domain.TinkoffInstrument
		err         error
	}
	resChan := make(chan result, len(endpoints))

	for _, ep := range endpoints {
		go func(endpoint string) {
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.instrumentsBaseURL+endpoint, bytes.NewReader(b))
			if err != nil {
				resChan <- result{err: err}
				return
			}

			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")

			resp, err := a.client.Do(req)
			if err != nil {
				resChan <- result{err: err}
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				resChan <- result{err: fmt.Errorf("tinkoff api error on %s: %d %s", endpoint, resp.StatusCode, string(body))}
				return
			}

			var payload getInstrumentsResponse
			if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
				resChan <- result{err: err}
				return
			}

			var matched []domain.TinkoffInstrument
			for _, inst := range payload.Instruments {
				if strings.Contains(strings.ToLower(inst.Name), qLower) ||
					strings.Contains(strings.ToLower(inst.Ticker), qLower) ||
					strings.Contains(strings.ToLower(inst.Uid), qLower) {

					iType := inst.InstrumentType
					if iType == "" {
						switch endpoint {
						case "/Shares":
							iType = "share"
						case "/Etfs":
							iType = "etf"
						case "/Currencies":
							iType = "currency"
						case "/Futures":
							iType = "future"
						}
					}

					matched = append(matched, domain.TinkoffInstrument{
						UID:               inst.Uid,
						Figi:              inst.Figi,
						Ticker:            inst.Ticker,
						ClassCode:         inst.ClassCode,
						Isin:              inst.Isin,
						Lot:               inst.Lot,
						Currency:          inst.Currency,
						Name:              inst.Name,
						Exchange:          inst.Exchange,
						InstrumentType:    iType,
						APITradeAvailable: inst.ApiTradeAvailableFlag,
					})
				}
			}
			resChan <- result{instruments: matched}
		}(ep)
	}

	var allInstruments []domain.TinkoffInstrument
	var firstErr error
	for i := 0; i < len(endpoints); i++ {
		res := <-resChan
		if res.err != nil {
			if firstErr == nil {
				firstErr = res.err
			}
			continue
		}
		allInstruments = append(allInstruments, res.instruments...)
	}

	if len(allInstruments) == 0 && firstErr != nil {
		return nil, firstErr
	}

	return allInstruments, nil
}

func (a *TinkoffAdapter) GetInstrumentByUID(ctx context.Context, token string, uid string) (domain.TinkoffInstrument, error) {
	if token == "" {
		return domain.TinkoffInstrument{}, ErrTinkoffUnavailable
	}
	if uid == "" {
		return domain.TinkoffInstrument{}, errors.New("instrument uid is required")
	}

	reqBody := getInstrumentByRequest{
		IdType: "INSTRUMENT_ID_TYPE_UID",
		Id:     uid,
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return domain.TinkoffInstrument{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.instrumentsBaseURL+"/GetInstrumentBy", bytes.NewReader(b))
	if err != nil {
		return domain.TinkoffInstrument{}, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
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

func (a *TinkoffAdapter) GetCandles(ctx context.Context, token string, uid string, timeframe string, from time.Time, to time.Time) ([]domain.Candle, error) {
	if token == "" {
		return nil, ErrTinkoffUnavailable
	}
	if uid == "" {
		return nil, errors.New("instrument uid is required")
	}

	interval := mapTimeframeToTinkoff(timeframe)
	if interval == "" {
		return nil, fmt.Errorf("unsupported timeframe for tinkoff: %s", timeframe)
	}
	if !from.Before(to) {
		return nil, nil
	}

	maxWindow := maxTinkoffCandleWindow(timeframe)
	out := make([]domain.Candle, 0)
	for chunkFrom := from.UTC(); chunkFrom.Before(to.UTC()); {
		chunkTo := chunkFrom.Add(maxWindow)
		if chunkTo.After(to.UTC()) {
			chunkTo = to.UTC()
		}

		items, err := a.getCandlesChunk(ctx, token, uid, timeframe, interval, chunkFrom, chunkTo)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
		chunkFrom = chunkTo
	}

	return out, nil
}

func (a *TinkoffAdapter) getCandlesChunk(
	ctx context.Context,
	token string,
	uid string,
	timeframe string,
	interval string,
	from time.Time,
	to time.Time,
) ([]domain.Candle, error) {
	reqBody := getCandlesRequest{
		InstrumentId: uid,
		From:         from.UTC().Format(time.RFC3339),
		To:           to.UTC().Format(time.RFC3339),
		Interval:     interval,
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.marketDataBaseURL+"/GetCandles", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("tinkoff api error (GetCandles): %d %s", resp.StatusCode, string(body))
	}

	var payload getCandlesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	out := make([]domain.Candle, 0, len(payload.Candles))
	ingestedAt := time.Now().UTC()

	for _, c := range payload.Candles {
		ts, err := time.Parse(time.RFC3339, c.Time)
		if err != nil {
			continue
		}

		vol, _ := parseJSONInt64(c.Volume)

		out = append(out, domain.Candle{
			Timestamp:  ts.UTC(),
			Open:       c.Open.ToFloat(),
			High:       c.High.ToFloat(),
			Low:        c.Low.ToFloat(),
			Close:      c.Close.ToFloat(),
			Volume:     vol,
			Timeframe:  timeframe,
			Source:     "tinkoff",
			IngestedAt: ingestedAt,
		})
	}

	return out, nil
}

func (a *TinkoffAdapter) IsMarketOpen(ctx context.Context, token string, exchange string) (bool, error) {
	if token == "" {
		return false, ErrTinkoffUnavailable
	}

	now := time.Now().UTC()
	reqBody := getTradingSchedulesRequest{
		Exchange: exchange,
		From:     now.Format(time.RFC3339),
		To:       now.Format(time.RFC3339),
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.instrumentsBaseURL+"/GetTradingSchedules", bytes.NewReader(b))
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return true, nil
	}

	var payload getTradingSchedulesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return true, nil
	}

	for _, ex := range payload.Exchanges {
		if ex.Exchange == exchange {
			for _, day := range ex.Days {
				if !day.IsTradingDay {
					return false, nil
				}

				start, _ := time.Parse(time.RFC3339, day.StartTime)
				end, _ := time.Parse(time.RFC3339, day.EndTime)

				if now.Before(start) || now.After(end) {
					return false, nil
				}
				return true, nil
			}
		}
	}

	return true, nil
}

func loadCertPool(path string) (*x509.CertPool, error) {
	roots, err := x509.SystemCertPool()
	if err != nil || roots == nil {
		roots = x509.NewCertPool()
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if !roots.AppendCertsFromPEM(raw) {
		return nil, fmt.Errorf("no PEM certificates found in %s", path)
	}
	return roots, nil
}

type getInstrumentsRequest struct {
	InstrumentStatus string `json:"instrumentStatus,omitempty"`
}

type getInstrumentsResponse struct {
	Instruments []struct {
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
	} `json:"instruments"`
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

type getCandlesRequest struct {
	Figi         string `json:"figi,omitempty"`
	From         string `json:"from"`
	To           string `json:"to"`
	Interval     string `json:"interval"`
	InstrumentId string `json:"instrumentId,omitempty"`
}

type tinkoffQuotation struct {
	Units int64 `json:"units"`
	Nano  int32 `json:"nano"`
}

func (q *tinkoffQuotation) UnmarshalJSON(raw []byte) error {
	var payload struct {
		Units json.RawMessage `json:"units"`
		Nano  json.RawMessage `json:"nano"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	units, err := parseJSONInt64(payload.Units)
	if err != nil {
		return fmt.Errorf("units: %w", err)
	}
	nano, err := parseJSONInt64(payload.Nano)
	if err != nil {
		return fmt.Errorf("nano: %w", err)
	}
	q.Units = units
	q.Nano = int32(nano)
	return nil
}

func (q tinkoffQuotation) ToFloat() float64 {
	return float64(q.Units) + float64(q.Nano)/1e9
}

func parseJSONInt64(raw json.RawMessage) (int64, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, nil
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return strconv.ParseInt(asString, 10, 64)
	}
	var asNumber int64
	if err := json.Unmarshal(raw, &asNumber); err != nil {
		return 0, err
	}
	return asNumber, nil
}

type getCandlesResponse struct {
	Candles []struct {
		Open       tinkoffQuotation `json:"open"`
		High       tinkoffQuotation `json:"high"`
		Low        tinkoffQuotation `json:"low"`
		Close      tinkoffQuotation `json:"close"`
		Volume     json.RawMessage  `json:"volume"`
		Time       string           `json:"time"`
		IsComplete bool             `json:"isComplete"`
	} `json:"candles"`
}

type getTradingSchedulesRequest struct {
	Exchange string `json:"exchange"`
	From     string `json:"from"`
	To       string `json:"to"`
}

type getTradingSchedulesResponse struct {
	Exchanges []struct {
		Exchange string `json:"exchange"`
		Days     []struct {
			Date                    string `json:"date"`
			IsTradingDay            bool   `json:"isTradingDay"`
			StartTime               string `json:"startTime"`
			EndTime                 string `json:"endTime"`
			OpeningAuctionStartTime string `json:"openingAuctionStartTime"`
			ClosingAuctionEndTime   string `json:"closingAuctionEndTime"`
		} `json:"days"`
	} `json:"exchanges"`
}

func mapTimeframeToTinkoff(tf string) string {
	switch tf {
	case "1m":
		return "CANDLE_INTERVAL_1_MIN"
	case "5m":
		return "CANDLE_INTERVAL_5_MIN"
	case "15m":
		return "CANDLE_INTERVAL_15_MIN"
	case "1h":
		return "CANDLE_INTERVAL_HOUR"
	case "1d":
		return "CANDLE_INTERVAL_DAY"
	default:
		return ""
	}
}

func maxTinkoffCandleWindow(tf string) time.Duration {
	switch tf {
	case "1m":
		return 24 * time.Hour
	case "5m", "15m":
		return 24 * time.Hour
	case "1h":
		return 7 * 24 * time.Hour
	case "1d":
		return 365 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}
