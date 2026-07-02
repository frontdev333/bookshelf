package middleware

import (
	"bookshelf/books-service/internal/client"
	"bookshelf/books-service/internal/handler"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

func AuthMiddleware(authClient *client.AuthClient) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			errDto := struct {
				Error string `json:"error"`
			}{}

			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok {
				errDto.Error = "missing authorization header"
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(errDto)
				return
			}

			vResp, err := authClient.VerifyToken(r.Context(), token)
			if err != nil {
				slog.Error("AuthMiddleware", "error", err)
				errDto.Error = err.Error()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(errDto)
				return
			}

			ctx := context.WithValue(r.Context(), handler.UserIDKey, vResp.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
