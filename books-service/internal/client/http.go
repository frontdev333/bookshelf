package client

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

type HTTPClient struct {
	Client  *http.Client `json:"client"`
	BaseURL string       `json:"base_url"`
}

func NewHTTPClient(baseURL string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		Client: &http.Client{
			Timeout: timeout,
		},
		BaseURL: baseURL,
	}
}

func (c *HTTPClient) Get(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	path, err := url.JoinPath(c.BaseURL, path)
	if err != nil {
		slog.Error("HTTPClient.Get() url.JoinPath", "error", err)
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	res, err := c.Client.Do(req)
	if err != nil {
		slog.Error("HTTPClient.Get() client.Do", "error", err)
		return nil, err
	}

	return res, nil
}

func (c *HTTPClient) Post(ctx context.Context, path string, body []byte, headers map[string]string) (*http.Response, error) {
	path, err := url.JoinPath(c.BaseURL, path)
	if err != nil {
		slog.Error("HTTPClient.Post() url.JoinPath", "error", err)
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	res, err := c.Client.Do(req)
	if err != nil {
		slog.Error("HTTPClient.Post() client.Do", "error", err)
		return nil, err
	}

	return res, nil
}
