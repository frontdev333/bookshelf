package handler

import (
	"bookshelf/auth-svc/internal/domain"
	"bookshelf/auth-svc/internal/service"
	"context"
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type contextKey string

const userIDKey contextKey = "userID"
const requestID contextKey = "requestID"
const version = "1.0.0"

type AuthHandler struct {
	svc       *service.UserService
	jwtSecret string
}

func New(service *service.UserService, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		svc:       service,
		jwtSecret: jwtSecret,
	}
}

type pong struct {
	Status    string    `json:"status,omitempty"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
}

func (h *AuthHandler) Health(w http.ResponseWriter, r *http.Request) {
	v := pong{
		Status:    "ok",
		Version:   version,
		Timestamp: time.Now(),
	}

	writeJSON(w, http.StatusOK, v)
}

func (h *AuthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	type readyPong struct {
		pong
		Database string `json:"database"`
	}

	v := readyPong{
		pong: pong{
			Status:    "ok",
			Version:   version,
			Timestamp: time.Now(),
		},
		Database: "ok",
	}

	writeJSON(w, http.StatusOK, v)
}

func AuthMiddleware(svc *service.UserService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authTitle := r.Header.Get("Authorization")
			if len(strings.TrimSpace(authTitle)) == 0 {
				writeError(w, r, http.StatusUnauthorized, "AUTH_HEADER_REQUIRED", "Authorization header required")
				return
			}

			authSlice := strings.Split(authTitle, " ")
			if strings.ToLower(authSlice[0]) != "bearer" {
				writeError(w, r, http.StatusUnauthorized, "INVALID_AUTH_HEADER_FORMAT", "Invalid authorization header format")
				return
			}

			if len(authSlice) != 2 {
				writeError(w, r, http.StatusUnauthorized, "INVALID_AUTH_HEADER_FORMAT", "Invalid authorization header format")
				return
			}

			userID, err := svc.ValidateToken(authSlice[1])
			if err != nil {
				writeError(w, r, http.StatusUnauthorized, "INVALID_OR_EXPIRED_TOKEN", "Invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ServiceKeyMiddleware(expectedKey string) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actualSKey := r.Header.Get("X-Service-Key")
			if len(strings.TrimSpace(actualSKey)) == 0 {
				writeError(w, r, http.StatusForbidden, "SERVICEKEY_HEADER_REQUIRED", "missing service key")
				return
			}

			if subtle.ConstantTimeCompare([]byte(expectedKey), []byte(actualSKey)) == 0 {
				writeError(w, r, http.StatusForbidden, "INVALID_SERVICEKEY", "invalid service key")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		slog.Error("writeJSON", "error", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err = w.Write(dataBytes); err != nil {
		slog.Error("writeJSON", "error", err)
		return
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	reqID := middleware.GetReqID(r.Context())

	errResp := domain.ErrorResponse{
		Code:      code,
		Message:   message,
		RequestID: reqID,
	}

	msg := struct {
		Error domain.ErrorResponse `json:"error"`
	}{
		Error: errResp,
	}

	writeJSON(w, status, msg)
}

func writeValidationError(w http.ResponseWriter, r *http.Request, details []domain.ErrorDetail) {
	reqID := middleware.GetReqID(r.Context())
	errResp := domain.ErrorResponse{
		Code:      "VALIDATION ERROR",
		Message:   "Invalid Input",
		Details:   details,
		RequestID: reqID,
	}

	msg := struct {
		Error domain.ErrorResponse `json:"error"`
	}{
		Error: errResp,
	}

	writeJSON(w, http.StatusUnprocessableEntity, msg)
}

func decode(r *http.Request, v interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return err
	}

	return nil
}

func getUserID(ctx context.Context) string {
	return ctx.Value(userIDKey).(string)
}
