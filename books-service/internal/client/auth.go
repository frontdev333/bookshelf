package client

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"time"
)

type AuthClient struct {
	httpClient *HTTPClient
	serviceKey string
}

func NewAuthClient(baseURL, svcKey string, timeout time.Duration) *AuthClient {
	return &AuthClient{
		httpClient: NewHTTPClient(baseURL, timeout),
		serviceKey: svcKey,
	}
}

type UserPublic struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type VerifyResponse struct {
	Valid     bool      `json:"valid"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	Error     string    `json:"error"`
}

func (c *AuthClient) VerifyToken(ctx context.Context, token string) (*VerifyResponse, error) {
	payload := struct {
		Token string `json:"token"`
	}{
		Token: token,
	}

	pBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Error("AuthClient.VerifyToken", "error", err)
		return nil, err
	}

	resp, err := c.httpClient.Post(ctx, "/internal/v1/auth/verify", bytes.NewBuffer(pBytes), map[string]string{
		"Content-Type":  "application/json",
		"X-Service-Key": c.serviceKey,
	})

	if err != nil {
		slog.Error("AuthClient.VerifyToken", "error", err)
		return nil, err
	}

	defer resp.Body.Close()

	var dto VerifyResponse

	if err = json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		slog.Error("AuthClient.VerifyToken", "error", err)
		return nil, err
	}

	return &dto, nil
}

func (c *AuthClient) GetUsersByIDs(ctx context.Context, ids []string) ([]UserPublic, error) {
	payload := struct {
		IDs []string `json:"ids"`
	}{
		IDs: ids,
	}

	pBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Error("AuthClient.GetUsersByIDs", "error", err)
		return nil, err
	}

	resp, err := c.httpClient.Post(ctx, "/internal/v1/users/batch", pBytes, map[string]string{
		"Content-Type":  "application/json",
		"X-Service-Key": c.serviceKey,
	})

	if err != nil {
		slog.Error("AuthClient.GetUsersByIDs", "error", err)
		return nil, err
	}

	defer resp.Body.Close()

	var dto []UserPublic

	if err = json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		slog.Error("AuthClient.GetUsersByIDs", "error", err)
		return nil, err
	}

	return dto, nil
}
