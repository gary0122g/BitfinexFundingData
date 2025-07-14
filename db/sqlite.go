package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gary0122g/BitfinexFundingData/api"
)

// Database encapsulates interaction with the SQLite database
type Database struct {
	db *sql.DB
}

// NewDatabase creates a new database connection
func NewDatabase(db *sql.DB) *Database {
	return &Database{db: db}
}

type Storage interface {
	// FundingStats related methods
	SaveFundingStats(currency string, stats api.FundingStats) (int64, error)
	GetFundingStats(currency string, limit int) ([]api.FundingStats, error)

	// TradingBook related methods
	SaveTradingBook(symbol string, book api.TradingBook) (int64, error)
	GetTradingBook(symbol string, isBid bool, limit int) ([]api.TradingBook, error)

	// FundingBook related methods
	SaveFundingBook(currency string, book api.FundingBook) (int64, error)
	GetLatestFundingBook(currency string) ([]api.FundingBook, error)

	// RawTradingBook related methods
	SaveRawTradingBook(symbol string, book api.RawTradingBook) (int64, error)

	// RawFundingBook related methods
	SaveRawFundingBook(currency string, book api.RawFundingBook) (int64, error)
	GetLatestRawFundingBook(currency string) ([]api.RawFundingBook, error)

	// TradingTicker related methods
	SaveTradingTicker(symbol string, ticker api.TradingTicker) (int64, error)
	GetLatestTradingTicker(symbol string) (api.TradingTicker, error)
	GetHistoricalTradingTickers(symbol string, startTime, endTime time.Time, limit int) ([]api.TradingTicker, error)

	// FundingTicker related methods
	SaveFundingTicker(currency string, ticker api.FundingTicker) (int64, error)
	GetLatestFundingTicker(currency string) (api.FundingTicker, error)
	GetHistoricalFundingTickers(currency string, startTime, endTime time.Time, limit int) ([]api.FundingTicker, error)

	// WebSocket Funding Trades related methods
	SaveWSFundingTrade(currency string, trade api.FundingTrade, msgType string) (int64, error)
	GetLatestWSFundingTrades(currency string, limit int) ([]api.FundingTrade, error)
	GetHistoricalWSFundingTrades(currency string, startTime, endTime time.Time, limit int) ([]api.FundingTrade, error)
}

// SaveFundingStats saves FundingStats data to the database
func (d *Database) SaveFundingStats(currency string, stats api.FundingStats) (int64, error) {
	// If MTS is 0, use current time
	if stats.MTS == 0 {
		stats.MTS = time.Now().UnixMilli()
	}

	query := `
    INSERT INTO funding_stats 
    (currency, mts, frr, avg_period, funding_amount, funding_amount_used, funding_below_threshold)
    VALUES (?, ?, ?, ?, ?, ?, ?)`

	result, err := d.db.Exec(
		query,
		currency,
		stats.MTS,
		stats.FRR,
		stats.AveragePeriod,
		stats.FundingAmount,
		stats.FundingAmountUsed,
		stats.FundingBelowThreshold,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetFundingStats retrieves FundingStats for the specified currency from the database
func (d *Database) GetFundingStats(currency string, limit int) ([]api.FundingStats, error) {
	query := `
    SELECT mts, frr, avg_period, funding_amount, funding_amount_used, funding_below_threshold
    FROM funding_stats
    WHERE currency = ?
    ORDER BY mts DESC
    LIMIT ?`

	rows, err := d.db.Query(query, currency, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []api.FundingStats
	for rows.Next() {
		var s api.FundingStats
		var frr, avgPeriod, fundingAmount, fundingAmountUsed, fundingBelowThreshold sql.NullFloat64
		var mts sql.NullInt64

		if err := rows.Scan(
			&mts,
			&frr,
			&avgPeriod,
			&fundingAmount,
			&fundingAmountUsed,
			&fundingBelowThreshold,
		); err != nil {
			return nil, err
		}

		if mts.Valid {
			s.MTS = mts.Int64
		} else {
			s.MTS = time.Now().UnixMilli() // Use current time as default value
		}

		if frr.Valid {
			s.FRR = frr.Float64 * 365 * 365
		}

		if avgPeriod.Valid {
			s.AveragePeriod = avgPeriod.Float64
		}

		if fundingAmount.Valid {
			s.FundingAmount = fundingAmount.Float64
		}

		if fundingAmountUsed.Valid {
			s.FundingAmountUsed = fundingAmountUsed.Float64
		}

		if fundingBelowThreshold.Valid {
			s.FundingBelowThreshold = fundingBelowThreshold.Float64
		}

		stats = append(stats, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stats, nil
}

// SaveTradingBook saves TradingBook data to the database
func (d *Database) SaveTradingBook(symbol string, book api.TradingBook) (int64, error) {
	query := `
	INSERT INTO trading_book 
	(symbol, price, count, amount, is_bid)
	VALUES (?, ?, ?, ?, ?)`

	// In TradingBook, amount > 0 indicates bid, < 0 indicates ask
	isBid := book.Amount > 0

	result, err := d.db.Exec(
		query,
		symbol,
		book.Price,
		book.Count,
		book.Amount,
		isBid,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetTradingBook retrieves TradingBook data for the specified trading pair from the database
func (d *Database) GetTradingBook(symbol string, isBid bool, limit int) ([]api.TradingBook, error) {
	query := `
	SELECT price, count, amount
	FROM trading_book
	WHERE symbol = ? AND is_bid = ?
	ORDER BY price DESC
	LIMIT ?`

	rows, err := d.db.Query(query, symbol, isBid, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []api.TradingBook
	for rows.Next() {
		var b api.TradingBook
		if err := rows.Scan(
			&b.Price,
			&b.Count,
			&b.Amount,
		); err != nil {
			return nil, err
		}
		books = append(books, b)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return books, nil
}

// SaveFundingBook saves FundingBook data to the database
func (d *Database) SaveFundingBook(currency string, book api.FundingBook) (int64, error) {
	query := `
	INSERT INTO funding_book 
	(currency, rate, period, count, amount, is_bid)
	VALUES (?, ?, ?, ?, ?, ?)`

	// In FundingBook, amount > 0 indicates asks, < 0 indicates bids
	isBid := book.Amount < 0

	result, err := d.db.Exec(
		query,
		currency,
		book.Rate,
		book.Period,
		book.Count,
		book.Amount,
		isBid,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// SaveRawTradingBook saves RawTradingBook data to the database
func (d *Database) SaveRawTradingBook(symbol string, book api.RawTradingBook) (int64, error) {
	query := `
	INSERT INTO raw_trading_book 
	(symbol, order_id, price, amount, is_bid)
	VALUES (?, ?, ?, ?, ?)`

	// In RawTradingBook, amount > 0 indicates bid, < 0 indicates ask
	isBid := book.Amount > 0

	result, err := d.db.Exec(
		query,
		symbol,
		book.OrderID,
		book.Price,
		book.Amount,
		isBid,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// SaveRawFundingBook saves RawFundingBook data to the database
func (d *Database) SaveRawFundingBook(currency string, book api.RawFundingBook) (int64, error) {
	query := `
	INSERT INTO raw_funding_book 
	(currency, offer_id, period, rate, amount, is_bid)
	VALUES (?, ?, ?, ?, ?, ?)`

	// In RawFundingBook, amount > 0 indicates asks, < 0 indicates bids
	isBid := book.Amount < 0

	result, err := d.db.Exec(
		query,
		currency,
		book.OfferID,
		book.Period,
		book.Rate,
		book.Amount,
		isBid,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// SaveTradingTicker saves TradingTicker data to the database
func (d *Database) SaveTradingTicker(symbol string, ticker api.TradingTicker) (int64, error) {
	query := `
	INSERT INTO trading_ticker 
	(symbol, bid, bid_size, ask, ask_size, daily_change, daily_change_relative, 
	last_price, volume, high, low)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := d.db.Exec(
		query,
		symbol,
		ticker.Bid,
		ticker.BidSize,
		ticker.Ask,
		ticker.AskSize,
		ticker.DailyChange,
		ticker.DailyChangeRelative,
		ticker.LastPrice,
		ticker.Volume,
		ticker.High,
		ticker.Low,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetLatestTradingTicker retrieves the latest TradingTicker for the specified trading pair from the database
func (d *Database) GetLatestTradingTicker(symbol string) (api.TradingTicker, error) {
	query := `
	SELECT bid, bid_size, ask, ask_size, daily_change, daily_change_relative, 
	last_price, volume, high, low
	FROM trading_ticker
	WHERE symbol = ?
	ORDER BY timestamp DESC
	LIMIT 1`

	var ticker api.TradingTicker
	err := d.db.QueryRow(query, symbol).Scan(
		&ticker.Bid,
		&ticker.BidSize,
		&ticker.Ask,
		&ticker.AskSize,
		&ticker.DailyChange,
		&ticker.DailyChangeRelative,
		&ticker.LastPrice,
		&ticker.Volume,
		&ticker.High,
		&ticker.Low,
	)

	if err == sql.ErrNoRows {
		return ticker, errors.New("no ticker found for symbol: " + symbol)
	}

	return ticker, err
}

// SaveFundingTicker saves FundingTicker data to the database
func (d *Database) SaveFundingTicker(currency string, ticker api.FundingTicker) (int64, error) {
	query := `
	INSERT INTO funding_ticker 
	(currency, frr, bid, bid_period, bid_size, ask, ask_period, ask_size, 
	daily_change, daily_change_percent, last_price, volume, high, low, frr_amount_available)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := d.db.Exec(
		query,
		currency,
		ticker.FRR,
		ticker.Bid,
		ticker.BidPeriod,
		ticker.BidSize,
		ticker.Ask,
		ticker.AskPeriod,
		ticker.AskSize,
		ticker.DailyChange,
		ticker.DailyChangePercent,
		ticker.LastPrice,
		ticker.Volume,
		ticker.High,
		ticker.Low,
		ticker.FRRAmountAvailable,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetLatestFundingTicker retrieves the latest FundingTicker for the specified currency from the database
func (d *Database) GetLatestFundingTicker(currency string) (api.FundingTicker, error) {
	query := `
	SELECT frr, bid, bid_period, bid_size, ask, ask_period, ask_size, 
	daily_change, daily_change_percent, last_price, volume, high, low, frr_amount_available
	FROM funding_ticker
	WHERE currency = ?
	ORDER BY timestamp DESC
	LIMIT 1`

	var ticker api.FundingTicker
	err := d.db.QueryRow(query, currency).Scan(
		&ticker.FRR,
		&ticker.Bid,
		&ticker.BidPeriod,
		&ticker.BidSize,
		&ticker.Ask,
		&ticker.AskPeriod,
		&ticker.AskSize,
		&ticker.DailyChange,
		&ticker.DailyChangePercent,
		&ticker.LastPrice,
		&ticker.Volume,
		&ticker.High,
		&ticker.Low,
		&ticker.FRRAmountAvailable,
	)

	if err == sql.ErrNoRows {
		return ticker, errors.New("no ticker found for currency: " + currency)
	}

	return ticker, err
}

// GetHistoricalTradingTickers retrieves historical TradingTicker data for the specified trading pair
func (d *Database) GetHistoricalTradingTickers(symbol string, startTime, endTime time.Time, limit int) ([]api.TradingTicker, error) {
	query := `
	SELECT bid, bid_size, ask, ask_size, daily_change, daily_change_relative, 
	last_price, volume, high, low
	FROM trading_ticker
	WHERE symbol = ? AND timestamp BETWEEN ? AND ?
	ORDER BY timestamp DESC
	LIMIT ?`

	rows, err := d.db.Query(query, symbol, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickers []api.TradingTicker
	for rows.Next() {
		var t api.TradingTicker
		if err := rows.Scan(
			&t.Bid,
			&t.BidSize,
			&t.Ask,
			&t.AskSize,
			&t.DailyChange,
			&t.DailyChangeRelative,
			&t.LastPrice,
			&t.Volume,
			&t.High,
			&t.Low,
		); err != nil {
			return nil, err
		}
		tickers = append(tickers, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tickers, nil
}

// GetHistoricalFundingTickers retrieves historical FundingTicker data for the specified currency
func (d *Database) GetHistoricalFundingTickers(currency string, startTime, endTime time.Time, limit int) ([]api.FundingTicker, error) {
	query := `
	SELECT frr, bid, bid_period, bid_size, ask, ask_period, ask_size, 
	daily_change, daily_change_percent, last_price, volume, high, low, frr_amount_available
	FROM funding_ticker
	WHERE currency = ? AND timestamp BETWEEN ? AND ?
	ORDER BY timestamp DESC
	LIMIT ?`

	rows, err := d.db.Query(query, currency, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickers []api.FundingTicker
	for rows.Next() {
		var t api.FundingTicker
		if err := rows.Scan(
			&t.FRR,
			&t.Bid,
			&t.BidPeriod,
			&t.BidSize,
			&t.Ask,
			&t.AskPeriod,
			&t.AskSize,
			&t.DailyChange,
			&t.DailyChangePercent,
			&t.LastPrice,
			&t.Volume,
			&t.High,
			&t.Low,
			&t.FRRAmountAvailable,
		); err != nil {
			return nil, err
		}
		tickers = append(tickers, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tickers, nil
}

// GetLatestFundingBook retrieves the latest funding order book data
func (d *Database) GetLatestFundingBook(currency string) ([]api.FundingBook, error) {
	// Query the latest timestamp
	var latestTimestamp int64
	err := d.db.QueryRow(`
		SELECT MAX(timestamp) 
		FROM funding_book 
		WHERE currency = ?
	`, currency).Scan(&latestTimestamp)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("no funding book found for currency: " + currency)
		}
		return nil, err
	}

	// Query all orders at the latest timestamp
	query := `
	SELECT rate, period, count, amount
	FROM funding_book
	WHERE currency = ? AND timestamp = ?
	ORDER BY CASE WHEN is_bid = 1 THEN rate END DESC,
	         CASE WHEN is_bid = 0 THEN rate END ASC`

	rows, err := d.db.Query(query, currency, latestTimestamp)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []api.FundingBook
	for rows.Next() {
		var b api.FundingBook
		if err := rows.Scan(
			&b.Rate,
			&b.Period,
			&b.Count,
			&b.Amount,
		); err != nil {
			return nil, err
		}
		books = append(books, b)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(books) == 0 {
		return nil, errors.New("no funding book found for currency: " + currency)
	}

	return books, nil
}

// GetLatestRawFundingBook retrieves the latest raw funding order book data
func (d *Database) GetLatestRawFundingBook(currency string) ([]api.RawFundingBook, error) {
	// Query the latest timestamp
	var latestTimestamp int64
	err := d.db.QueryRow(`
		SELECT MAX(timestamp) 
		FROM raw_funding_book 
		WHERE currency = ?
	`, currency).Scan(&latestTimestamp)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("no raw funding book found for currency: " + currency)
		}
		return nil, err
	}

	// Query all orders at the latest timestamp
	query := `
	SELECT offer_id, period, rate, amount
	FROM raw_funding_book
	WHERE currency = ? AND timestamp = ?
	ORDER BY CASE WHEN is_bid = 1 THEN rate END DESC,
	         CASE WHEN is_bid = 0 THEN rate END ASC`

	rows, err := d.db.Query(query, currency, latestTimestamp)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []api.RawFundingBook
	for rows.Next() {
		var b api.RawFundingBook
		if err := rows.Scan(
			&b.OfferID,
			&b.Period,
			&b.Rate,
			&b.Amount,
		); err != nil {
			return nil, err
		}
		books = append(books, b)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(books) == 0 {
		return nil, errors.New("no raw funding book found for currency: " + currency)
	}

	return books, nil
}

// SaveWSFundingTrade saves a WebSocket funding trade to the database
func (d *Database) SaveWSFundingTrade(currency string, trade api.FundingTrade, msgType string) (int64, error) {
	query := `
	INSERT INTO ws_funding_trades 
	(trade_id, currency, timestamp, amount, rate, period, msg_type)
	VALUES (?, ?, ?, ?, ?, ?, ?)`

	result, err := d.db.Exec(
		query,
		trade.ID,
		currency,
		trade.MTS,
		trade.Amount,
		trade.Rate,
		trade.Period,
		msgType,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetLatestWSFundingTrades retrieves the latest WebSocket funding trades for the specified currency
func (d *Database) GetLatestWSFundingTrades(currency string, limit int) ([]api.FundingTrade, error) {
	query := `
	SELECT trade_id, timestamp, amount, rate, period
	FROM ws_funding_trades
	WHERE currency = ?
	ORDER BY timestamp DESC
	LIMIT ?`

	rows, err := d.db.Query(query, currency, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []api.FundingTrade
	for rows.Next() {
		var t api.FundingTrade
		if err := rows.Scan(
			&t.ID,
			&t.MTS,
			&t.Amount,
			&t.Rate,
			&t.Period,
		); err != nil {
			return nil, err
		}
		trades = append(trades, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return trades, nil
}

// GetHistoricalWSFundingTrades retrieves historical WebSocket funding trades for the specified currency
func (d *Database) GetHistoricalWSFundingTrades(currency string, startTime, endTime time.Time, limit int) ([]api.FundingTrade, error) {
	query := `
	SELECT trade_id, timestamp, amount, rate, period
	FROM ws_funding_trades
	WHERE currency = ? AND timestamp BETWEEN ? AND ?
	ORDER BY timestamp DESC
	LIMIT ?`

	rows, err := d.db.Query(query, currency, startTime.UnixMilli(), endTime.UnixMilli(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []api.FundingTrade
	for rows.Next() {
		var t api.FundingTrade
		if err := rows.Scan(
			&t.ID,
			&t.MTS,
			&t.Amount,
			&t.Rate,
			&t.Period,
		); err != nil {
			return nil, err
		}
		trades = append(trades, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return trades, nil
}

// FundingTradeDistribution represents the distribution of funding trades for a given hour
type FundingTradeDistribution struct {
	Hour        string  `json:"hour"`
	AvgRate     float64 `json:"avg_rate"`
	MaxRate     float64 `json:"max_rate"`
	MinRate     float64 `json:"min_rate"`
	TradeCount  int     `json:"trade_count"`
	TotalAmount float64 `json:"total_amount"`
}

// GetFundingTradesDistribution retrieves the distribution of funding trades by hour
func (db *Database) GetFundingTradesDistribution(currency string, limit int) ([]FundingTradeDistribution, error) {
	query := `
		SELECT 
			strftime('%Y-%m-%d %H:00:00', datetime(timestamp/1000, 'unixepoch', 'localtime')) as hour,
			AVG(rate) as avg_rate,
			MAX(rate) as max_rate,
			MIN(rate) as min_rate,
			COUNT(*) as trade_count,
			SUM(amount) as total_amount
		FROM ws_funding_trades
		WHERE currency = ?
		GROUP BY hour
		ORDER BY hour DESC
		LIMIT ?
	`

	rows, err := db.db.Query(query, currency, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query funding trades distribution: %v", err)
	}
	defer rows.Close()

	var distributions []FundingTradeDistribution
	for rows.Next() {
		var d FundingTradeDistribution
		err := rows.Scan(&d.Hour, &d.AvgRate, &d.MaxRate, &d.MinRate, &d.TradeCount, &d.TotalAmount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan funding trade distribution row: %v", err)
		}
		// Convert rates from decimal to percentage
		d.AvgRate *= 100
		d.MaxRate *= 100
		d.MinRate *= 100
		distributions = append(distributions, d)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating funding trade distribution rows: %v", err)
	}

	return distributions, nil
}

// GetDB returns the underlying sql.DB instance
func (d *Database) GetDB() *sql.DB {
	return d.db
}

// GetAllWSFundingTrades 獲取所有WebSocket資金交易（用於初始化分布）
func (d *Database) GetAllWSFundingTrades(currency string) ([]api.FundingTrade, error) {
	query := `
	SELECT trade_id, timestamp, amount, rate, period
	FROM ws_funding_trades
	WHERE currency = ?
	ORDER BY trade_id ASC`

	rows, err := d.db.Query(query, currency)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []api.FundingTrade
	for rows.Next() {
		var t api.FundingTrade
		if err := rows.Scan(&t.ID, &t.MTS, &t.Amount, &t.Rate, &t.Period); err != nil {
			return nil, err
		}
		trades = append(trades, t)
	}

	return trades, rows.Err()
}

// GetWSFundingTradesAfterID 獲取指定ID之後的交易（用於增量更新）
func (d *Database) GetWSFundingTradesAfterID(currency string, lastID int64) ([]api.FundingTrade, error) {
	query := `
	SELECT trade_id, timestamp, amount, rate, period
	FROM ws_funding_trades
	WHERE currency = ? AND trade_id > ?
	ORDER BY trade_id ASC`

	rows, err := d.db.Query(query, currency, lastID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []api.FundingTrade
	for rows.Next() {
		var t api.FundingTrade
		if err := rows.Scan(&t.ID, &t.MTS, &t.Amount, &t.Rate, &t.Period); err != nil {
			return nil, err
		}
		trades = append(trades, t)
	}

	return trades, rows.Err()
}
