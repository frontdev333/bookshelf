package handler

import (
	"bookshelf/auth-svc/internal/service"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type InternalHandler struct {
	svc *service.UserService
}

type VerifyResponse struct {
	Valid     bool      `json:"valid"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	Error     string    `json:"error"`
}

func NewInternalHandler(svc *service.UserService) *InternalHandler {
	return &InternalHandler{
		svc: svc,
	}
}

func (h *InternalHandler) VerifyToken(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var token struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&token); err != nil {
		slog.Error("InternalHandler.VerifyToekn() json.NewDecoder()", "error", err)
		writeJSON(
			w,
			http.StatusBadRequest,
			VerifyResponse{
				Valid: false,
				Error: err.Error(),
			},
		)
		return
	}

	claims, err := h.svc.ValidateToken(token.Token)
	if err != nil {
		slog.Error("InternalHandler.VerifyToekn() h.UService.ValidateToken", "error", err)
		writeJSON(
			w,
			http.StatusUnauthorized,
			VerifyResponse{
				Valid: false,
				Error: err.Error(),
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		VerifyResponse{
			Valid:     true,
			UserID:    claims.Subject,
			ExpiresAt: claims.ExpiresAt.Time,
			Error:     "",
		},
	)
	return
}

func (h *InternalHandler) GetUsersByIDs(w http.ResponseWriter, r *http.Request) {
	var dto struct {
		IDs []string `json:"ids"`
	}

	body := r.Body
	_ = body

	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		slog.Error("InternalHandler.GetUserByIDs", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if dto.IDs == nil {
		writeError(w, r, http.StatusBadRequest, "IDs_FORMAT_ERROR", "wrong ids format")
	}

	users, err := h.svc.GetUsersByIDs(r.Context(), dto.IDs)
	if err != nil {
		slog.Error("InternalHandler.GetUserByIDs", "error", err)
		writeError(w, r, http.StatusInternalServerError, "SERVER_ERROR", "error try later")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(users); err != nil {
		slog.Error("InternalHandler.GetUsersByIDs", "error", err)
	}
}
