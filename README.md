# BitfinexFundingData

A Go application for collecting, storing, and analyzing funding data from the Bitfinex cryptocurrency exchange API.

## Overview

BitfinexFundingData is a comprehensive tool designed to interact with the Bitfinex API to collect various types of funding and trading data, store it in a SQLite database, and provide access to historical and real-time market information.

The application focuses primarily on funding statistics, order books, and market tickers, which are especially useful for users engaging in margin funding activities on the Bitfinex platform.

## Features

- **Funding Statistics**: Collect and store historical funding rate statistics for various currencies
- **Order Books**: Retrieve both aggregated and raw order books for funding and trading
- **Market Tickers**: Monitor real-time and historical ticker data for trading pairs and funding currencies
- **Real-time Funding Trades**: WebSocket connection for live funding trade updates
- **Scheduled Data Collection**: Task scheduler for periodic data retrieval
- **Persistent Storage**: SQLite database for storing all collected data
- **Context-aware API Calls**: All API requests support context for proper cancellation and timeout handling
- **Web Interface**: Browser-based dashboard for visualizing and monitoring data

## Project Structure

- `api/`: Bitfinex API client implementation
  - `fundingStat.go`: Funding statistics endpoints
  - `ticker.go`: Trading and funding ticker endpoints
- `db/`: Database layer for persistent storage
  - `sqlite.go`: SQLite implementation of the storage interface
- `scheduler/`: Task scheduling system
  - `scheduler_impl.go`: Implementation of the task scheduler
- `task/`: Task definitions for data collection
  - `task.go`: Specific task implementations
- `web/`: Frontend interface components
  - `templates/`: HTML templates for the web interface
  - `static/`: CSS, JavaScript, and other static assets

## Usage

### Setting Up

1. Clone the repository
```bash
git clone https://github.com/gary0122g/BitfinexFundingData.git
cd BitfinexFundingData
```

2. Install dependencies
```bash
go mod tidy
```

3. Run the application
```bash
go run main.go
```

4. Access the web interface
```
Open your browser and navigate to http://localhost:8080
```

### Web Interface

The application includes a web-based dashboard accessible at `http://localhost:8080` when the application is running. The interface provides:

- Real-time funding rate charts and statistics
- Historical funding data visualization
- Order book depth charts
- Interactive data filtering and exploration
- Market overview and monitoring panels
- Real-time funding trade updates

#### Available Views

1. **Funding Statistics View**
   - Displays historical funding rate (FRR) trends
   - Shows detailed statistics including average period, funding amount, and usage
   - Interactive chart for visualizing rate changes over time
   - Filterable by currency and time range

2. **Funding Ticker View**
   - Real-time funding market overview
   - Current FRR, best bid/ask rates and sizes
   - 24-hour price changes and volume
   - Available funding amounts

3. **Funding Order Book View**
   - Live order book for funding markets
   - Separate views for offers (lending) and bids (borrowing)
   - Order depth visualization
   - Real-time updates of market depth

4. **FRR vs Trade Rate Comparison View**
   - Compares FRR with actual trade rates
   - Visualizes rate deviations
   - Historical trade data analysis
   - Interactive filtering and time range selection

### Real-time Funding Trades

The application maintains a WebSocket connection to Bitfinex to receive real-time funding trade updates. This feature:

- Automatically reconnects on connection loss
- Maintains subscription state
- Handles trade messages and subscription responses
- Stores trades in the database for historical analysis

Example usage:

```go
// Create a new WebSocket client
wsClient := api.NewWebSocketClient()

// Connect to Bitfinex WebSocket
if err := wsClient.Connect(); err != nil {
    log.Printf("Failed to connect: %v", err)
    return
}
defer wsClient.Close()

// Subscribe to funding trades for a specific currency
if err := wsClient.SubscribeToFundingTrades("fUSD"); err != nil {
    log.Printf("Failed to subscribe: %v", err)
    return
}

// Handle incoming trades
wsClient.HandleFundingTrades(func(trade api.FundingTrade, msgType string) error {
    // Process the trade
    log.Printf("Received trade: %+v", trade)
    return nil
})
```

The WebSocket client includes:
- Automatic reconnection with configurable retry attempts
- Thread-safe operations
- Graceful shutdown handling
- Error handling and logging
- State tracking for connection and subscription status

### Collecting Funding Statistics

```go
// Create a client
client := api.NewClient("https://api.bitfinex.com")

// Get funding statistics for USD
stats, err := client.GetFundingStats("fUSD", 100)
if err != nil {
    log.Fatal(err)
}

// Process the stats
for _, stat := range stats {
    fmt.Printf("Time: %d, FRR: %.4f%%, Average Period: %.2f days\n",
        stat.MTS,
        stat.FRR * 100,
        stat.AveragePeriod)
}
```

### Collecting Ticker Data

```go
// Get trading ticker (e.g., BTC/USD)
tradingTicker, err := client.GetTradingTicker("tBTCUSD")
if err != nil {
    log.Fatal(err)
}

// Get funding ticker (e.g., USD)
fundingTicker, err := client.GetFundingTicker("fUSD")
if err != nil {
    log.Fatal(err)
}
```

### Scheduling Regular Data Collection

```go
// Create a scheduler with 5 workers and a queue size of 100
scheduler := scheduler.NewScheduler(5, 100)
scheduler.Start()

// Schedule recurring funding stats collection every hour
statsTask := task.NewGetFundingStatsTask(client, "fUSD", 100, resultChan, 1)
scheduler.ScheduleRecurring(context.Background(), statsTask, time.Hour)
```

## Database Structure

The application stores data in a SQLite database with tables for:

- `funding_stats`: Funding statistics for currencies
- `funding_ticker`: Real-time funding ticker information
- `trading_ticker`: Real-time trading ticker information
- `funding_book`: Aggregated funding order book data
- `raw_funding_book`: Raw funding order book data
- `trading_book`: Aggregated trading order book data
- `raw_trading_book`: Raw trading order book data

## Future Improvements

- Enhancing the web interface with additional visualization options



## Acknowledgments

- [Bitfinex API Documentation](https://docs.bitfinex.com/docs)
