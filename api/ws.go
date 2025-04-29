package api

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	bitfinexWSURL = "wss://api-pub.bitfinex.com/ws/2"
	maxRetries    = 5
	retryDelay    = 5 * time.Second
)

type FundingTrade struct {
	ID     int64   `json:"id"`
	MTS    int64   `json:"mts"`
	Amount float64 `json:"amount"`
	Rate   float64 `json:"rate"`
	Period int     `json:"period"`
}

type SubscribeMessage struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
	Symbol  string `json:"symbol"`
}

type SubscribedResponse struct {
	Event    string `json:"event"`
	Channel  string `json:"channel"`
	ChanID   int    `json:"chanId"`
	Symbol   string `json:"symbol"`
	Currency string `json:"currency"`
}

type WebSocketClient struct {
	conn       *websocket.Conn
	mu         sync.Mutex
	subscribed bool
	stopChan   chan struct{}
	reconnect  bool
}

func NewWebSocketClient() *WebSocketClient {
	return &WebSocketClient{
		stopChan:  make(chan struct{}),
		reconnect: true,
	}
}

func (wsc *WebSocketClient) Connect() error {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()

	if wsc.conn != nil {
		return nil
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	var err error
	for i := 0; i < maxRetries; i++ {
		wsc.conn, _, err = dialer.Dial(bitfinexWSURL, nil)
		if err == nil {
			log.Printf("Successfully connected to Bitfinex WebSocket")
			return nil
		}
		log.Printf("Failed to connect to Bitfinex (attempt %d/%d): %v", i+1, maxRetries, err)
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	return fmt.Errorf("failed to connect to Bitfinex after %d attempts: %v", maxRetries, err)
}

func (wsc *WebSocketClient) SubscribeToFundingTrades(symbol string) error {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()

	if wsc.conn == nil {
		return fmt.Errorf("not connected to Bitfinex")
	}

	subscribeMsg := SubscribeMessage{
		Event:   "subscribe",
		Channel: "trades",
		Symbol:  symbol,
	}

	msg, err := json.Marshal(subscribeMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal subscribe message: %v", err)
	}

	err = wsc.conn.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		return fmt.Errorf("failed to send subscribe message: %v", err)
	}

	wsc.subscribed = true
	return nil
}

func (wsc *WebSocketClient) HandleFundingTrades(handler func(trade FundingTrade, msgType string) error) {
	go func() {
		for {
			select {
			case <-wsc.stopChan:
				return
			default:
				if err := wsc.readAndHandleMessages(handler); err != nil {
					if wsc.reconnect {
						log.Printf("WebSocket error, attempting to reconnect: %v", err)
						wsc.reconnectWebSocket()
					} else {
						log.Printf("WebSocket error: %v", err)
						return
					}
				}
			}
		}
	}()
}

func (wsc *WebSocketClient) readAndHandleMessages(handler func(trade FundingTrade, msgType string) error) error {
	wsc.mu.Lock()
	if wsc.conn == nil {
		wsc.mu.Unlock()
		return fmt.Errorf("not connected to Bitfinex")
	}
	wsc.mu.Unlock()

	_, message, err := wsc.conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("error reading message: %v", err)
	}

	// First check if it's a subscription response
	var subResp SubscribedResponse
	if err := json.Unmarshal(message, &subResp); err == nil && subResp.Event == "subscribed" {
		log.Printf("Successfully subscribed to channel %d for %s", subResp.ChanID, subResp.Symbol)
		return nil
	}

	// Handle trade messages
	var data []interface{}
	if err := json.Unmarshal(message, &data); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		return nil
	}

	if len(data) < 3 {
		return nil
	}

	// Check if it's a trade message
	if msgType, ok := data[1].(string); ok {
		if msgType == "fte" || msgType == "ftu" {
			if tradeData, ok := data[2].([]interface{}); ok && len(tradeData) >= 5 {
				trade := FundingTrade{
					ID:     int64(tradeData[0].(float64)),
					MTS:    int64(tradeData[1].(float64)),
					Amount: tradeData[2].(float64),
					Rate:   tradeData[3].(float64),
					Period: int(tradeData[4].(float64)),
				}
				if err := handler(trade, msgType); err != nil {
					log.Printf("Error handling trade: %v", err)
				}
			}
		}
	}

	return nil
}

func (wsc *WebSocketClient) reconnectWebSocket() {
	wsc.mu.Lock()
	if wsc.conn != nil {
		wsc.conn.Close()
		wsc.conn = nil
	}
	wsc.mu.Unlock()

	for {
		if err := wsc.Connect(); err != nil {
			log.Printf("Failed to reconnect: %v", err)
			time.Sleep(retryDelay)
			continue
		}

		// Re-subscribe if needed
		if wsc.subscribed {
			if err := wsc.SubscribeToFundingTrades("fUSD"); err != nil {
				log.Printf("Failed to re-subscribe: %v", err)
				continue
			}
		}

		return
	}
}

func (wsc *WebSocketClient) Close() {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()

	wsc.reconnect = false
	close(wsc.stopChan)
	if wsc.conn != nil {
		wsc.conn.Close()
		wsc.conn = nil
	}
}
