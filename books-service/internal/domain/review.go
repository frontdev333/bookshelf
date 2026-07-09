package domain

import (
	"bookshelf/books-service/internal/client"
	"database/sql"
	"time"
)

type Review struct {
	ID        string         `db:"id"`
	BookID    string         `db:"book_id"`
	UserID    string         `db:"user_id"`
	Rating    int            `db:"rating"`
	Title     sql.NullString `db:"title"`
	Content   string         `db:"content"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
}

type ReviewResponse struct {
	ID        string             `json:"id"`
	BookID    string             `json:"book_id"`
	UserID    string             `json:"user_id"`
	User      client.UserSummary `json:"user"`
	Rating    int                `json:"rating"`
	Title     *string            `json:"title"`
	Content   string             `json:"content"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

type ReviewListResponse struct {
	Data       []ReviewResponse `json:"data"`
	Pagination Pagination       `json:"pagination"`
}

type CreateReviewRequest struct {
	Rating  int     `json:"rating"`
	Content string  `json:"content"`
	Title   *string `json:"title"`
}

type UpdateReviewRequest struct {
	Rating  *int    `json:"rating"`
	Content *string `json:"content"`
	Title   *string `json:"title"`
}

func (r *Review) ToResponse(user client.UserSummary) *ReviewResponse {

	var title *string
	title = nil

	if r.Title.Valid {
		title = &r.Title.String
	}

	return &ReviewResponse{
		ID:        r.ID,
		BookID:    r.BookID,
		UserID:    r.UserID,
		User:      user,
		Rating:    r.Rating,
		Title:     title,
		Content:   r.Content,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}
