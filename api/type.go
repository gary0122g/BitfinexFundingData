package api

import "net/http"

type Client struct {
	APIKey     string
	APISecret  string
	HTTPClient *http.Client
	BaseURL    string
}

type BitfinexError struct {
	StatusCode int
	ErrorCode  string
	Message    string
	RawBody    string
}

// FundingStats represents funding statistics for a given currency
type FundingStats struct {
	MTS                   int64   `json:"mts"`
	FRR                   float64 `json:"frr"`
	AveragePeriod         float64 `json:"avg_period"`
	FundingAmount         float64 `json:"funding_amount"`
	FundingAmountUsed     float64 `json:"funding_amount_used"`
	FundingBelowThreshold float64 `json:"funding_below_threshold"`
}

// TradingBook represents a price aggregated order book entry for trading pairs
type TradingBook struct {
	Price  float64 `json:"price"`
	Count  int     `json:"count"`
	Amount float64 `json:"amount"` // > 0 for bids, < 0 for asks
}

// FundingBook represents a rate aggregated order book entry for funding currencies
type FundingBook struct {
	Rate   float64 `json:"rate"`
	Period int     `json:"period"`
	Count  int     `json:"count"`
	Amount float64 `json:"amount"` // > 0 for asks, < 0 for bids
}

// RawTradingBook represents a raw order book entry for trading pairs
type RawTradingBook struct {
	OrderID int     `json:"order_id"`
	Price   float64 `json:"price"`
	Amount  float64 `json:"amount"` // > 0 for bids, < 0 for asks
}

// RawFundingBook represents a raw order book entry for funding currencies
type RawFundingBook struct {
	OfferID int     `json:"offer_id"`
	Period  int     `json:"period"`
	Rate    float64 `json:"rate"`
	Amount  float64 `json:"amount"` // > 0 for asks, < 0 for bids
}

// TradingTicker represents the ticker data for a trading pair
type TradingTicker struct {
	Bid                 float64 `json:"bid"`                   // Price of last highest bid
	BidSize             float64 `json:"bid_size"`              // Sum of the 25 highest bid sizes
	Ask                 float64 `json:"ask"`                   // Price of last lowest ask
	AskSize             float64 `json:"ask_size"`              // Sum of the 25 lowest ask sizes
	DailyChange         float64 `json:"daily_change"`          // Amount that the last price has changed since yesterday
	DailyChangeRelative float64 `json:"daily_change_relative"` // Relative price change since yesterday (*100 for percentage change)
	LastPrice           float64 `json:"last_price"`            // Price of the last trade
	Volume              float64 `json:"volume"`                // Daily volume
	High                float64 `json:"high"`                  // Daily high
	Low                 float64 `json:"low"`                   // Daily low
}

// FundingTicker represents the ticker data for a funding currency
type FundingTicker struct {
	FRR                float64 `json:"frr"`                  // Flash Return Rate - average of all fixed rate funding over the last hour
	Bid                float64 `json:"bid"`                  // Price of last highest bid
	BidPeriod          int     `json:"bid_period"`           // Bid period covered in days
	BidSize            float64 `json:"bid_size"`             // Sum of the 25 highest bid sizes
	Ask                float64 `json:"ask"`                  // Price of last lowest ask
	AskPeriod          int     `json:"ask_period"`           // Ask period covered in days
	AskSize            float64 `json:"ask_size"`             // Sum of the 25 lowest ask sizes
	DailyChange        float64 `json:"daily_change"`         // Amount that the last price has changed since yesterday
	DailyChangePercent float64 `json:"daily_change_perc"`    // Relative price change since yesterday (*100 for percentage change)
	LastPrice          float64 `json:"last_price"`           // Price of the last trade
	Volume             float64 `json:"volume"`               // Daily volume
	High               float64 `json:"high"`                 // Daily high
	Low                float64 `json:"low"`                  // Daily low
	FRRAmountAvailable float64 `json:"frr_amount_available"` // The amount of funding that is available at the Flash Return Rate
}
