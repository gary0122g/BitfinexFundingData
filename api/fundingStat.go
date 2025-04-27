package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

// GetFundingStats retrieves funding statistics data for the specified symbol (maintains backward compatibility)
func (c *Client) GetFundingStats(symbol string, limit int) ([]FundingStats, error) {
	// Call the version that supports context, using background context
	return c.GetFundingStatsWithContext(context.Background(), symbol, limit)
}

// GetFundingStatsWithContext retrieves funding statistics data for the specified symbol using context
func (c *Client) GetFundingStatsWithContext(ctx context.Context, symbol string, limit int) ([]FundingStats, error) {
	endpoint := fmt.Sprintf("%s/v2/funding/stats/%s/hist?limit=%d", c.BaseURL, symbol, limit)

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	// Execute request
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

	// Convert raw data to FundingStats
	fundingStats := make([]FundingStats, len(rawData))
	for i, data := range rawData {
		if len(data) >= 12 {
			fundingStats[i] = FundingStats{
				MTS:                   int64(data[0].(float64)),
				FRR:                   data[3].(float64),
				AveragePeriod:         data[4].(float64),
				FundingAmount:         data[7].(float64),
				FundingAmountUsed:     data[8].(float64),
				FundingBelowThreshold: data[11].(float64),
			}
		}
	}

	return fundingStats, nil
}

// GetFundingStatsWithTimeRange retrieves funding statistics data for the specified time range (maintains backward compatibility)
func (c *Client) GetFundingStatsWithTimeRange(symbol string, start, end int64, limit int) ([]FundingStats, error) {
	// Call the version that supports context, using background context
	return c.GetFundingStatsWithTimeRangeWithContext(context.Background(), symbol, start, end, limit)
}

// GetFundingStatsWithTimeRangeWithContext retrieves funding statistics data for the specified time range using context
func (c *Client) GetFundingStatsWithTimeRangeWithContext(ctx context.Context, symbol string, start, end int64, limit int) ([]FundingStats, error) {
	// Build base URL
	baseEndpoint := fmt.Sprintf("%s/v2/funding/stats/%s/hist", c.BaseURL, symbol)

	// Build query parameters
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	if start > 0 {
		query.Set("start", strconv.FormatInt(start, 10))
	}
	if end > 0 {
		query.Set("end", strconv.FormatInt(end, 10))
	}

	// Combine complete URL
	endpoint := baseEndpoint
	if len(query) > 0 {
		endpoint = fmt.Sprintf("%s?%s", baseEndpoint, query.Encode())
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	// Execute request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		var bitfinexError BitfinexError
		bitfinexError.StatusCode = resp.StatusCode
		bitfinexError.Message = string(body)
		return nil, &bitfinexError
	}

	var rawData [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawData); err != nil {
		return nil, err
	}

	// Convert raw data to FundingStats
	fundingStats := make([]FundingStats, 0, len(rawData))
	for _, data := range rawData {
		if len(data) >= 12 {
			stat := FundingStats{
				MTS:                   int64(data[0].(float64)),
				FRR:                   data[3].(float64),
				AveragePeriod:         data[4].(float64),
				FundingAmount:         data[7].(float64),
				FundingAmountUsed:     data[8].(float64),
				FundingBelowThreshold: data[11].(float64),
			}
			fundingStats = append(fundingStats, stat)
		}
	}

	return fundingStats, nil
}
