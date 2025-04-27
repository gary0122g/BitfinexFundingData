package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// BookPrecision represents the precision level for order book data
type BookPrecision string

const (
	// Precision for trading books
	PrecisionP0  BookPrecision = "P0" // Highest level of aggregation (least precise)
	PrecisionP1  BookPrecision = "P1"
	PrecisionP2  BookPrecision = "P2"
	PrecisionP3  BookPrecision = "P3"
	PrecisionP4  BookPrecision = "P4" // Lowest level of aggregation (most precise)
	PrecisionRaw BookPrecision = "R0" // Raw, non-aggregated order books
)

// / GetRawFundingBookWithContext
func (c *Client) GetRawFundingBookWithContext(ctx context.Context, symbol string) ([]RawFundingBook, error) {
	endpoint := fmt.Sprintf("%s/v2/book/%s/R0", c.BaseURL, symbol)
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

	var rawData [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawData); err != nil {
		return nil, err
	}

	// Convert raw data to RawFundingBook
	rawFundingBook := make([]RawFundingBook, len(rawData))
	for i, data := range rawData {
		if len(data) >= 4 {
			rawFundingBook[i] = RawFundingBook{
				OfferID: int(data[0].(float64)),
				Period:  int(data[1].(float64)),
				Rate:    data[2].(float64),
				Amount:  data[3].(float64),
			}
		}
	}

	return rawFundingBook, nil
}

// GetFundingBookWithContext 使用上下文獲取資金訂單簿
func (c *Client) GetFundingBookWithContext(ctx context.Context, symbol string, precision BookPrecision) ([]FundingBook, error) {
	endpoint := fmt.Sprintf("%s/v2/book/%s/%s", c.BaseURL, symbol, precision)
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

	var rawData [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawData); err != nil {
		return nil, err
	}

	// Convert raw data to FundingBook
	fundingBook := make([]FundingBook, len(rawData))
	for i, data := range rawData {
		if len(data) >= 4 {
			fundingBook[i] = FundingBook{
				Rate:   data[0].(float64),
				Period: int(data[1].(float64)),
				Count:  int(data[2].(float64)),
				Amount: data[3].(float64),
			}
		}
	}

	return fundingBook, nil
}
