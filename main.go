package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gary0122g/BitfinexFundingData/api"
	"github.com/gary0122g/BitfinexFundingData/db"
	"github.com/gary0122g/BitfinexFundingData/scheduler"
	"github.com/gary0122g/BitfinexFundingData/server"
	"github.com/gary0122g/BitfinexFundingData/task"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Unable to get current working directory: %v", err)
	}

	dbPath := filepath.Join(currentDir, "test.db")

	// Check if database file exists
	_, err = os.Stat(dbPath)
	if os.IsNotExist(err) {
		log.Printf("Database file %s does not exist, will create a new database", dbPath)
		// Can continue, InitDB will create a new database
	}

	// Initialize database and get connection
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer sqlDB.Close()

	fmt.Println("Successfully connected to database!")

	// Create database wrapper
	database := db.NewDatabase(sqlDB)
	apiServer := server.NewAPIServer(database)
	// Create scheduler
	scheduler := scheduler.NewScheduler(5, 50) // 5 workers, queue size 50
	scheduler.Start()
	defer scheduler.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create API client
	client := api.NewClient()

	currencies := []string{"fUSD", "fUST"}

	// Get initial data for each currency
	for _, currency := range currencies {
		// Get initial FundingStats data
		if err := fetchInitialFundingStats(ctx, client, database, currency); err != nil {
			log.Printf("Failed to get initial FundingStats data for %s: %v", currency, err)
		}

		// Get initial FundingTicker data
		if err := fetchInitialFundingTicker(ctx, client, database, currency); err != nil {
			log.Printf("Failed to get initial FundingTicker data for %s: %v", currency, err)
		}

		// Get initial FundingBook data
		if err := fetchInitialFundingBook(ctx, client, database, currency); err != nil {
			log.Printf("Failed to get initial FundingBook data for %s: %v", currency, err)
		}
	}

	// Create periodic tasks for each currency
	for _, currency := range currencies {
		currency := currency // Create local copy for use in closures

		// Create hourly FundingStats task
		hourlyStatsTask := scheduler.NewPeriodicTask(
			fmt.Sprintf("FundingStats_%s_Hourly", currency),
			1*time.Hour, // Run once per hour
			func(ctx context.Context) error {
				return updateFundingStats(ctx, client, database, currency)
			},
			3, // Number of retries
		)
		scheduler.SubmitTask(hourlyStatsTask)
		log.Printf("Set up hourly FundingStats data collection task for %s", currency)

		tickerTask := scheduler.NewPeriodicTask(
			fmt.Sprintf("FundingTicker_%s", currency),
			1*time.Hour,
			func(ctx context.Context) error {
				return updateFundingTicker(ctx, client, database, currency)
			},
			3, // Number of retries
		)
		scheduler.SubmitTask(tickerTask)
		log.Printf("Set up hourly FundingTicker data collection task for %s", currency)

		// Create FundingBook task to run every minute
		bookTask := scheduler.NewPeriodicTask(
			fmt.Sprintf("FundingBook_%s", currency),
			1*time.Minute, // Run every minute
			func(ctx context.Context) error {
				return updateFundingBook(ctx, client, database, currency)
			},
			3, // Number of retries
		)
		scheduler.SubmitTask(bookTask)
		log.Printf("Set up minute FundingBook data collection task for %s", currency)
	}

	// Create a signal capture
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Start API server in a new goroutine
	go func() {
		if err := apiServer.Start(":8080"); err != nil {
			log.Fatalf("Failed to start API server: %v", err)
		}
	}()

	// Wait for termination signal
	<-signalChan
	fmt.Println("Received stop signal, gracefully exiting...")
	scheduler.Stop() // Stop scheduler
}

// Get initial FundingStats data
func fetchInitialFundingStats(ctx context.Context, client *api.Client, database *db.Database, currency string) error {
	// Check if data already exists
	stats, err := database.GetFundingStats(currency, 1)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check database: %v", err)
	}

	// If data already exists, no need to get initial data
	if len(stats) > 0 {
		log.Printf("FundingStats records for %s already exist in database, skipping initial data collection", currency)
		return nil
	}

	// Create result channel
	resultChan := make(chan task.FundingStatsResult, 1)

	// Create and execute task to get initial 250 records
	statsTask := task.NewGetFundingStatsTask(client, currency, 250, resultChan, 3)
	if err := statsTask.Execute(ctx); err != nil {
		return fmt.Errorf("failed to execute initial data collection task: %v", err)
	}

	// Get result
	result := <-resultChan
	if result.Error != nil {
		return fmt.Errorf("failed to get initial data: %v", result.Error)
	}

	// Save to database
	count := 0
	for _, stat := range result.Data {
		_, err := database.SaveFundingStats(currency, stat)
		if err != nil {
			log.Printf("failed to save FundingStats data: %v", err)
			continue
		}
		count++
	}

	log.Printf("Successfully retrieved and saved %d initial FundingStats records for %s", count, currency)
	return nil
}

// Update FundingStats data
func updateFundingStats(ctx context.Context, client *api.Client, database *db.Database, currency string) error {
	// Get latest data
	latestStats, err := database.GetFundingStats(currency, 1)
	if err != nil {
		return fmt.Errorf("failed to get latest data: %v", err)
	}

	var latestMts int64 = 0
	if len(latestStats) > 0 {
		latestMts = latestStats[0].MTS
	}

	// Create result channel
	resultChan := make(chan task.FundingStatsResult, 1)

	// Create task to get only the newest record
	statsTask := task.NewGetFundingStatsTaskWithTimeRange(
		client,
		currency,
		latestMts+1, // Start from after the latest timestamp
		0,           // No end time specified
		1,           // Only get 1 record
		resultChan,
		3,
	)

	if err := statsTask.Execute(ctx); err != nil {
		return fmt.Errorf("failed to execute data retrieval task: %v", err)
	}

	// Get result
	result := <-resultChan
	if result.Error != nil {
		return fmt.Errorf("failed to get data: %v", result.Error)
	}

	// If new data exists, save to database
	count := 0
	for _, stat := range result.Data {
		_, err := database.SaveFundingStats(currency, stat)
		if err != nil {
			log.Printf("failed to save FundingStats data: %v", err)
			continue
		}
		count++
	}

	if count > 0 {
		log.Printf("Successfully retrieved and saved %d new FundingStats records for %s", count, currency)
	} else {
		log.Printf("No new FundingStats data for %s", currency)
	}

	return nil
}

// Get initial FundingTicker data
func fetchInitialFundingTicker(ctx context.Context, client *api.Client, database *db.Database, currency string) error {
	// Check if data already exists
	_, err := database.GetLatestFundingTicker(currency)
	if err == nil {
		// Data already exists
		log.Printf("FundingTicker records for %s already exist in database, skipping initial data collection", currency)
		return nil
	} else if err.Error() != "no ticker found for currency: "+currency && err != sql.ErrNoRows {
		// Other error occurred
		return fmt.Errorf("failed to check database: %v", err)
	}

	// Create result channel
	resultChan := make(chan task.FundingTickerResult, 1)

	// Create and execute task to get initial data
	tickerTask := task.NewGetFundingTickerTask(client, currency, resultChan, 3)
	if err := tickerTask.Execute(ctx); err != nil {
		return fmt.Errorf("failed to execute initial data collection task: %v", err)
	}

	// Get result
	result := <-resultChan
	if result.Error != nil {
		return fmt.Errorf("failed to get initial data: %v", result.Error)
	}

	// Save to database
	_, err = database.SaveFundingTicker(currency, *result.Data)
	if err != nil {
		return fmt.Errorf("failed to save initial data: %v", err)
	}

	log.Printf("Successfully retrieved and saved initial FundingTicker data for %s", currency)
	return nil
}

// Update FundingTicker data
func updateFundingTicker(ctx context.Context, client *api.Client, database *db.Database, currency string) error {
	// Create result channel
	resultChan := make(chan task.FundingTickerResult, 1)

	// Create task to get latest data
	tickerTask := task.NewGetFundingTickerTask(client, currency, resultChan, 3)
	if err := tickerTask.Execute(ctx); err != nil {
		return fmt.Errorf("failed to execute data retrieval task: %v", err)
	}

	// Get result
	result := <-resultChan
	if result.Error != nil {
		return fmt.Errorf("failed to get data: %v", result.Error)
	}
	// Save to database
	_, err := database.SaveFundingTicker(currency, *result.Data)
	if err != nil {
		return fmt.Errorf("failed to save data: %v", err)
	}

	log.Printf("Successfully retrieved and saved latest FundingTicker data for %s", currency)
	return nil
}

// Get initial FundingBook data
func fetchInitialFundingBook(ctx context.Context, client *api.Client, database *db.Database, currency string) error {
	// Get raw funding book
	rawBooks, err := client.GetRawFundingBookWithContext(ctx, currency)
	if err != nil {
		return fmt.Errorf("failed to get raw funding book: %v", err)
	}

	// Save raw funding book data
	rawCount := 0
	for _, rawBook := range rawBooks {
		_, err := database.SaveRawFundingBook(currency, rawBook)
		if err != nil {
			log.Printf("failed to save RawFundingBook data: %v", err)
			continue
		}
		rawCount++
	}
	log.Printf("Successfully retrieved and saved %d initial raw funding book records for %s", rawCount, currency)

	// Get aggregated funding book (P0 Precision)
	books, err := client.GetFundingBookWithContext(ctx, currency, api.PrecisionP0)
	if err != nil {
		return fmt.Errorf("failed to get aggregated funding book: %v", err)
	}

	// Save aggregated funding book data
	bookCount := 0
	for _, book := range books {
		_, err := database.SaveFundingBook(currency, book)
		if err != nil {
			log.Printf("failed to save FundingBook data: %v", err)
			continue
		}
		bookCount++
	}
	log.Printf("Successfully retrieved and saved %d initial aggregated funding book records for %s", bookCount, currency)

	return nil
}

// Update FundingBook data
func updateFundingBook(ctx context.Context, client *api.Client, database *db.Database, currency string) error {
	// Get raw funding book
	rawBooks, err := client.GetRawFundingBookWithContext(ctx, currency)
	if err != nil {
		return fmt.Errorf("failed to get raw funding book: %v", err)
	}

	// Save raw funding book data
	rawCount := 0
	for _, rawBook := range rawBooks {
		_, err := database.SaveRawFundingBook(currency, rawBook)
		if err != nil {
			log.Printf("failed to save RawFundingBook data: %v", err)
			continue
		}
		rawCount++
	}
	log.Printf("Successfully retrieved and saved %d latest raw funding book records for %s", rawCount, currency)

	// Get aggregated funding book (P0 Precision)
	books, err := client.GetFundingBookWithContext(ctx, currency, api.PrecisionP0)
	if err != nil {
		return fmt.Errorf("failed to get aggregated funding book: %v", err)
	}

	// Save aggregated funding book data
	bookCount := 0
	for _, book := range books {
		_, err := database.SaveFundingBook(currency, book)
		if err != nil {
			log.Printf("failed to save FundingBook data: %v", err)
			continue
		}
		bookCount++
	}
	log.Printf("Successfully retrieved and saved %d latest aggregated funding book records for %s", bookCount, currency)

	return nil
}
