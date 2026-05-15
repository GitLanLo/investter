package domain

import "time"

type TinkoffInstrument struct {
	UID                 string
	Figi                string
	Ticker              string
	ClassCode           string
	Isin                string
	Lot                 int32
	Currency            string
	Name                string
	Exchange            string
	InstrumentType      string
	APITradeAvailable   bool
	First1MinCandleDate *time.Time
	First1DayCandleDate *time.Time
}

type UserTinkoffCredential struct {
	UserID         int64
	TokenEncrypted []byte
	TokenNonce     []byte
	TokenHint      string
	IsSandbox      bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type BrokerConnection struct {
	ID             int64
	UserID         int64
	Name           string
	TokenEncrypted []byte
	TokenNonce     []byte
	TokenHint      string
	IsSandbox      bool
	IsActive       bool
	LastSyncAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type BrokerAccountSelection struct {
	UserID       int64
	ConnectionID int64
	AccountID    string
	UpdatedAt    time.Time
}

type BrokerAccount struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	OpenedDate  *time.Time `json:"opened_date,omitempty"`
	ClosedDate  *time.Time `json:"closed_date,omitempty"`
	AccessLevel string     `json:"access_level"`
}

type MoneyValue struct {
	Currency string  `json:"currency"`
	Units    int64   `json:"units"`
	Nano     int32   `json:"nano"`
	Amount   float64 `json:"amount"`
}

type BrokerPortfolio struct {
	AccountID             string                    `json:"account_id"`
	TotalAmountPortfolio  MoneyValue                `json:"total_amount_portfolio"`
	TotalAmountShares     MoneyValue                `json:"total_amount_shares"`
	TotalAmountBonds      MoneyValue                `json:"total_amount_bonds"`
	TotalAmountETF        MoneyValue                `json:"total_amount_etf"`
	TotalAmountCurrencies MoneyValue                `json:"total_amount_currencies"`
	TotalAmountFutures    MoneyValue                `json:"total_amount_futures"`
	TotalAmountOptions    MoneyValue                `json:"total_amount_options"`
	ExpectedYield         float64                   `json:"expected_yield"`
	DailyYield            MoneyValue                `json:"daily_yield"`
	DailyYieldRelative    float64                   `json:"daily_yield_relative"`
	Positions             []BrokerPortfolioPosition `json:"positions"`
}

type BrokerPortfolioPosition struct {
	Figi                 string     `json:"figi"`
	InstrumentType       string     `json:"instrument_type"`
	Quantity             float64    `json:"quantity"`
	AveragePositionPrice MoneyValue `json:"average_position_price"`
	ExpectedYield        float64    `json:"expected_yield"`
	CurrentPrice         MoneyValue `json:"current_price"`
	Blocked              bool       `json:"blocked"`
	BlockedLots          float64    `json:"blocked_lots"`
	PositionUID          string     `json:"position_uid"`
	InstrumentUID        string     `json:"instrument_uid"`
	Ticker               string     `json:"ticker"`
	ClassCode            string     `json:"class_code"`
	DailyYield           MoneyValue `json:"daily_yield"`
}

type BrokerPositions struct {
	AccountID               string                   `json:"account_id"`
	Money                   []MoneyValue             `json:"money"`
	Blocked                 []MoneyValue             `json:"blocked"`
	Securities              []BrokerSecurityPosition `json:"securities"`
	Futures                 []BrokerSecurityPosition `json:"futures"`
	Options                 []BrokerSecurityPosition `json:"options"`
	LimitsLoadingInProgress bool                     `json:"limits_loading_in_progress"`
}

type BrokerSecurityPosition struct {
	Figi          string `json:"figi"`
	InstrumentUID string `json:"instrument_uid"`
	Ticker        string `json:"ticker"`
	ClassCode     string `json:"class_code"`
	Blocked       int64  `json:"blocked"`
	Balance       int64  `json:"balance"`
}

type BrokerOperationsRequest struct {
	AccountID string
	From      time.Time
	To        time.Time
	Cursor    string
	Limit     int
}

type BrokerOperationsPage struct {
	HasNext    bool              `json:"has_next"`
	NextCursor string            `json:"next_cursor"`
	Items      []BrokerOperation `json:"items"`
}

type BrokerOperation struct {
	Cursor          string     `json:"cursor"`
	BrokerAccountID string     `json:"broker_account_id"`
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Date            *time.Time `json:"date,omitempty"`
	Type            string     `json:"type"`
	Description     string     `json:"description"`
	State           string     `json:"state"`
	InstrumentUID   string     `json:"instrument_uid"`
	QuantityDone    int64      `json:"quantity_done"`
}

type BrokerOrder struct {
	OrderID               string     `json:"order_id"`
	OrderRequestID        string     `json:"order_request_id,omitempty"`
	ExecutionReportStatus string     `json:"execution_report_status"`
	LotsRequested         int64      `json:"lots_requested"`
	LotsExecuted          int64      `json:"lots_executed"`
	LotsLeft              int64      `json:"lots_left"`
	InitialOrderPrice     MoneyValue `json:"initial_order_price"`
	ExecutedOrderPrice    MoneyValue `json:"executed_order_price"`
	InitialSecurityPrice  MoneyValue `json:"initial_security_price"`
	Direction             string     `json:"direction"`
	OrderType             string     `json:"order_type"`
	InstrumentUID         string     `json:"instrument_uid"`
	Figi                  string     `json:"figi"`
	CreatedAt             *time.Time `json:"created_at,omitempty"`
	Message               string     `json:"message,omitempty"`
}

type BrokerPlaceOrderRequest struct {
	AccountID    string
	InstrumentID string
	Quantity     int64
	PriceDecimal string
	Direction    string
	OrderType    string
	OrderID      string
}

type BrokerCancelOrderResult struct {
	AccountID string     `json:"account_id"`
	OrderID   string     `json:"order_id"`
	Time      *time.Time `json:"time,omitempty"`
}

type BrokerSandboxPayInResult struct {
	AccountID string     `json:"account_id"`
	Balance   MoneyValue `json:"balance"`
}
