package ai

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
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
