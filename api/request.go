package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

func NewClient() *Client {
	return &Client{
		APIKey:     "your_api_key",
		APISecret:  "your_api_secret",
		HTTPClient: &http.Client{},
		BaseURL:    "https://api.bitfinex.com",
	}
}

func (c *Client) SendRequest(method, path string, body interface{}) ([]byte, error) {
	// Serialize request body
	var bodyStr string
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("error serializing request body: %w", err)
		}
		bodyStr = string(jsonData)
	}

	// Generate nonce
	nonce := strconv.FormatInt(time.Now().UnixNano()/1000000, 10)

	// Create signature payload
	signaturePayload := "/api/" + path + nonce + bodyStr

	// Calculate signature
	h := hmac.New(sha512.New384, []byte(c.APISecret))
	h.Write([]byte(signaturePayload))
	signature := hex.EncodeToString(h.Sum(nil))

	// Create request
	url := c.BaseURL + "/" + path
	req, err := http.NewRequest(method, url, bytes.NewBufferString(bodyStr))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("bfx-nonce", nonce)
	req.Header.Set("bfx-apikey", c.APIKey)
	req.Header.Set("bfx-signature", signature)

	// Send request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		var errorResp []interface{}
		err := json.Unmarshal(respBody, &errorResp)

		bfxErr := BitfinexError{
			StatusCode: resp.StatusCode,
			RawBody:    string(respBody),
		}

		if err == nil && len(errorResp) >= 3 {
			if code, ok := errorResp[1].(string); ok {
				bfxErr.ErrorCode = code
			}
			if msg, ok := errorResp[2].(string); ok {
				bfxErr.Message = msg
			}
		} else {
			bfxErr.Message = "Failed to parse error response"
		}

		return nil, bfxErr
	}

	return respBody, nil
}

func (e BitfinexError) Error() string {
	return fmt.Sprintf("Bitfinex API Error [%s]: %s (Status Code: %d)",
		e.ErrorCode, e.Message, e.StatusCode)
}
