package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL     string
	internalKey string
	httpClient  *http.Client
}

func NewClient(baseURL, internalKey string) *Client {
	return &Client{
		baseURL:     baseURL,
		internalKey: internalKey,
		httpClient:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (c *Client) Query(question, scope, userID string) (io.ReadCloser, error) {
	params := url.Values{
		"q":       {question},
		"scope":   {scope},
		"user_id": {userID},
	}
	endpoint := fmt.Sprintf("%s/v1/ai/query?%s", c.baseURL, params.Encode())

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	if c.internalKey != "" {
		req.Header.Set("X-Internal-Key", c.internalKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("ai service returned status %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func (c *Client) Chat(message, scope, userID string, allowMutations bool, history []map[string]string) (io.ReadCloser, error) {
	payload := struct {
		Message        string              `json:"message"`
		Scope          string              `json:"scope"`
		UserID         string              `json:"user_id"`
		AllowMutations bool                `json:"allow_mutations"`
		History        []map[string]string `json:"history"`
	}{
		Message:        message,
		Scope:          scope,
		UserID:         userID,
		AllowMutations: allowMutations,
		History:        history,
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/v1/ai/chat", c.baseURL)
	req, err := http.NewRequest(http.MethodPost, endpoint, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")
	if c.internalKey != "" {
		req.Header.Set("X-Internal-Key", c.internalKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("ai service returned status %d", resp.StatusCode)
	}
	return resp.Body, nil
}
