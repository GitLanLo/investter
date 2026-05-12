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
