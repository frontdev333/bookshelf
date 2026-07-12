package handler

import (
	"bookshelf/books-service/internal/domain"
	"bookshelf/books-service/internal/service"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (h *BookHandler) List(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	sort := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")

	f := domain.BookFilter{
		Search: &search,
		Sort:   &sort,
		Order:  &order,
		Page:   &page,
		Limit:  &limit,
	}

	list, err := h.svc.List(r.Context(), f)
	if err != nil {
		if errors.Is(err, service.ErrInvalidPagination) {
			writeError(w, r, http.StatusBadRequest, "INVALID_PAGINATION", "page and limit must be positive integers")
			return
		}

		if errors.Is(err, service.ErrCreatorLookupError) {
			slog.Error("book_handler.ListBook()", "error", err)
			writeError(w, r, http.StatusBadGateway, "AUTH_SERVICE_ERROR", "failed to load book creator")
			return
		}

		slog.Error("book_handler.ListBook()", "error", err)
		writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "unknown error")
		return
	}

	writeJSON(w, http.StatusOK, list)
}

func (h *BookHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	bookId := chi.URLParam(r, "book_id")

	b, err := h.svc.GetByID(r.Context(), bookId)
	if err != nil {
		if errors.Is(err, service.ErrBookNotFound) {
			writeError(w, r, http.StatusNotFound, "BOOK_NOT_FOUND", "book not found")
			return
		}
		if errors.Is(err, service.ErrCreatorLookupError) {
			slog.Error("book_handler.GetBook()", "error", err)
			writeError(w, r, http.StatusBadGateway, "AUTH_SERVICE_ERROR", "failed to load book creator")
			return
		}
		slog.Error("book_handler.GetBook()", "error", err)
		writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "unknown error")
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (h *BookHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r.Context())
	var req domain.CreateBookRequest

	if err := decode(r, &req); err != nil {
		slog.Error("book_handler.CreateBook()", "error", err)
		writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON in request body")
		return
	}

	errDetails := make([]domain.ErrorDetail, 0)

	if strings.TrimSpace(req.Title) == "" {
		errDetails = append(errDetails, domain.ErrorDetail{
			Field:   "Title",
			Message: "Title is required",
		})
	}

	if strings.TrimSpace(req.Author) == "" {
		errDetails = append(errDetails, domain.ErrorDetail{
			Field:   "Author",
			Message: "Author is required",
		})
	}

	if len(errDetails) > 0 {
		writeValidationError(w, r, errDetails)
		return
	}

	b, err := h.svc.Create(r.Context(), userID, req)
	if err != nil {
		if handled := writeBookValidationError(w, r, err); handled {
			return
		}

		if errors.Is(err, service.ErrCreatorLookupError) {
			slog.Error("book_handler.CreateBook()", "error", err)
			writeError(w, r, http.StatusBadGateway, "AUTH_SERVICE_ERROR", "failed to load book creator")
			return
		}

		slog.Error("book_handler.CreateBook()", "error", err)
		writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "unknown error")
		return
	}

	writeJSON(w, http.StatusCreated, b)
}

func (h *BookHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r.Context())
	bookID := chi.URLParam(r, "book_id")

	var req domain.UpdateBookRequest

	if err := decode(r, &req); err != nil {
		slog.Error("book_handler.UpdateBook()", "error", err)
		writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON in request body")
		return
	}

	b, err := h.svc.Update(r.Context(), userID, bookID, req)
	if err != nil {
		if errors.Is(err, service.ErrNotBookOwner) {
			writeError(w, r, http.StatusForbidden, "NOT_BOOK_OWNER", "you are not the book owner")
			return
		}

		if errors.Is(err, service.ErrBookNotFound) {
			writeError(w, r, http.StatusNotFound, "BOOK_NOT_FOUND", "book not found")
			return
		}

		if handled := writeBookValidationError(w, r, err); handled {
			return
		}

		if errors.Is(err, service.ErrCreatorLookupError) {
			slog.Error("book_handler.UpdateBook()", "error", err)
			writeError(w, r, http.StatusBadGateway, "AUTH_SERVICE_ERROR", "failed to load book creator")
			return
		}

		slog.Error("book_handler.UpdateBook()", "error", err)
		writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "unknown error")
		return
	}

	writeJSON(w, http.StatusOK, b)
}

func (h *BookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r.Context())
	bookID := chi.URLParam(r, "book_id")

	if err := h.svc.Delete(r.Context(), userID, bookID); err != nil {
		if errors.Is(err, service.ErrNotBookOwner) {
			writeError(w, r, http.StatusForbidden, "NOT_BOOK_OWNER", "you are not the book owner")
			return
		}

		if errors.Is(err, service.ErrBookNotFound) {
			writeError(w, r, http.StatusNotFound, "BOOK_NOT_FOUND", "book not found")
			return
		}

		slog.Error("book_handler.DeleteBook()", "error", err)
		writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "unknown error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeBookValidationError(w http.ResponseWriter, r *http.Request, err error) bool {
	switch {
	case errors.Is(err, service.ErrBookTitleEmpty):
		writeValidationError(w, r, []domain.ErrorDetail{{
			Field:   "Title",
			Message: "Title is required",
		}})
		return true
	case errors.Is(err, service.ErrBookAuthorEmpty):
		writeValidationError(w, r, []domain.ErrorDetail{{
			Field:   "Author",
			Message: "Author is required",
		}})
		return true
	default:
		return false
	}
}
