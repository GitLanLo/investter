package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"invest/backend/internal/app"
	"invest/backend/internal/domain"
	"invest/backend/internal/service"
)

type brokerConnectionRequest struct {
	Name      string `json:"name"`
	Token     string `json:"token"`
	IsSandbox bool   `json:"is_sandbox"`
}

type brokerAccountSelectionRequest struct {
	ConnectionID int64  `json:"connection_id"`
	AccountID    string `json:"account_id"`
}

type brokerPlaceOrderRequest struct {
	ConnectionID int64  `json:"connection_id"`
	AccountID    string `json:"account_id"`
	InstrumentID string `json:"instrument_id"`
	Quantity     int64  `json:"quantity"`
	Price        string `json:"price"`
	Direction    string `json:"direction"`
	OrderType    string `json:"order_type"`
	OrderID      string `json:"order_id"`
}

type brokerSandboxAccountRequest struct {
	ConnectionID   int64  `json:"connection_id"`
	InitialBalance string `json:"initial_balance"`
}

type brokerSandboxPayInRequest struct {
	ConnectionID int64  `json:"connection_id"`
	AccountID    string `json:"account_id"`
	Amount       string `json:"amount"`
}

type brokerConnectionDTO struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	TokenHint  string `json:"token_hint"`
	IsSandbox  bool   `json:"is_sandbox"`
	IsActive   bool   `json:"is_active"`
	LastSyncAt string `json:"last_sync_at,omitempty"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type brokerAccountDTO struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	OpenedDate  string `json:"opened_date,omitempty"`
	ClosedDate  string `json:"closed_date,omitempty"`
	AccessLevel string `json:"access_level"`
}

type brokerConnectionsResponse struct {
	Items []brokerConnectionDTO `json:"items"`
}

type brokerAccountsResponse struct {
	Connection brokerConnectionDTO `json:"connection"`
	Items      []brokerAccountDTO  `json:"items"`
}

type brokerContextResponse struct {
	ConnectionID int64                `json:"connection_id"`
	AccountID    string               `json:"account_id"`
	Connection   *brokerConnectionDTO `json:"connection,omitempty"`
}

type brokerPortfolioResponse struct {
	Connection brokerConnectionDTO    `json:"connection"`
	AccountID  string                 `json:"account_id"`
	Portfolio  domain.BrokerPortfolio `json:"portfolio"`
}

type brokerPositionsResponse struct {
	Connection brokerConnectionDTO    `json:"connection"`
	AccountID  string                 `json:"account_id"`
	Positions  domain.BrokerPositions `json:"positions"`
}

type brokerOperationsResponse struct {
	Connection brokerConnectionDTO         `json:"connection"`
	AccountID  string                      `json:"account_id"`
	Page       domain.BrokerOperationsPage `json:"page"`
}

type brokerOrdersResponse struct {
	Connection brokerConnectionDTO  `json:"connection"`
	AccountID  string               `json:"account_id"`
	Items      []domain.BrokerOrder `json:"items"`
}

type brokerOrderResponse struct {
	Connection brokerConnectionDTO `json:"connection"`
	AccountID  string              `json:"account_id"`
	Order      domain.BrokerOrder  `json:"order"`
}

type brokerCancelOrderResponse struct {
	Connection brokerConnectionDTO            `json:"connection"`
	AccountID  string                         `json:"account_id"`
	Result     domain.BrokerCancelOrderResult `json:"result"`
}

type brokerSandboxAccountResponse struct {
	Connection brokerConnectionDTO              `json:"connection"`
	AccountID  string                           `json:"account_id"`
	PayIn      *domain.BrokerSandboxPayInResult `json:"pay_in,omitempty"`
}

type brokerSandboxPayInResponse struct {
	Connection brokerConnectionDTO             `json:"connection"`
	AccountID  string                          `json:"account_id"`
	PayIn      domain.BrokerSandboxPayInResult `json:"pay_in"`
}

func ListBrokerConnections(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !brokerServiceAvailable(w, container) {
			return
		}
		userID := r.Context().Value("user_id").(int64)
		items, err := container.Services.TinkoffCredentials.ListBrokerConnections(r.Context(), userID)
		if err != nil {
			writeBrokerError(w, err, "broker_connections_list_failed")
			return
		}
		out := brokerConnectionsResponse{Items: make([]brokerConnectionDTO, 0, len(items))}
		for _, item := range items {
			out.Items = append(out.Items, toBrokerConnectionDTO(item))
		}
		WriteJSON(w, http.StatusOK, out)
	}
}

func CreateBrokerConnection(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !brokerServiceAvailable(w, container) {
			return
		}
		userID := r.Context().Value("user_id").(int64)
		var req brokerConnectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body", nil)
			return
		}
		req.Token = strings.TrimSpace(req.Token)
		if req.Token == "" {
			WriteError(w, http.StatusBadRequest, "validation_error", "token is required", nil)
			return
		}
		if err := container.Services.TinkoffCredentials.TestToken(r.Context(), req.Token, req.IsSandbox); err != nil {
			WriteError(w, http.StatusUnauthorized, "broker_token_invalid", err.Error(), nil)
			return
		}
		connection, err := container.Services.TinkoffCredentials.CreateBrokerConnection(r.Context(), userID, req.Name, req.Token, req.IsSandbox)
		if err != nil {
			writeBrokerError(w, err, "broker_connection_create_failed")
			return
		}
		WriteJSON(w, http.StatusCreated, toBrokerConnectionDTO(connection))
	}
}

func SetActiveBrokerConnection(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !brokerServiceAvailable(w, container) {
			return
		}
		userID := r.Context().Value("user_id").(int64)
		connectionID, ok := brokerPathID(w, r)
		if !ok {
			return
		}
		if err := container.Services.TinkoffCredentials.SetActiveBrokerConnection(r.Context(), userID, connectionID); err != nil {
			writeBrokerError(w, err, "broker_connection_activate_failed")
			return
		}
		WriteJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}
}

func DeleteBrokerConnection(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !brokerServiceAvailable(w, container) {
			return
		}
		userID := r.Context().Value("user_id").(int64)
		connectionID, ok := brokerPathID(w, r)
		if !ok {
			return
		}
		if err := container.Services.TinkoffCredentials.DeleteBrokerConnection(r.Context(), userID, connectionID); err != nil {
			writeBrokerError(w, err, "broker_connection_delete_failed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func ListBrokerAccounts(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !brokerServiceAvailable(w, container) {
			return
		}
		userID := r.Context().Value("user_id").(int64)
		connectionID := brokerQueryID(r, "connection_id")
		accounts, connection, err := container.Services.TinkoffCredentials.ListBrokerAccounts(r.Context(), userID, connectionID)
		if err != nil {
			writeBrokerError(w, err, "broker_accounts_list_failed")
			return
		}
		out := brokerAccountsResponse{
			Connection: toBrokerConnectionDTO(connection),
			Items:      make([]brokerAccountDTO, 0, len(accounts)),
		}
		for _, account := range accounts {
			out.Items = append(out.Items, toBrokerAccountDTO(account))
		}
		WriteJSON(w, http.StatusOK, out)
	}
}

func GetBrokerContext(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !brokerServiceAvailable(w, container) {
			return
		}
		userID := r.Context().Value("user_id").(int64)
		connectionID := brokerQueryID(r, "connection_id")
		accountID := strings.TrimSpace(r.URL.Query().Get("account_id"))
		connection, resolvedAccountID, err := container.Services.TinkoffCredentials.GetBrokerContext(r.Context(), userID, connectionID, accountID)
		if err != nil {
			writeBrokerError(w, err, "broker_context_failed")
			return
		}
		dto := toBrokerConnectionDTO(connection)
		WriteJSON(w, http.StatusOK, brokerContextResponse{
			ConnectionID: connection.ID,
			AccountID:    resolvedAccountID,
			Connection:   &dto,
		})
	}
}

func SaveBrokerContext(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !brokerServiceAvailable(w, container) {
			return
		}
		userID := r.Context().Value("user_id").(int64)
		var req brokerAccountSelectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body", nil)
			return
		}
		req.AccountID = strings.TrimSpace(req.AccountID)
		if req.AccountID == "" {
			WriteError(w, http.StatusBadRequest, "validation_error", "account_id is required", nil)
			return
		}
		if err := container.Services.TinkoffCredentials.SaveBrokerAccountSelection(r.Context(), userID, req.ConnectionID, req.AccountID); err != nil {
			writeBrokerError(w, err, "broker_context_save_failed")
			return
		}
		WriteJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}
}

func GetBrokerPortfolio(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !brokerServiceAvailable(w, container) {
			return
		}
		userID := r.Context().Value("user_id").(int64)
		connectionID := brokerQueryID(r, "connection_id")
		accountID := strings.TrimSpace(r.URL.Query().Get("account_id"))
		connection, portfolio, err := container.Services.TinkoffCredentials.GetPortfolio(r.Context(), userID, connectionID, accountID)
		if err != nil {
			writeBrokerError(w, err, "broker_portfolio_failed")
			return
		}
		WriteJSON(w, http.StatusOK, brokerPortfolioResponse{
			Connection: toBrokerConnectionDTO(connection),
			AccountID:  portfolio.AccountID,
			Portfolio:  portfolio,
		})
	}
}

func GetBrokerPositions(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !brokerServiceAvailable(w, container) {
			return
		}
		userID := r.Context().Value("user_id").(int64)
		connectionID := brokerQueryID(r, "connection_id")
		accountID := strings.TrimSpace(r.URL.Query().Get("account_id"))
		connection, positions, err := container.Services.TinkoffCredentials.GetPositions(r.Context(), userID, connectionID, accountID)
		if err != nil {
			writeBrokerError(w, err, "broker_positions_failed")
			return
		}
		WriteJSON(w, http.StatusOK, brokerPositionsResponse{
			Connection: toBrokerConnectionDTO(connection),
			AccountID:  positions.AccountID,
			Positions:  positions,
		})
	}
}

func GetBrokerOperations(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !brokerServiceAvailable(w, container) {
			return
		}
		userID := r.Context().Value("user_id").(int64)
		connectionID := brokerQueryID(r, "connection_id")
		request, ok := brokerOperationsRequestFromQuery(w, r)
		if !ok {
			return
		}
		connection, page, err := container.Services.TinkoffCredentials.GetOperations(r.Context(), userID, connectionID, request)
		if err != nil {
			writeBrokerError(w, err, "broker_operations_failed")
			return
		}
		WriteJSON(w, http.StatusOK, brokerOperationsResponse{
			Connection: toBrokerConnectionDTO(connection),
			AccountID:  request.AccountID,
			Page:       page,
		})
	}
}

func GetBrokerOrders(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !brokerServiceAvailable(w, container) {
			return
		}
		userID := r.Context().Value("user_id").(int64)
		connectionID := brokerQueryID(r, "connection_id")
		accountID := strings.TrimSpace(r.URL.Query().Get("account_id"))
		connection, orders, err := container.Services.TinkoffCredentials.GetOrders(r.Context(), userID, connectionID, accountID)
		if err != nil {
			writeBrokerError(w, err, "broker_orders_failed")
			return
		}
		WriteJSON(w, http.StatusOK, brokerOrdersResponse{
			Connection: toBrokerConnectionDTO(connection),
			AccountID:  accountID,
			Items:      orders,
		})
	}
}

func PostBrokerOrder(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !brokerServiceAvailable(w, container) {
			return
		}
		userID := r.Context().Value("user_id").(int64)
		req, ok := brokerPlaceOrderRequestFromBody(w, r)
		if !ok {
			return
		}
		connection, order, err := container.Services.TinkoffCredentials.PostOrder(r.Context(), userID, req.ConnectionID, domain.BrokerPlaceOrderRequest{
			AccountID:    req.AccountID,
			InstrumentID: req.InstrumentID,
			Quantity:     req.Quantity,
			PriceDecimal: req.Price,
			Direction:    req.Direction,
			OrderType:    req.OrderType,
			OrderID:      req.OrderID,
		})
		if err != nil {
			writeBrokerError(w, err, "broker_order_post_failed")
			return
		}
		WriteJSON(w, http.StatusCreated, brokerOrderResponse{
			Connection: toBrokerConnectionDTO(connection),
			AccountID:  req.AccountID,
			Order:      order,
		})
	}
}

func GetBrokerOrderState(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !brokerServiceAvailable(w, container) {
			return
		}
		userID := r.Context().Value("user_id").(int64)
		connectionID := brokerQueryID(r, "connection_id")
		accountID := strings.TrimSpace(r.URL.Query().Get("account_id"))
		orderID := strings.TrimSpace(r.PathValue("order_id"))
		if orderID == "" {
			WriteError(w, http.StatusBadRequest, "validation_error", "order_id is required", nil)
			return
		}
		connection, order, err := container.Services.TinkoffCredentials.GetOrderState(r.Context(), userID, connectionID, accountID, orderID)
		if err != nil {
			writeBrokerError(w, err, "broker_order_state_failed")
			return
		}
		WriteJSON(w, http.StatusOK, brokerOrderResponse{
			Connection: toBrokerConnectionDTO(connection),
			AccountID:  accountID,
			Order:      order,
		})
	}
}

func CancelBrokerOrder(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !brokerServiceAvailable(w, container) {
			return
		}
		userID := r.Context().Value("user_id").(int64)
		connectionID := brokerQueryID(r, "connection_id")
		accountID := strings.TrimSpace(r.URL.Query().Get("account_id"))
		orderID := strings.TrimSpace(r.PathValue("order_id"))
		if orderID == "" {
			WriteError(w, http.StatusBadRequest, "validation_error", "order_id is required", nil)
			return
		}
		connection, result, err := container.Services.TinkoffCredentials.CancelOrder(r.Context(), userID, connectionID, accountID, orderID)
		if err != nil {
			writeBrokerError(w, err, "broker_order_cancel_failed")
			return
		}
		WriteJSON(w, http.StatusOK, brokerCancelOrderResponse{
			Connection: toBrokerConnectionDTO(connection),
			AccountID:  result.AccountID,
			Result:     result,
		})
	}
}

func OpenBrokerSandboxAccount(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !brokerServiceAvailable(w, container) {
			return
		}
		userID := r.Context().Value("user_id").(int64)
		var req brokerSandboxAccountRequest
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&req)
		}
		connection, accountID, err := container.Services.TinkoffCredentials.OpenSandboxAccount(r.Context(), userID, req.ConnectionID)
		if err != nil {
			writeBrokerError(w, err, "broker_sandbox_account_create_failed")
			return
		}
		response := brokerSandboxAccountResponse{
			Connection: toBrokerConnectionDTO(connection),
			AccountID:  accountID,
		}
		initialBalance := strings.TrimSpace(req.InitialBalance)
		if initialBalance != "" && initialBalance != "0" {
			amount, ok := brokerMoneyValueFromString(w, initialBalance, "rub")
			if !ok {
				return
			}
			_, payIn, err := container.Services.TinkoffCredentials.SandboxPayIn(r.Context(), userID, connection.ID, accountID, amount)
			if err != nil {
				writeBrokerError(w, err, "broker_sandbox_pay_in_failed")
				return
			}
			response.PayIn = &payIn
		}
		WriteJSON(w, http.StatusCreated, response)
	}
}

func BrokerSandboxPayIn(container app.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !brokerServiceAvailable(w, container) {
			return
		}
		userID := r.Context().Value("user_id").(int64)
		var req brokerSandboxPayInRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body", nil)
			return
		}
		amount, ok := brokerMoneyValueFromString(w, req.Amount, "rub")
		if !ok {
			return
		}
		connection, payIn, err := container.Services.TinkoffCredentials.SandboxPayIn(r.Context(), userID, req.ConnectionID, strings.TrimSpace(req.AccountID), amount)
		if err != nil {
			writeBrokerError(w, err, "broker_sandbox_pay_in_failed")
			return
		}
		WriteJSON(w, http.StatusOK, brokerSandboxPayInResponse{
			Connection: toBrokerConnectionDTO(connection),
			AccountID:  payIn.AccountID,
			PayIn:      payIn,
		})
	}
}

func brokerServiceAvailable(w http.ResponseWriter, container app.Container) bool {
	if container.Services.TinkoffCredentials == nil {
		WriteError(w, http.StatusServiceUnavailable, "broker_unavailable", "broker service is not configured", nil)
		return false
	}
	return true
}

func brokerPathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		WriteError(w, http.StatusBadRequest, "invalid_connection_id", "connection id must be a positive integer", nil)
		return 0, false
	}
	return id, true
}

func brokerQueryID(r *http.Request, key string) int64 {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return 0
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

func brokerOperationsRequestFromQuery(w http.ResponseWriter, r *http.Request) (domain.BrokerOperationsRequest, bool) {
	now := time.Now().UTC()
	req := domain.BrokerOperationsRequest{
		AccountID: strings.TrimSpace(r.URL.Query().Get("account_id")),
		From:      now.AddDate(0, -1, 0),
		To:        now,
		Cursor:    strings.TrimSpace(r.URL.Query().Get("cursor")),
		Limit:     50,
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit <= 0 || limit > 1000 {
			WriteError(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 1000", nil)
			return domain.BrokerOperationsRequest{}, false
		}
		req.Limit = limit
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("from")); raw != "" {
		value, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_from", "from must be RFC3339 timestamp", nil)
			return domain.BrokerOperationsRequest{}, false
		}
		req.From = value.UTC()
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("to")); raw != "" {
		value, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_to", "to must be RFC3339 timestamp", nil)
			return domain.BrokerOperationsRequest{}, false
		}
		req.To = value.UTC()
	}
	return req, true
}

func brokerPlaceOrderRequestFromBody(w http.ResponseWriter, r *http.Request) (brokerPlaceOrderRequest, bool) {
	var req brokerPlaceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body", nil)
		return brokerPlaceOrderRequest{}, false
	}
	req.AccountID = strings.TrimSpace(req.AccountID)
	req.InstrumentID = strings.TrimSpace(req.InstrumentID)
	req.Price = strings.TrimSpace(req.Price)
	req.Direction = normalizeOrderDirection(req.Direction)
	req.OrderType = normalizeOrderType(req.OrderType)
	req.OrderID = strings.TrimSpace(req.OrderID)
	if req.InstrumentID == "" {
		WriteError(w, http.StatusBadRequest, "validation_error", "instrument_id is required", nil)
		return brokerPlaceOrderRequest{}, false
	}
	if req.Quantity <= 0 {
		WriteError(w, http.StatusBadRequest, "validation_error", "quantity must be positive lot count", nil)
		return brokerPlaceOrderRequest{}, false
	}
	if req.Direction == "" {
		WriteError(w, http.StatusBadRequest, "validation_error", "direction must be buy or sell", nil)
		return brokerPlaceOrderRequest{}, false
	}
	if req.OrderType == "" {
		WriteError(w, http.StatusBadRequest, "validation_error", "order_type must be limit or market", nil)
		return brokerPlaceOrderRequest{}, false
	}
	if req.OrderType == "ORDER_TYPE_LIMIT" && req.Price == "" {
		WriteError(w, http.StatusBadRequest, "validation_error", "price is required for limit orders", nil)
		return brokerPlaceOrderRequest{}, false
	}
	if len(req.OrderID) > 36 {
		WriteError(w, http.StatusBadRequest, "validation_error", "order_id must be 36 characters or less", nil)
		return brokerPlaceOrderRequest{}, false
	}
	return req, true
}

func normalizeOrderDirection(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "BUY", "ORDER_DIRECTION_BUY":
		return "ORDER_DIRECTION_BUY"
	case "SELL", "ORDER_DIRECTION_SELL":
		return "ORDER_DIRECTION_SELL"
	default:
		return ""
	}
}

func normalizeOrderType(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "LIMIT", "ORDER_TYPE_LIMIT":
		return "ORDER_TYPE_LIMIT"
	case "MARKET", "ORDER_TYPE_MARKET":
		return "ORDER_TYPE_MARKET"
	default:
		return ""
	}
}

func brokerMoneyValueFromString(w http.ResponseWriter, raw string, currency string) (domain.MoneyValue, bool) {
	value := strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
	if value == "" {
		WriteError(w, http.StatusBadRequest, "validation_error", "amount is required", nil)
		return domain.MoneyValue{}, false
	}
	if strings.HasPrefix(value, "-") {
		WriteError(w, http.StatusBadRequest, "validation_error", "amount must be positive", nil)
		return domain.MoneyValue{}, false
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		WriteError(w, http.StatusBadRequest, "validation_error", "amount must be decimal number", nil)
		return domain.MoneyValue{}, false
	}
	units, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "validation_error", "amount units must be integer", nil)
		return domain.MoneyValue{}, false
	}
	nano := int64(0)
	if len(parts) == 2 {
		fractional := parts[1]
		if len(fractional) > 9 {
			WriteError(w, http.StatusBadRequest, "validation_error", "amount supports up to 9 decimal places", nil)
			return domain.MoneyValue{}, false
		}
		for len(fractional) < 9 {
			fractional += "0"
		}
		nano, err = strconv.ParseInt(fractional, 10, 32)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "validation_error", "amount fraction must be numeric", nil)
			return domain.MoneyValue{}, false
		}
	}
	if units == 0 && nano == 0 {
		WriteError(w, http.StatusBadRequest, "validation_error", "amount must be greater than zero", nil)
		return domain.MoneyValue{}, false
	}
	return domain.MoneyValue{
		Currency: currency,
		Units:    units,
		Nano:     int32(nano),
		Amount:   float64(units) + float64(nano)/1e9,
	}, true
}

func writeBrokerError(w http.ResponseWriter, err error, code string) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		WriteError(w, http.StatusNotFound, "broker_not_found", "broker connection or account was not found", nil)
	case errors.Is(err, service.ErrTinkoffUnavailable):
		WriteError(w, http.StatusServiceUnavailable, "broker_unavailable", err.Error(), nil)
	default:
		WriteError(w, http.StatusInternalServerError, code, err.Error(), nil)
	}
}

func toBrokerConnectionDTO(item domain.BrokerConnection) brokerConnectionDTO {
	out := brokerConnectionDTO{
		ID:        item.ID,
		Name:      item.Name,
		TokenHint: item.TokenHint,
		IsSandbox: item.IsSandbox,
		IsActive:  item.IsActive,
		CreatedAt: formatOptionalTime(item.CreatedAt),
		UpdatedAt: formatOptionalTime(item.UpdatedAt),
	}
	if item.LastSyncAt != nil {
		out.LastSyncAt = item.LastSyncAt.UTC().Format(time.RFC3339)
	}
	return out
}

func toBrokerAccountDTO(item domain.BrokerAccount) brokerAccountDTO {
	out := brokerAccountDTO{
		ID:          item.ID,
		Type:        item.Type,
		Name:        item.Name,
		Status:      item.Status,
		AccessLevel: item.AccessLevel,
	}
	if item.OpenedDate != nil {
		out.OpenedDate = item.OpenedDate.UTC().Format(time.RFC3339)
	}
	if item.ClosedDate != nil {
		out.ClosedDate = item.ClosedDate.UTC().Format(time.RFC3339)
	}
	return out
}
