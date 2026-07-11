package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
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

type UserSummary struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func (u *UserPublic) ToSummary() UserSummary {
	return UserSummary{
		ID:       u.ID,
		Username: u.Username,
	}
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

func (c *AuthClient) GetUserByID(ctx context.Context, id string) (UserPublic, error) {
	payload := struct {
		ID string `json:"id"`
	}{
		ID: id,
	}

	pBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Error("AuthClient.GetUsersByID", "error", err)
		return UserPublic{}, err
	}

	resp, err := c.httpClient.Post(ctx, "/api/v1/users/me", pBytes, map[string]string{
		"Content-Type":  "application/json",
		"X-Service-Key": c.serviceKey,
	})

	if err != nil {
		slog.Error("AuthClient.GetUsersByID", "error", err)
		return UserPublic{}, err
	}

	defer resp.Body.Close()

	var dto UserPublic

	if err = json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		slog.Error("AuthClient.GetUsersByID", "error", err)
		return UserPublic{}, err
	}

	return dto, nil
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

	if resp.StatusCode != http.StatusOK {
		slog.Warn("AuthClient.GetUsersByIDs", "status", resp.Status, "code", resp.StatusCode)
		return nil, fmt.Errorf("AuthClient.GetUsersByIDs, status: %s code: %d", resp.Status, resp.StatusCode)
	}

	var dto []UserPublic

	if err = json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		slog.Error("AuthClient.GetUsersByIDs", "error", err)
		return nil, err
	}

	return dto, nil
}
