// Package sarp adapts the sarp-scrapper query endpoint to performance snapshots.
package sarp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/pixelados-net/discord-bot/internal/performance"
)

const userAgent = "discord-bot-business-performance/1.0"

// Client reads business details through sarp-scrapper.
type Client struct {
	config performance.Config
	http   *http.Client
}

// NewClient creates the configured SARP source.
func NewClient(config performance.Config) *Client {
	return &Client{config: config, http: &http.Client{Timeout: config.HTTPTimeout}}
}

// Fetch reads one current business snapshot.
func (client *Client) Fetch(ctx context.Context, businessID int64) (performance.Snapshot, error) {
	body, err := client.query(ctx, fmt.Sprintf("/businesses/%d", businessID))
	if err != nil {
		return performance.Snapshot{}, err
	}
	snapshot, err := decodeSnapshot(body)
	if err != nil {
		return performance.Snapshot{}, err
	}
	if snapshot.BusinessID == 0 {
		snapshot.BusinessID = businessID
	}
	if snapshot.BusinessID != businessID {
		return performance.Snapshot{}, fmt.Errorf("SARP returned business %d, expected %d", snapshot.BusinessID, businessID)
	}
	ranksBody, err := client.query(ctx, fmt.Sprintf("/businesses/%d/ranks", businessID))
	if err != nil {
		return performance.Snapshot{}, fmt.Errorf("fetch SARP ranks: %w", err)
	}
	snapshot.Ranks, err = decodeRanks(ranksBody)
	if err != nil {
		return performance.Snapshot{}, err
	}
	return snapshot, nil
}

func (client *Client) query(ctx context.Context, path string) ([]byte, error) {
	requestBody := struct {
		Provider string `json:"provider"`
		Path     string `json:"path"`
	}{Provider: "gta-rol", Path: path}
	encoded, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.config.Endpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("create SARP request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", userAgent)
	if client.config.EndpointToken != "" {
		request.Header.Set("Authorization", "Bearer "+client.config.EndpointToken)
	}
	response, err := client.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call SARP endpoint: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, client.config.MaxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read SARP response: %w", err)
	}
	if int64(len(body)) > client.config.MaxResponseBytes {
		return nil, fmt.Errorf("SARP response exceeds configured limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("SARP endpoint returned HTTP %d: %s", response.StatusCode, responseMessage(body))
	}
	return body, nil
}

type flexibleInt int64

func (value *flexibleInt) UnmarshalJSON(data []byte) error {
	var number json.Number
	if err := json.Unmarshal(data, &number); err == nil {
		parsed, parseErr := strconv.ParseFloat(string(number), 64)
		if parseErr == nil {
			*value = flexibleInt(math.Round(parsed))
			return nil
		}
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return nil
	}
	clean := strings.NewReplacer("$", "", ",", "", " ", "").Replace(text)
	parsed, err := strconv.ParseFloat(clean, 64)
	if err != nil && clean != "" {
		return fmt.Errorf("invalid integer %q", text)
	}
	*value = flexibleInt(math.Round(parsed))
	return nil
}
