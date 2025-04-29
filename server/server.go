package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gary0122g/BitfinexFundingData/db"
	"github.com/gorilla/mux"
)

// APIServer handles API requests
type APIServer struct {
	database *db.Database
	router   *mux.Router
}

// NewAPIServer creates a new API server
func NewAPIServer(database *db.Database) *APIServer {
	server := &APIServer{
		database: database,
		router:   mux.NewRouter(),
	}
	server.routes()
	return server
}

// routes sets up API routes
func (s *APIServer) routes() {
	// Static file service
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Homepage
	s.router.HandleFunc("/", s.handleHome).Methods("GET")

	// API endpoints
	api := s.router.PathPrefix("/api").Subrouter()

	// FundingStats API
	api.HandleFunc("/funding-stats/{currency}", s.handleGetFundingStats).Methods("GET")

	// FundingTicker API
	api.HandleFunc("/funding-ticker/{currency}", s.handleGetFundingTicker).Methods("GET")

	// FundingBook API
	api.HandleFunc("/funding-book/{currency}", s.handleGetFundingBook).Methods("GET")
	api.HandleFunc("/raw-funding-book/{currency}", s.handleGetRawFundingBook).Methods("GET")

	// Funding Trades Comparison API
	api.HandleFunc("/funding-trades-comparison/{currency}", s.handleGetFundingTradesComparison).Methods("GET")
}

// Start launches the API server
func (s *APIServer) Start(addr string) error {
	fmt.Printf("API server listening on %s\n", addr)
	return http.ListenAndServe(addr, s.router)
}

// handleHome processes homepage requests
func (s *APIServer) handleHome(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/index.html")
}

// handleGetFundingStats processes requests for funding statistics data
func (s *APIServer) handleGetFundingStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	currency := vars["currency"]
	if !strings.HasPrefix(currency, "f") {
		currency = "f" + currency
	}

	// Get query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // Default limit
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Get data from database
	stats, err := s.database.GetFundingStats(currency, limit)
	if err != nil {
		http.Error(w, "Failed to retrieve funding statistics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleGetFundingTicker processes requests for funding ticker data
func (s *APIServer) handleGetFundingTicker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	currency := vars["currency"]
	if !strings.HasPrefix(currency, "f") {
		currency = "f" + currency
	}

	// Get data from database
	ticker, err := s.database.GetLatestFundingTicker(currency)
	if err != nil {
		http.Error(w, "Failed to retrieve funding ticker data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticker)
}

// handleGetFundingBook processes requests for funding book data
func (s *APIServer) handleGetFundingBook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	currency := vars["currency"]
	if !strings.HasPrefix(currency, "f") {
		currency = "f" + currency
	}

	// Get data from database
	books, err := s.database.GetLatestFundingBook(currency)
	if err != nil {
		http.Error(w, "Failed to retrieve funding book data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}

// handleGetRawFundingBook processes requests for raw funding book data
func (s *APIServer) handleGetRawFundingBook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	currency := vars["currency"]
	if !strings.HasPrefix(currency, "f") {
		currency = "f" + currency
	}

	// Get data from database
	rawBooks, err := s.database.GetLatestRawFundingBook(currency)
	if err != nil {
		http.Error(w, "Failed to retrieve raw funding book data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rawBooks)
}

// handleGetFundingTradesComparison processes requests for funding trades comparison data
func (s *APIServer) handleGetFundingTradesComparison(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	currency := vars["currency"]
	if !strings.HasPrefix(currency, "f") {
		currency = "f" + currency
	}

	// Get query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // Default limit
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Get funding stats data
	stats, err := s.database.GetFundingStats(currency, limit)
	if err != nil {
		http.Error(w, "Failed to retrieve funding statistics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get historical funding trades data
	startTime := time.Now().Add(-24 * time.Hour) // Last 24 hours
	endTime := time.Now()
	trades, err := s.database.GetHistoricalWSFundingTrades(currency, startTime, endTime, limit)
	if err != nil {
		http.Error(w, "Failed to retrieve funding trades: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Combine and format the data
	response := map[string]interface{}{
		"stats":  stats,
		"trades": trades,
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
