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
	usersBaseURL       string
	instrumentsBaseURL string
	marketDataBaseURL  string
	operationsBaseURL  string
	ordersBaseURL      string
	sandboxBaseURL     string
	client             *http.Client
}

var ErrTinkoffUnavailable = errors.New("tinkoff api token is required")

func NewTinkoffAdapter(target string, caCertFiles ...string) *TinkoffAdapter {
	usersURL := "https://invest-public-api.tbank.ru/rest/tinkoff.public.invest.api.contract.v1.UsersService"
	instURL := "https://invest-public-api.tbank.ru/rest/tinkoff.public.invest.api.contract.v1.InstrumentsService"
	mdURL := "https://invest-public-api.tbank.ru/rest/tinkoff.public.invest.api.contract.v1.MarketDataService"
	opsURL := "https://invest-public-api.tbank.ru/rest/tinkoff.public.invest.api.contract.v1.OperationsService"
	ordersURL := "https://invest-public-api.tbank.ru/rest/tinkoff.public.invest.api.contract.v1.OrdersService"
	sandboxURL := "https://invest-public-api.tbank.ru/rest/tinkoff.public.invest.api.contract.v1.SandboxService"
	if target == "sandbox" {
		usersURL = "https://sandbox-invest-public-api.tbank.ru/rest/tinkoff.public.invest.api.contract.v1.UsersService"
		instURL = "https://sandbox-invest-public-api.tbank.ru/rest/tinkoff.public.invest.api.contract.v1.InstrumentsService"
		mdURL = "https://sandbox-invest-public-api.tbank.ru/rest/tinkoff.public.invest.api.contract.v1.MarketDataService"
		opsURL = "https://sandbox-invest-public-api.tbank.ru/rest/tinkoff.public.invest.api.contract.v1.OperationsService"
		ordersURL = "https://sandbox-invest-public-api.tbank.ru/rest/tinkoff.public.invest.api.contract.v1.OrdersService"
		sandboxURL = "https://sandbox-invest-public-api.tbank.ru/rest/tinkoff.public.invest.api.contract.v1.SandboxService"
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
		usersBaseURL:       usersURL,
		instrumentsBaseURL: instURL,
		marketDataBaseURL:  mdURL,
		operationsBaseURL:  opsURL,
		ordersBaseURL:      ordersURL,
		sandboxBaseURL:     sandboxURL,
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

func (a *TinkoffAdapter) GetAccounts(ctx context.Context, token string) ([]domain.BrokerAccount, error) {
	if token == "" {
		return nil, ErrTinkoffUnavailable
	}
	var payload getAccountsResponse
	endpoint := a.usersBaseURL + "/GetAccounts"
	if a.target == "sandbox" {
		endpoint = a.sandboxBaseURL + "/GetSandboxAccounts"
	}
	if err := a.postJSON(ctx, endpoint, token, map[string]any{"status": "ACCOUNT_STATUS_ALL"}, &payload); err != nil {
		return nil, err
	}
	out := make([]domain.BrokerAccount, 0, len(payload.Accounts))
	for _, item := range payload.Accounts {
		out = append(out, domain.BrokerAccount{
			ID:          item.ID,
			Type:        item.Type,
			Name:        item.Name,
			Status:      item.Status,
			OpenedDate:  parseOptionalRFC3339(item.OpenedDate),
			ClosedDate:  parseOptionalRFC3339(item.ClosedDate),
			AccessLevel: item.AccessLevel,
		})
	}
	return out, nil
}

func (a *TinkoffAdapter) GetPortfolio(ctx context.Context, token string, accountID string) (domain.BrokerPortfolio, error) {
	if token == "" {
		return domain.BrokerPortfolio{}, ErrTinkoffUnavailable
	}
	var payload getPortfolioResponse
	endpoint := a.operationsBaseURL + "/GetPortfolio"
	if a.target == "sandbox" {
		endpoint = a.sandboxBaseURL + "/GetSandboxPortfolio"
	}
	if err := a.postJSON(ctx, endpoint, token, map[string]any{"accountId": accountID, "currency": "RUB"}, &payload); err != nil {
		return domain.BrokerPortfolio{}, err
	}
	out := domain.BrokerPortfolio{
		AccountID:             coalesceString(payload.AccountID, accountID),
		TotalAmountPortfolio:  payload.TotalAmountPortfolio.ToDomain(),
		TotalAmountShares:     payload.TotalAmountShares.ToDomain(),
		TotalAmountBonds:      payload.TotalAmountBonds.ToDomain(),
		TotalAmountETF:        payload.TotalAmountETF.ToDomain(),
		TotalAmountCurrencies: payload.TotalAmountCurrencies.ToDomain(),
		TotalAmountFutures:    payload.TotalAmountFutures.ToDomain(),
		TotalAmountOptions:    payload.TotalAmountOptions.ToDomain(),
		ExpectedYield:         payload.ExpectedYield.ToFloat(),
		DailyYield:            payload.DailyYield.ToDomain(),
		DailyYieldRelative:    payload.DailyYieldRelative.ToFloat(),
		Positions:             make([]domain.BrokerPortfolioPosition, 0, len(payload.Positions)),
	}
	for _, item := range payload.Positions {
		out.Positions = append(out.Positions, domain.BrokerPortfolioPosition{
			Figi:                 item.Figi,
			InstrumentType:       item.InstrumentType,
			Quantity:             item.Quantity.ToFloat(),
			AveragePositionPrice: item.AveragePositionPrice.ToDomain(),
			ExpectedYield:        item.ExpectedYield.ToFloat(),
			CurrentPrice:         item.CurrentPrice.ToDomain(),
			Blocked:              item.Blocked,
			BlockedLots:          item.BlockedLots.ToFloat(),
			PositionUID:          item.PositionUID,
			InstrumentUID:        item.InstrumentUID,
			Ticker:               item.Ticker,
			ClassCode:            item.ClassCode,
			DailyYield:           item.DailyYield.ToDomain(),
		})
	}
	return out, nil
}

func (a *TinkoffAdapter) GetPositions(ctx context.Context, token string, accountID string) (domain.BrokerPositions, error) {
	if token == "" {
		return domain.BrokerPositions{}, ErrTinkoffUnavailable
	}
	var payload getPositionsResponse
	endpoint := a.operationsBaseURL + "/GetPositions"
	if a.target == "sandbox" {
		endpoint = a.sandboxBaseURL + "/GetSandboxPositions"
	}
	if err := a.postJSON(ctx, endpoint, token, map[string]any{"accountId": accountID}, &payload); err != nil {
		return domain.BrokerPositions{}, err
	}
	out := domain.BrokerPositions{
		AccountID:               coalesceString(payload.AccountID, accountID),
		Money:                   mapMoneyValues(payload.Money),
		Blocked:                 mapMoneyValues(payload.Blocked),
		Securities:              mapSecurityPositions(payload.Securities),
		Futures:                 mapSecurityPositions(payload.Futures),
		Options:                 mapSecurityPositions(payload.Options),
		LimitsLoadingInProgress: payload.LimitsLoadingInProgress,
	}
	return out, nil
}

func (a *TinkoffAdapter) GetOperationsByCursor(ctx context.Context, token string, request domain.BrokerOperationsRequest) (domain.BrokerOperationsPage, error) {
	if token == "" {
		return domain.BrokerOperationsPage{}, ErrTinkoffUnavailable
	}
	limit := request.Limit
	if limit <= 0 || limit > 1000 {
		limit = 50
	}
	body := map[string]any{
		"accountId":          request.AccountID,
		"limit":              limit,
		"withoutCommissions": false,
		"withoutTrades":      true,
		"withoutOvernights":  false,
	}
	if !request.From.IsZero() {
		body["from"] = request.From.UTC().Format(time.RFC3339)
	}
	if !request.To.IsZero() {
		body["to"] = request.To.UTC().Format(time.RFC3339)
	}
	if request.Cursor != "" {
		body["cursor"] = request.Cursor
	}
	var payload getOperationsByCursorResponse
	endpoint := a.operationsBaseURL + "/GetOperationsByCursor"
	if a.target == "sandbox" {
		endpoint = a.sandboxBaseURL + "/GetSandboxOperationsByCursor"
	}
	if err := a.postJSON(ctx, endpoint, token, body, &payload); err != nil {
		return domain.BrokerOperationsPage{}, err
	}
	out := domain.BrokerOperationsPage{
		HasNext:    payload.HasNext,
		NextCursor: payload.NextCursor,
		Items:      make([]domain.BrokerOperation, 0, len(payload.Items)),
	}
	for _, item := range payload.Items {
		quantityDone, _ := parseJSONInt64(item.QuantityDone)
		out.Items = append(out.Items, domain.BrokerOperation{
			Cursor:          item.Cursor,
			BrokerAccountID: item.BrokerAccountID,
			ID:              item.ID,
			Name:            item.Name,
			Date:            parseOptionalRFC3339(item.Date),
			Type:            item.Type,
			Description:     item.Description,
			State:           item.State,
			InstrumentUID:   item.InstrumentUID,
			QuantityDone:    quantityDone,
		})
	}
	return out, nil
}

func (a *TinkoffAdapter) GetOrders(ctx context.Context, token string, accountID string) ([]domain.BrokerOrder, error) {
	if token == "" {
		return nil, ErrTinkoffUnavailable
	}
	var payload getOrdersResponse
	endpoint := a.ordersBaseURL + "/GetOrders"
	if a.target == "sandbox" {
		endpoint = a.sandboxBaseURL + "/GetSandboxOrders"
	}
	if err := a.postJSON(ctx, endpoint, token, map[string]any{"accountId": accountID}, &payload); err != nil {
		return nil, err
	}
	out := make([]domain.BrokerOrder, 0, len(payload.Orders))
	for _, item := range payload.Orders {
		out = append(out, item.ToDomain())
	}
	return out, nil
}

func (a *TinkoffAdapter) PostOrder(ctx context.Context, token string, request domain.BrokerPlaceOrderRequest) (domain.BrokerOrder, error) {
	if token == "" {
		return domain.BrokerOrder{}, ErrTinkoffUnavailable
	}
	body := map[string]any{
		"accountId":    request.AccountID,
		"instrumentId": request.InstrumentID,
		"quantity":     strconv.FormatInt(request.Quantity, 10),
		"direction":    request.Direction,
		"orderType":    request.OrderType,
		"orderId":      request.OrderID,
	}
	if request.OrderType == "ORDER_TYPE_LIMIT" || strings.TrimSpace(request.PriceDecimal) != "" {
		price, err := decimalStringToQuotation(request.PriceDecimal)
		if err != nil {
			return domain.BrokerOrder{}, err
		}
		body["price"] = price
	}
	var payload tinkoffOrderState
	endpoint := a.ordersBaseURL + "/PostOrder"
	if a.target == "sandbox" {
		endpoint = a.sandboxBaseURL + "/PostSandboxOrder"
	}
	if err := a.postJSON(ctx, endpoint, token, body, &payload); err != nil {
		return domain.BrokerOrder{}, err
	}
	return payload.ToDomain(), nil
}

func (a *TinkoffAdapter) CancelOrder(ctx context.Context, token string, accountID string, orderID string) (domain.BrokerCancelOrderResult, error) {
	if token == "" {
		return domain.BrokerCancelOrderResult{}, ErrTinkoffUnavailable
	}
	var payload cancelOrderResponse
	endpoint := a.ordersBaseURL + "/CancelOrder"
	if a.target == "sandbox" {
		endpoint = a.sandboxBaseURL + "/CancelSandboxOrder"
	}
	if err := a.postJSON(ctx, endpoint, token, map[string]any{"accountId": accountID, "orderId": orderID}, &payload); err != nil {
		return domain.BrokerCancelOrderResult{}, err
	}
	return domain.BrokerCancelOrderResult{
		AccountID: accountID,
		OrderID:   orderID,
		Time:      parseOptionalRFC3339(payload.Time),
	}, nil
}

func (a *TinkoffAdapter) GetOrderState(ctx context.Context, token string, accountID string, orderID string) (domain.BrokerOrder, error) {
	if token == "" {
		return domain.BrokerOrder{}, ErrTinkoffUnavailable
	}
	var payload tinkoffOrderState
	endpoint := a.ordersBaseURL + "/GetOrderState"
	if a.target == "sandbox" {
		endpoint = a.sandboxBaseURL + "/GetSandboxOrderState"
	}
	if err := a.postJSON(ctx, endpoint, token, map[string]any{"accountId": accountID, "orderId": orderID}, &payload); err != nil {
		return domain.BrokerOrder{}, err
	}
	return payload.ToDomain(), nil
}

func (a *TinkoffAdapter) OpenSandboxAccount(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", ErrTinkoffUnavailable
	}
	var payload openSandboxAccountResponse
	if err := a.postJSON(ctx, a.sandboxBaseURL+"/OpenSandboxAccount", token, map[string]any{}, &payload); err != nil {
		return "", err
	}
	if payload.AccountID == "" {
		return "", sql.ErrNoRows
	}
	return payload.AccountID, nil
}

func (a *TinkoffAdapter) SandboxPayIn(ctx context.Context, token string, accountID string, amount domain.MoneyValue) (domain.BrokerSandboxPayInResult, error) {
	if token == "" {
		return domain.BrokerSandboxPayInResult{}, ErrTinkoffUnavailable
	}
	var payload sandboxPayInResponse
	if err := a.postJSON(ctx, a.sandboxBaseURL+"/SandboxPayIn", token, map[string]any{
		"accountId": accountID,
		"amount": tinkoffMoneyValue{
			Currency: amount.Currency,
			Units:    amount.Units,
			Nano:     amount.Nano,
		},
	}, &payload); err != nil {
		return domain.BrokerSandboxPayInResult{}, err
	}
	return domain.BrokerSandboxPayInResult{AccountID: accountID, Balance: payload.Balance.ToDomain()}, nil
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

func (a *TinkoffAdapter) postJSON(ctx context.Context, url string, token string, request any, response any) error {
	b, err := json.Marshal(request)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("tinkoff api error: %d %s", resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(response)
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

type tinkoffMoneyValue struct {
	Currency string `json:"currency"`
	Units    int64  `json:"units"`
	Nano     int32  `json:"nano"`
}

func (m *tinkoffMoneyValue) UnmarshalJSON(raw []byte) error {
	var payload struct {
		Currency string          `json:"currency"`
		Units    json.RawMessage `json:"units"`
		Nano     json.RawMessage `json:"nano"`
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
	m.Currency = payload.Currency
	m.Units = units
	m.Nano = int32(nano)
	return nil
}

func (m tinkoffMoneyValue) ToDomain() domain.MoneyValue {
	amount := float64(m.Units) + float64(m.Nano)/1e9
	return domain.MoneyValue{
		Currency: m.Currency,
		Units:    m.Units,
		Nano:     m.Nano,
		Amount:   amount,
	}
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

type getAccountsResponse struct {
	Accounts []struct {
		ID          string `json:"id"`
		Type        string `json:"type"`
		Name        string `json:"name"`
		Status      string `json:"status"`
		OpenedDate  string `json:"openedDate"`
		ClosedDate  string `json:"closedDate"`
		AccessLevel string `json:"accessLevel"`
	} `json:"accounts"`
}

type getPortfolioResponse struct {
	TotalAmountShares     tinkoffMoneyValue `json:"totalAmountShares"`
	TotalAmountBonds      tinkoffMoneyValue `json:"totalAmountBonds"`
	TotalAmountETF        tinkoffMoneyValue `json:"totalAmountEtf"`
	TotalAmountCurrencies tinkoffMoneyValue `json:"totalAmountCurrencies"`
	TotalAmountFutures    tinkoffMoneyValue `json:"totalAmountFutures"`
	TotalAmountOptions    tinkoffMoneyValue `json:"totalAmountOptions"`
	TotalAmountPortfolio  tinkoffMoneyValue `json:"totalAmountPortfolio"`
	ExpectedYield         tinkoffQuotation  `json:"expectedYield"`
	DailyYield            tinkoffMoneyValue `json:"dailyYield"`
	DailyYieldRelative    tinkoffQuotation  `json:"dailyYieldRelative"`
	AccountID             string            `json:"accountId"`
	Positions             []struct {
		Figi                 string            `json:"figi"`
		InstrumentType       string            `json:"instrumentType"`
		Quantity             tinkoffQuotation  `json:"quantity"`
		AveragePositionPrice tinkoffMoneyValue `json:"averagePositionPrice"`
		ExpectedYield        tinkoffQuotation  `json:"expectedYield"`
		CurrentPrice         tinkoffMoneyValue `json:"currentPrice"`
		Blocked              bool              `json:"blocked"`
		BlockedLots          tinkoffQuotation  `json:"blockedLots"`
		PositionUID          string            `json:"positionUid"`
		InstrumentUID        string            `json:"instrumentUid"`
		Ticker               string            `json:"ticker"`
		ClassCode            string            `json:"classCode"`
		DailyYield           tinkoffMoneyValue `json:"dailyYield"`
	} `json:"positions"`
}

type getPositionsResponse struct {
	Money                   []tinkoffMoneyValue       `json:"money"`
	Blocked                 []tinkoffMoneyValue       `json:"blocked"`
	Securities              []tinkoffSecurityPosition `json:"securities"`
	Futures                 []tinkoffSecurityPosition `json:"futures"`
	Options                 []tinkoffSecurityPosition `json:"options"`
	LimitsLoadingInProgress bool                      `json:"limitsLoadingInProgress"`
	AccountID               string                    `json:"accountId"`
}

type tinkoffSecurityPosition struct {
	Figi          string          `json:"figi"`
	InstrumentUID string          `json:"instrumentUid"`
	Ticker        string          `json:"ticker"`
	ClassCode     string          `json:"classCode"`
	Blocked       json.RawMessage `json:"blocked"`
	Balance       json.RawMessage `json:"balance"`
}

type getOperationsByCursorResponse struct {
	HasNext    bool   `json:"hasNext"`
	NextCursor string `json:"nextCursor"`
	Items      []struct {
		Cursor          string          `json:"cursor"`
		BrokerAccountID string          `json:"brokerAccountId"`
		ID              string          `json:"id"`
		Name            string          `json:"name"`
		Date            string          `json:"date"`
		Type            string          `json:"type"`
		Description     string          `json:"description"`
		State           string          `json:"state"`
		InstrumentUID   string          `json:"instrumentUid"`
		QuantityDone    json.RawMessage `json:"quantityDone"`
	} `json:"items"`
}

type getOrdersResponse struct {
	Orders []tinkoffOrderState `json:"orders"`
}

type tinkoffOrderState struct {
	OrderID               string            `json:"orderId"`
	OrderRequestID        string            `json:"orderRequestId"`
	ExecutionReportStatus string            `json:"executionReportStatus"`
	LotsRequested         json.RawMessage   `json:"lotsRequested"`
	LotsExecuted          json.RawMessage   `json:"lotsExecuted"`
	LotsLeft              json.RawMessage   `json:"lotsLeft"`
	InitialOrderPrice     tinkoffMoneyValue `json:"initialOrderPrice"`
	ExecutedOrderPrice    tinkoffMoneyValue `json:"executedOrderPrice"`
	TotalOrderAmount      tinkoffMoneyValue `json:"totalOrderAmount"`
	InitialSecurityPrice  tinkoffMoneyValue `json:"initialSecurityPrice"`
	Direction             string            `json:"direction"`
	OrderType             string            `json:"orderType"`
	InstrumentUID         string            `json:"instrumentUid"`
	Figi                  string            `json:"figi"`
	OrderDate             string            `json:"orderDate"`
	Message               string            `json:"message"`
}

func (item tinkoffOrderState) ToDomain() domain.BrokerOrder {
	lotsRequested, _ := parseJSONInt64(item.LotsRequested)
	lotsExecuted, _ := parseJSONInt64(item.LotsExecuted)
	lotsLeft, _ := parseJSONInt64(item.LotsLeft)
	initialOrderPrice := item.InitialOrderPrice.ToDomain()
	if initialOrderPrice.Currency == "" && item.TotalOrderAmount.Currency != "" {
		initialOrderPrice = item.TotalOrderAmount.ToDomain()
	}
	return domain.BrokerOrder{
		OrderID:               item.OrderID,
		OrderRequestID:        item.OrderRequestID,
		ExecutionReportStatus: item.ExecutionReportStatus,
		LotsRequested:         lotsRequested,
		LotsExecuted:          lotsExecuted,
		LotsLeft:              lotsLeft,
		InitialOrderPrice:     initialOrderPrice,
		ExecutedOrderPrice:    item.ExecutedOrderPrice.ToDomain(),
		InitialSecurityPrice:  item.InitialSecurityPrice.ToDomain(),
		Direction:             item.Direction,
		OrderType:             item.OrderType,
		InstrumentUID:         item.InstrumentUID,
		Figi:                  item.Figi,
		CreatedAt:             parseOptionalRFC3339(item.OrderDate),
		Message:               item.Message,
	}
}

type cancelOrderResponse struct {
	Time string `json:"time"`
}

type openSandboxAccountResponse struct {
	AccountID string `json:"accountId"`
}

type sandboxPayInResponse struct {
	Balance tinkoffMoneyValue `json:"balance"`
}

func parseOptionalRFC3339(raw string) *time.Time {
	if raw == "" {
		return nil
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	value = value.UTC()
	return &value
}

func coalesceString(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func mapMoneyValues(values []tinkoffMoneyValue) []domain.MoneyValue {
	out := make([]domain.MoneyValue, 0, len(values))
	for _, value := range values {
		out = append(out, value.ToDomain())
	}
	return out
}

func mapSecurityPositions(values []tinkoffSecurityPosition) []domain.BrokerSecurityPosition {
	out := make([]domain.BrokerSecurityPosition, 0, len(values))
	for _, value := range values {
		blocked, _ := parseJSONInt64(value.Blocked)
		balance, _ := parseJSONInt64(value.Balance)
		out = append(out, domain.BrokerSecurityPosition{
			Figi:          value.Figi,
			InstrumentUID: value.InstrumentUID,
			Ticker:        value.Ticker,
			ClassCode:     value.ClassCode,
			Blocked:       blocked,
			Balance:       balance,
		})
	}
	return out
}

func decimalStringToQuotation(raw string) (tinkoffQuotation, error) {
	value := strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
	if value == "" {
		return tinkoffQuotation{}, errors.New("price is required")
	}
	if strings.HasPrefix(value, "-") {
		return tinkoffQuotation{}, errors.New("price must be positive")
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return tinkoffQuotation{}, fmt.Errorf("invalid price: %s", raw)
	}
	units, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return tinkoffQuotation{}, fmt.Errorf("invalid price units: %w", err)
	}
	nano := int64(0)
	if len(parts) == 2 {
		fractional := parts[1]
		if fractional == "" {
			fractional = "0"
		}
		if len(fractional) > 9 {
			return tinkoffQuotation{}, errors.New("price supports up to 9 decimal places")
		}
		for len(fractional) < 9 {
			fractional += "0"
		}
		nano, err = strconv.ParseInt(fractional, 10, 32)
		if err != nil {
			return tinkoffQuotation{}, fmt.Errorf("invalid price nano: %w", err)
		}
	}
	if units == 0 && nano == 0 {
		return tinkoffQuotation{}, errors.New("price must be greater than zero")
	}
	return tinkoffQuotation{Units: units, Nano: int32(nano)}, nil
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
