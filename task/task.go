package task

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/gary0122g/BitfinexFundingData/api"
	"github.com/gary0122g/BitfinexFundingData/db"
	"github.com/gary0122g/BitfinexFundingData/scheduler"
)

type RawFundingBookResult struct {
	Data  []api.RawFundingBook
	Error error
}

type FundingBookResult struct {
	Data  []api.FundingBook
	Error error
}

type FundingStatsResult struct {
	Data  []api.FundingStats
	Error error
}

type FundingTickerResult struct {
	Data  *api.FundingTicker
	Error error
}

type GetRawFundingBookTask struct {
	scheduler.BaseTask
	Client     *api.Client
	Symbol     string
	ResultChan chan<- RawFundingBookResult
	Storage    db.Storage
}

func NewGetRawFundingBookTask(client *api.Client, symbol string, resultChan chan<- RawFundingBookResult, priority int) *GetRawFundingBookTask {
	return &GetRawFundingBookTask{
		BaseTask: scheduler.BaseTask{
			Name:     fmt.Sprintf("GetRawFundingBook_%s", symbol),
			Priority: priority,
			RetryPolicy: scheduler.RetryPolicy{
				MaxRetries:  3,
				BackoffBase: 500 * time.Millisecond,
			},
		},
		Client:     client,
		Symbol:     symbol,
		ResultChan: resultChan,
	}
}

func (t *GetRawFundingBookTask) Execute(ctx context.Context) error {
	// Create cancelable request using context
	result, err := t.Client.GetRawFundingBookWithContext(ctx, t.Symbol)

	// Send result to channel
	t.ResultChan <- RawFundingBookResult{
		Data:  result,
		Error: err,
	}

	return err
}

type GetFundingBookTask struct {
	scheduler.BaseTask
	Client     *api.Client
	Symbol     string
	Precision  api.BookPrecision
	ResultChan chan<- FundingBookResult
	Storage    db.Storage
}

func NewGetFundingBookTask(client *api.Client, symbol string, precision api.BookPrecision, resultChan chan<- FundingBookResult, priority int) *GetFundingBookTask {
	return &GetFundingBookTask{
		BaseTask: scheduler.BaseTask{
			Name:     fmt.Sprintf("GetFundingBook_%s_%s", symbol, precision),
			Priority: priority,
			RetryPolicy: scheduler.RetryPolicy{
				MaxRetries:  3,
				BackoffBase: 500 * time.Millisecond,
			},
		},
		Client:     client,
		Symbol:     symbol,
		Precision:  precision,
		ResultChan: resultChan,
	}
}

func (t *GetFundingBookTask) Execute(ctx context.Context) error {
	result, err := t.Client.GetFundingBookWithContext(ctx, t.Symbol, t.Precision)

	t.ResultChan <- FundingBookResult{
		Data:  result,
		Error: err,
	}

	return err
}

// 3. Funding Stats Task
type GetFundingStatsTask struct {
	scheduler.BaseTask
	Client     *api.Client
	Symbol     string
	Start      int64 // Added: start timestamp
	End        int64 // Added: end timestamp
	Limit      int
	ResultChan chan<- FundingStatsResult
	Storage    db.Storage // Optional
}

// Original function to create funding stats task
func NewGetFundingStatsTask(client *api.Client, symbol string, limit int, resultChan chan<- FundingStatsResult, priority int) *GetFundingStatsTask {
	return &GetFundingStatsTask{
		BaseTask: scheduler.BaseTask{
			Name:     fmt.Sprintf("GetFundingStats_%s_%d", symbol, limit),
			Priority: priority,
			RetryPolicy: scheduler.RetryPolicy{
				MaxRetries:  3,
				BackoffBase: 500 * time.Millisecond,
			},
		},
		Client:     client,
		Symbol:     symbol,
		Limit:      limit,
		ResultChan: resultChan,
	}
}

// Added: Function to create funding stats task with time range
func NewGetFundingStatsTaskWithTimeRange(
	client *api.Client,
	symbol string,
	start int64,
	end int64,
	limit int,
	resultChan chan<- FundingStatsResult,
	priority int,
) *GetFundingStatsTask {
	return &GetFundingStatsTask{
		BaseTask: scheduler.BaseTask{
			Name:     fmt.Sprintf("GetFundingStats_%s_%d_%d_%d", symbol, start, end, limit),
			Priority: priority,
			RetryPolicy: scheduler.RetryPolicy{
				MaxRetries:  3,
				BackoffBase: 500 * time.Millisecond,
			},
		},
		Client:     client,
		Symbol:     symbol,
		Start:      start,
		End:        end,
		Limit:      limit,
		ResultChan: resultChan,
	}
}

func (t *GetFundingStatsTask) Execute(ctx context.Context) error {
	var err error
	var stats []api.FundingStats

	// Retry logic
	for attempt := 0; attempt <= t.RetryPolicy.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			t.ResultChan <- FundingStatsResult{Error: ctx.Err()}
			return ctx.Err()
		default:
			// Use different API call based on whether time range is provided
			if t.Start > 0 || t.End > 0 {
				stats, err = t.Client.GetFundingStatsWithTimeRangeWithContext(ctx, t.Symbol, t.Start, t.End, t.Limit)
			} else {
				stats, err = t.Client.GetFundingStatsWithContext(ctx, t.Symbol, t.Limit)
			}

			if err == nil {
				t.ResultChan <- FundingStatsResult{Data: stats}
				return nil
			}

			// If not the last attempt, wait before retrying
			if attempt < t.RetryPolicy.MaxRetries {
				backoffDuration := time.Duration(math.Pow(2, float64(attempt))) *
					t.RetryPolicy.BackoffBase
				select {
				case <-ctx.Done():
					t.ResultChan <- FundingStatsResult{Error: ctx.Err()}
					return ctx.Err()
				case <-time.After(backoffDuration):
					// Continue to next attempt
				}
			}
		}
	}

	// All retries failed
	t.ResultChan <- FundingStatsResult{Error: err}
	return err
}

// 4. Funding Ticker Task
type GetFundingTickerTask struct {
	scheduler.BaseTask
	Client     *api.Client
	Symbol     string
	ResultChan chan<- FundingTickerResult
	Storage    db.Storage // Optional
}

func NewGetFundingTickerTask(client *api.Client, symbol string, resultChan chan<- FundingTickerResult, priority int) *GetFundingTickerTask {
	return &GetFundingTickerTask{
		BaseTask: scheduler.BaseTask{
			Name:     fmt.Sprintf("GetFundingTicker_%s", symbol),
			Priority: priority,
			RetryPolicy: scheduler.RetryPolicy{
				MaxRetries:  3,
				BackoffBase: 500 * time.Millisecond,
			},
		},
		Client:     client,
		Symbol:     symbol,
		ResultChan: resultChan,
	}
}

func (t *GetFundingTickerTask) Execute(ctx context.Context) error {
	// Create cancelable request using context
	result, err := t.Client.GetFundingTickerWithContext(ctx, t.Symbol)

	// Send result to channel
	t.ResultChan <- FundingTickerResult{
		Data:  result,
		Error: err,
	}

	return err
}
