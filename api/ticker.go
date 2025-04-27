package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetTradingTicker retrieves market data for a trading pair (maintains backward compatibility)
func (c *Client) GetTradingTicker(symbol string) (*TradingTicker, error) {
	return c.GetTradingTickerWithContext(context.Background(), symbol)
}

// GetTradingTickerWithContext retrieves market data for a trading pair using context
func (c *Client) GetTradingTickerWithContext(ctx context.Context, symbol string) (*TradingTicker, error) {
	endpoint := fmt.Sprintf("%s/v2/ticker/%s", c.BaseURL, symbol)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var bitfinexError BitfinexError
		bitfinexError.StatusCode = resp.StatusCode
		return nil, &bitfinexError
	}

	var rawData []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawData); err != nil {
		return nil, err
	}

	// Check if there is enough data
	if len(rawData) < 10 {
		return nil, fmt.Errorf("invalid response format for trading ticker")
	}

	// Convert to TradingTicker
	ticker := &TradingTicker{
		Bid:                 rawData[0].(float64),
		BidSize:             rawData[1].(float64),
		Ask:                 rawData[2].(float64),
		AskSize:             rawData[3].(float64),
		DailyChange:         rawData[4].(float64),
		DailyChangeRelative: rawData[5].(float64),
		LastPrice:           rawData[6].(float64),
		Volume:              rawData[7].(float64),
		High:                rawData[8].(float64),
		Low:                 rawData[9].(float64),
	}

	return ticker, nil
}

// GetFundingTicker retrieves market data for a funding currency (maintains backward compatibility)
func (c *Client) GetFundingTicker(symbol string) (*FundingTicker, error) {
	return c.GetFundingTickerWithContext(context.Background(), symbol)
}

// GetFundingTickerWithContext retrieves market data for a funding currency using context
func (c *Client) GetFundingTickerWithContext(ctx context.Context, symbol string) (*FundingTicker, error) {
	endpoint := fmt.Sprintf("%s/v2/ticker/%s", c.BaseURL, symbol)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var bitfinexError BitfinexError
		bitfinexError.StatusCode = resp.StatusCode
		return nil, &bitfinexError
	}

	var rawData []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawData); err != nil {
		return nil, err
	}

	// Check if there is enough data
	if len(rawData) < 16 {
		return nil, fmt.Errorf("invalid response format for funding ticker")
	}

	// Convert to FundingTicker
	ticker := &FundingTicker{
		FRR:                rawData[0].(float64),
		Bid:                rawData[1].(float64),
		BidPeriod:          int(rawData[2].(float64)),
		BidSize:            rawData[3].(float64),
		Ask:                rawData[4].(float64),
		AskPeriod:          int(rawData[5].(float64)),
		AskSize:            rawData[6].(float64),
		DailyChange:        rawData[7].(float64),
		DailyChangePercent: rawData[8].(float64),
		LastPrice:          rawData[9].(float64),
		Volume:             rawData[10].(float64),
		High:               rawData[11].(float64),
		Low:                rawData[12].(float64),
		FRRAmountAvailable: rawData[15].(float64),
	}

	return ticker, nil
}

// GetTicker is a convenience function that determines the appropriate ticker type based on symbol prefix (maintains backward compatibility)
// t prefix = trading pair (e.g., tBTCUSD)
// f prefix = funding currency (e.g., fUSD)
func (c *Client) GetTicker(symbol string) (interface{}, error) {
	return c.GetTickerWithContext(context.Background(), symbol)
}

// GetTickerWithContext retrieves ticker data using context, determining the appropriate ticker type based on symbol prefix
// t prefix = trading pair (e.g., tBTCUSD)
// f prefix = funding currency (e.g., fUSD)
func (c *Client) GetTickerWithContext(ctx context.Context, symbol string) (interface{}, error) {
	if strings.HasPrefix(symbol, "t") {
		return c.GetTradingTickerWithContext(ctx, symbol)
	} else if strings.HasPrefix(symbol, "f") {
		return c.GetFundingTickerWithContext(ctx, symbol)
	}

	return nil, fmt.Errorf("invalid symbol format: %s, must start with 't' for trading or 'f' for funding", symbol)
}
