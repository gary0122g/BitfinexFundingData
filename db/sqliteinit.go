package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

// InitDB initializes the database connection and creates necessary tables
func InitDB(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}

	// Ensure connection is available
	if err = db.Ping(); err != nil {
		return nil, err
	}

	// Create tables
	if err = CreateTables(db); err != nil {
		return nil, err
	}

	return db, nil
}

// CreateTables creates the database schema
func CreateTables(db *sql.DB) error {
	createTableSQL := `
	-- FundingStats table
	CREATE TABLE IF NOT EXISTS funding_stats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		currency TEXT NOT NULL,
		mts INTEGER NOT NULL,
		frr REAL,
		avg_period REAL,
		funding_amount REAL,
		funding_amount_used REAL,
		funding_below_threshold REAL,
		created_at INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000),
		UNIQUE(currency, mts)
	);
	CREATE INDEX IF NOT EXISTS idx_funding_stats_currency_mts ON funding_stats(currency, mts);
	
	-- FundingTicker table
	CREATE TABLE IF NOT EXISTS funding_ticker (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		currency TEXT NOT NULL,
		timestamp INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000),
		frr REAL,
		bid REAL,
		bid_period INTEGER,
		bid_size REAL,
		ask REAL,
		ask_period INTEGER,
		ask_size REAL,
		daily_change REAL,
		daily_change_percent REAL,
		last_price REAL,
		volume REAL,
		high REAL,
		low REAL,
		frr_amount_available REAL,
		created_at INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000),
		UNIQUE(currency, timestamp)
	);
	CREATE INDEX IF NOT EXISTS idx_funding_ticker_currency_timestamp ON funding_ticker(currency, timestamp);
	
	-- FundingBook table
	CREATE TABLE IF NOT EXISTS funding_book (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		currency TEXT NOT NULL,
		timestamp INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000),
		rate REAL,
		period INTEGER,
		count INTEGER,
		amount REAL,
		is_bid BOOLEAN,
		created_at INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000)
	);
	CREATE INDEX IF NOT EXISTS idx_funding_book_currency_timestamp ON funding_book(currency, timestamp);
	
	-- RawFundingBook table
	CREATE TABLE IF NOT EXISTS raw_funding_book (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		currency TEXT NOT NULL,
		timestamp INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000),
		offer_id INTEGER,
		period INTEGER,
		rate REAL,
		amount REAL,
		is_bid BOOLEAN,
		created_at INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000)
	);
	CREATE INDEX IF NOT EXISTS idx_raw_funding_book_currency_timestamp ON raw_funding_book(currency, timestamp);
	
	-- TradingBook table
	CREATE TABLE IF NOT EXISTS trading_book (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		symbol TEXT NOT NULL,
		timestamp INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000),
		price REAL,
		count INTEGER,
		amount REAL,
		is_bid BOOLEAN,
		created_at INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000)
	);
	CREATE INDEX IF NOT EXISTS idx_trading_book_symbol_timestamp ON trading_book(symbol, timestamp);
	
	-- RawTradingBook table
	CREATE TABLE IF NOT EXISTS raw_trading_book (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		symbol TEXT NOT NULL,
		timestamp INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000),
		order_id INTEGER,
		price REAL,
		amount REAL,
		is_bid BOOLEAN,
		created_at INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000)
	);
	CREATE INDEX IF NOT EXISTS idx_raw_trading_book_symbol_timestamp ON raw_trading_book(symbol, timestamp);
	
	-- TradingTicker table
	CREATE TABLE IF NOT EXISTS trading_ticker (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		symbol TEXT NOT NULL,
		timestamp INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000),
		bid REAL,
		bid_size REAL,
		ask REAL,
		ask_size REAL,
		daily_change REAL,
		daily_change_relative REAL,
		last_price REAL,
		volume REAL,
		high REAL,
		low REAL,
		created_at INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000),
		UNIQUE(symbol, timestamp)
	);
	CREATE INDEX IF NOT EXISTS idx_trading_ticker_symbol_timestamp ON trading_ticker(symbol, timestamp);

	-- WebSocket Funding Trades table
	CREATE TABLE IF NOT EXISTS ws_funding_trades (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		trade_id INTEGER NOT NULL,
		currency TEXT NOT NULL,
		timestamp INTEGER NOT NULL,
		amount REAL NOT NULL,
		rate REAL NOT NULL,
		period INTEGER NOT NULL,
		msg_type TEXT NOT NULL, -- 'fte' for trade executed, 'ftu' for trade updated
		created_at INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000),
		UNIQUE(trade_id, msg_type)
	);
	CREATE INDEX IF NOT EXISTS idx_ws_funding_trades_currency_timestamp ON ws_funding_trades(currency, timestamp);
	CREATE INDEX IF NOT EXISTS idx_ws_funding_trades_trade_id ON ws_funding_trades(trade_id);
	`

	_, err := db.Exec(createTableSQL)
	return err
}
